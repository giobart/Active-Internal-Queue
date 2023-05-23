package main

import (
	"context"
	"fmt"
	"github.com/giobart/Active-Internal-Queue/cmd/SidecarQueue/streamgRPCspec"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
	"github.com/giobart/Active-Internal-Queue/pkg/gRPCspec"
	"github.com/giobart/Active-Internal-Queue/pkg/queue"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"time"
)

var NextService = ""
var QueueService queue.ActiveInternalQueue

type StreamServer struct {
	streamgRPCspec.UnimplementedFramesStreamServiceServer
}

func main() {

	quit := make(chan bool, 0)
	generatedQueue := make(chan queue.ActiveInternalQueue, 0)
	framesChan := make(chan element.Element, 5)
	//starting queue sidecar
	go startQueueClient(quit, generatedQueue, 50505, framesChan)
	queue := <-generatedQueue
	//starting processing and forwarding gRPC client
	go ProcessFrame(framesChan, queue.Dequeue)
	//starting receive gRPC routine
	go ReceiveFrameGrpcRoutine()

}

// ### Queue Service ####

func startQueueClient(quit <-chan bool, generatedQueue chan<- queue.ActiveInternalQueue, port int, forwardChan chan element.Element) {
	target := fmt.Sprintf("localhost:%d", port)
	clientConn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	gRPCclient := gRPCspec.NewQueueServiceClient(clientConn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	deqFunc := func(el *element.Element) {
		r, err := gRPCclient.NextFrame(ctx, &gRPCspec.Frame{
			Client: el.Client,
			Id:     el.Id,
			Qos:    "0",
			Data:   el.Data,
		})
		if err != nil {
			log.Println("Unable to process Next Frame!")
		}
		forwardChan <- element.Element{
			Client:               r.Client,
			Id:                   r.Id,
			QoS:                  0,
			ThresholdRequirement: element.Threshold{},
			Timestamp:            0,
			Data:                 r.Data,
		}
	}

	myQueue, _ := queue.New(
		deqFunc,
		queue.OptionSetAnalyticsService(100),
		queue.OptionQueueLength(100),
	)

	generatedQueue <- myQueue
	<-quit
	cancel()
}

// ### Frame Forwarding Client ####

func ProcessFrame(frames chan element.Element, dequeue func()) {
	sendFramesChan := make(chan element.Element, 10)
	go SendFrameGrpcRoutine(NextService, sendFramesChan)
	for true {
		frame := <-frames
		dequeue()
		sendFramesChan <- frame
	}
}

func SendFrameGrpcRoutine(nextService string, frames chan element.Element) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	//in case of failure just reboot the function in a new goroutine and try to connect again to the next service
	defer func() {
		time.Sleep(time.Second)
		go SendFrameGrpcRoutine(nextService, frames)
	}()
	stream, err := NextServiceConnect(nextService, ctx)
	if err != nil {
		log.Println("Unable to connect to next service: ", nextService)
		return
	}
	for true {
		frame := <-frames
		err := stream.SendMsg(frame)
		if err != nil {
			log.Println("Unable to send frame to next service: ", nextService)
			return
		}
	}
}

func NextServiceConnect(nextService string, ctx context.Context) (streamgRPCspec.FramesStreamService_StreamFramesClient, error) {
	clientConn, err := grpc.Dial(nextService, grpc.WithTransportCredentials(insecure.NewCredentials()))
	gRPCclient := streamgRPCspec.NewFramesStreamServiceClient(clientConn)
	var stream streamgRPCspec.FramesStreamService_StreamFramesClient = nil
	if err == nil {
		stream, err = gRPCclient.StreamFrames(ctx, &streamgRPCspec.Frame{})
	}
	return stream, err
}

// ### Frames Receive Server ####

func ReceiveFrameGrpcRoutine() {

	restart := func() {
		go ReceiveFrameGrpcRoutine()
	}
	serverListener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 50505))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	streamgRPCspec.RegisterFramesStreamServiceServer(s, StreamServer{})
	if err := s.Serve(serverListener); err != nil {
		log.Println("gRPC server error")
		log.Println(err)
		return
	}
	defer serverListener.Close()
	defer restart()
}

func (s StreamServer) StreamFrames(frame *streamgRPCspec.Frame, stream streamgRPCspec.FramesStreamService_StreamFramesServer) error {
	for true {
		nextFrame := element.Element{}
		err := stream.RecvMsg(&nextFrame)
		if err != nil {
			return err
		}
		err = QueueService.Enqueue(nextFrame)
		if err != nil {
			return err
		}
	}
	return nil
}
