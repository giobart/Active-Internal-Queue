package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/giobart/Active-Internal-Queue/cmd/SidecarQueue/requests"
	"github.com/giobart/Active-Internal-Queue/cmd/SidecarQueue/streamgRPCspec"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
	"github.com/giobart/Active-Internal-Queue/pkg/gRPCspec"
	"github.com/giobart/Active-Internal-Queue/pkg/queue"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var QueueService queue.ActiveInternalQueue
var BUFFER_SIZE = 64 * 1024

var NextService = flag.String("next", "", "Address of the next service in the pipeline")
var ExternalPort = flag.String("p", "50506", "port that will be exposed to receive the frames")
var SidecarAddress = flag.String("sidecar", "localhost:50505", "address of the sidecar service")
var isEntrypoint = flag.Bool("entry", false, "If True, this is an entrypoint, and frames will be received from the UDP socket")
var isExitpoint = flag.Bool("exit", false, "If True, this is an exitpoint, frames will be sent back to the client using the client address, if NextService is specified, the frames will be forwarded there as well")
var thershold = flag.Int("ms", 200, "Threshold in milliseconds, number of milliseconds after which a frame is considered obsolete and is discarded")
var analyticsTimer = flag.Float64("analytics", 0, "How often the analytics service will gather the information. The value refers to How many seconds to wait between one query and another. ")
var monitor = flag.String("monitor", "", "External monitor service for Application Aware Orchestration. Only works if service collects analytics.")
var debug = flag.Bool("debug", false, "Debug mode")
var parallelOutStream = flag.Int("parallel", 1, "Number of parallel output streams")
var qsize = flag.Int("qsize", 5, "Queue Size")

type StreamServer struct {
	streamgRPCspec.UnimplementedFramesStreamServiceServer
}

func main() {

	flag.Parse()
	quit := make(chan bool, 0)
	generatedQueue := make(chan queue.ActiveInternalQueue, 0)
	framesChan := make(chan element.Element, 2)
	//starting queue sidecar
	go startQueueClient(quit, generatedQueue, framesChan)
	QueueService = <-generatedQueue
	//starting processing and forwarding gRPC client
	go ProcessOutgoingFrames(framesChan, QueueService.Dequeue)
	if *isEntrypoint {
		//if this service is an entrypoint, then receive the frames directly from the client using UDP
		go ReceiveUDPFrameRoutine()
	} else {
		//starting receive gRPC routine
		go ReceiveFrameGrpcRoutine()
	}

	//starting analytics routine
	go collectAnalytics(QueueService)

	QueueService.Dequeue()
	//blocking until SIGINT or SIGTERM
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Blocking, press ctrl+c to kill the sidecar...")
	<-done // Will block here until user hits ctrl+c

}

// ### Analytics
func collectAnalytics(queue queue.ActiveInternalQueue) {
	if *analyticsTimer == 0 {
		return
	}
	timerDuration := time.Second * time.Duration(*analyticsTimer)
	// collect analytics
	for true {
		select {
		case <-time.After(timerDuration):
			analytics := queue.GetAnalytics()
			jsonanalytics, err := json.Marshal(analytics)
			if err == nil {
				body := string(jsonanalytics)
				log.Printf("%d;%s;%s;%s;\n", time.Now().UnixMilli(), "QUEUE", "analytics", fmt.Sprintf("{analytics:%s}", body))
				// If monitor service configured, forward the analytics
				if *monitor != "" {
					log.Println("Forwarding analytics to monitor service.")
					go requests.SendAnalytics(body, *monitor)
				}
			}
		}
	}
}

// ### Queue Service ####

func startQueueClient(quit <-chan bool, generatedQueue chan<- queue.ActiveInternalQueue, forwardChan chan element.Element) {
	log.Println("Waiting for sidecar service connection...", *SidecarAddress)
	clientConn, err := grpc.Dial(*SidecarAddress, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	log.Println("Sidecar CONNECTED! Initializing queue...")

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	gRPCclient := gRPCspec.NewQueueServiceClient(clientConn)

	deqFunc := func(el *element.Element) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		result := element.Element{
			Client:               el.Client,
			Id:                   el.Id,
			QoS:                  0,
			ThresholdRequirement: el.ThresholdRequirement,
			Timestamp:            0,
			Data:                 el.Data,
		}
		r, err := gRPCclient.NextFrame(ctx, &gRPCspec.Frame{
			Client: el.Client,
			Id:     el.Id,
			Qos:    "0",
			Data:   el.Data,
		})
		if err != nil {
			log.Println("Unable to process Next Frame: ", err)
			<-quit
		}
		result.Data = r.Data
		forwardChan <- result
		cancel()
	}

	myQueue, _ := queue.New(
		deqFunc,
		queue.OptionSetAnalyticsService(50),
		queue.OptionQueueLength(*qsize),
	)

	generatedQueue <- myQueue
	<-quit

}

// ### Forward frames to next service and/or output service####
func ProcessOutgoingFrames(frames chan element.Element, dequeue func()) {
	sendFramesChan := make(chan element.Element, 10)
	clientSendFramesChan := make(chan element.Element, 10)
	if *isExitpoint {
		// if this is the expitpoint the frames must be sent to the UDP socket instead of the gRPC channel for the next service
		go SendFramesToClientRoutine(clientSendFramesChan)
	}
	if *NextService != "" {
		for i := 0; i < *parallelOutStream; i++ {
			go SendFrameGrpcRoutine(*NextService, sendFramesChan)
		}
	}
	for true {
		frame := <-frames
		dequeue()
		if len(clientSendFramesChan) < cap(clientSendFramesChan) {
			clientSendFramesChan <- frame
		}
		if len(sendFramesChan) < cap(sendFramesChan) {
			sendFramesChan <- frame
		}
	}
}

func SendFramesToClientRoutine(frames chan element.Element) {
	connectionBuffer := make(map[string]*net.UDPConn, 100)
	for true {
		frame := <-frames
		connection, exist := connectionBuffer[frame.Client]
		if !exist || connection == nil {
			raddr, err := net.ResolveUDPAddr("udp", frame.Client)
			if err != nil {
				log.Println("Unable to send udp packet")
				continue
			}
			connection, err = net.DialUDP("udp", nil, raddr)
			if err != nil {
				log.Println("Unable to send udp packet")
				continue
			}
			connectionBuffer[frame.Client] = connection
		}
		_, _, err := (*connection).WriteMsgUDP(frame.Data, nil, nil)
		if err != nil {
			connectionBuffer[frame.Client] = nil
			log.Println("Unable to complete udp write to packet to ", frame.Client)
			continue
		}
	}
}

func SendFrameGrpcRoutine(nextService string, frames chan element.Element) {
	//in case of failure just reboot the function in a new goroutine and try to connect again to the next service
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer func() {
		cancel()
		go SendFrameGrpcRoutine(nextService, frames)
	}()
	stream, err := NextServiceConnect(nextService, ctx)
	if err != nil {
		log.Println("Unable to connect to next service: ", nextService)
		return
	}
	i_max := 300
	for i := 0; i < i_max; i++ {
		frame := <-frames
		if *debug {
			log.Println("DEBUG: {", frame.Client, frame.Id, frame.QoS, frame.Data, "}")
		}
		err := stream.Send(&streamgRPCspec.StreamFrame{
			Client: frame.Client,
			Id:     frame.Id,
			Qos:    "",
			Data:   frame.Data,
			Threshold: &streamgRPCspec.StreamFrameThreshold{
				Type:      "ms",
				Current:   float32(frame.ThresholdRequirement.Current),
				Threshold: float32(frame.ThresholdRequirement.Threshold),
			},
		})
		if err != nil || i == (i_max-1) {
			_ = stream.CloseSend()
			log.Println("Stream Closed ", nextService, err)
			return
		}
	}
}

func NextServiceConnect(nextService string, ctx context.Context) (streamgRPCspec.FramesStreamService_StreamFramesClient, error) {
	clientConn, err := grpc.DialContext(
		ctx,
		nextService,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithReturnConnectionError(),
		//grpc.WithInitialWindowSize(1024*1024*200),
	)
	gRPCclient := streamgRPCspec.NewFramesStreamServiceClient(clientConn)
	var stream streamgRPCspec.FramesStreamService_StreamFramesClient = nil
	if err == nil {
		stream, err = gRPCclient.StreamFrames(ctx)
	}
	return stream, err
}

// ### Frames Receive Server ####

func ReceiveFrameGrpcRoutine() {

	restart := func() {
		go ReceiveFrameGrpcRoutine()
	}
	port, err := strconv.Atoi(*ExternalPort)
	if err != nil {
		log.Fatalf("%v", err)
	}
	serverListener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer() //grpc.MaxRecvMsgSize(100*102*1024), grpc.MaxSendMsgSize(1024*1024*100)) //100MB max size
	streamgRPCspec.RegisterFramesStreamServiceServer(s, StreamServer{})
	if err := s.Serve(serverListener); err != nil {
		log.Println("gRPC server error")
		log.Println(err)
		return
	}
	defer serverListener.Close()
	defer restart()
}

func (s StreamServer) StreamFrames(stream streamgRPCspec.FramesStreamService_StreamFramesServer) error {
	for true {
		nextFrame, err := stream.Recv()
		if err != nil {
			log.Println(err)
			return err
		}
		if *debug {
			log.Println("DEBUG: {", nextFrame.Client, nextFrame.Id, nextFrame.Qos, nextFrame.Threshold.Current, "}")
		}
		if nextFrame.Threshold.Current >= nextFrame.Threshold.Threshold {
			log.Println("DEBUG: { thresholded frame }")
			continue
		}
		err = QueueService.Enqueue(element.Element{
			Client:    nextFrame.Client,
			Id:        nextFrame.Id,
			QoS:       0,
			Timestamp: 0,
			ThresholdRequirement: element.Threshold{
				Type:      element.MaxLatency,
				Threshold: float64(nextFrame.Threshold.GetThreshold()),
				Current:   float64(nextFrame.Threshold.GetCurrent()),
			},
			Data: nextFrame.Data,
		})
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

func ReceiveUDPFrameRoutine() {
	buffer := make([]byte, BUFFER_SIZE)
	port, err := strconv.Atoi(*ExternalPort)
	if err != nil {
		log.Fatal(err)
	}
	udpListenAddr := net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: port,
	}
	conn, err := net.ListenUDP("udp", &udpListenAddr)
	if err != nil {
		log.Fatal(err)
	}
	for true {
		packet := buffer
		n, from, err := conn.ReadFromUDP(packet)
		if err != nil {
			log.Println("Invalid upd message received")
			continue
		}
		data := make([]byte, n)
		copy(data, buffer[:n])
		frame := element.Element{
			Client: from.String(),
			Id:     "1",
			QoS:    0,
			ThresholdRequirement: element.Threshold{
				Type:      element.MaxLatency,
				Threshold: float64(*thershold),
				Current:   0,
			},
			Data: data,
		}
		err = QueueService.Enqueue(frame)
		if err != nil {
			log.Println("Impossible to queue the element. Queue Error")
			log.Println(err)
			continue
		}
	}
}
