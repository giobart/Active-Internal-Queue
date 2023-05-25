package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
	"github.com/giobart/Active-Internal-Queue/pkg/gRPCspec"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"
)

func BenchmarkMaxSpeed(t *testing.B) {
	dequeueNext := make(chan bool)
	finished := make(chan bool)

	deqFunc := func(el *element.Element) {
		dequeueNext <- true
	}

	myQueue, _ := New(deqFunc, OptionSetAnalyticsService(100), OptionQueueLength(1000))

	enqueueThread := func() {
		for i := 0; i < 1000; i++ {
			err := myQueue.Enqueue(element.Element{
				Client: strconv.Itoa(i),
				Id:     strconv.Itoa(i),
				QoS:    0,
				ThresholdRequirement: element.Threshold{
					Type:      element.MaxLatency,
					Current:   0,
					Threshold: 200,
				},
				Timestamp: 0,
				Data:      []byte("test"),
			})
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	dequeueThread := func() {
		for i := 0; i < 1000; i++ {
			myQueue.Dequeue()
			<-dequeueNext
		}
		finished <- true
	}

	t.ResetTimer()
	go dequeueThread()
	go enqueueThread()
	<-finished
}

func BenchmarkDequeue500gRPC(t *testing.B) {

	//thread that simulates an external application receiving and sending udp frames
	serverListener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 50505))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	externalgRPCApplication := func() {
		s := grpc.NewServer()
		gRPCspec.RegisterQueueServiceServer(s, &gRPCspec.QueueServer{})
		if err := s.Serve(serverListener); err != nil {
			return
		}
	}

	//dequeue function sending the frames to the external application
	elemChan := make(chan element.Element, 10)
	clientConn, err := grpc.Dial("localhost:50505", grpc.WithTransportCredentials(insecure.NewCredentials()))
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
			log.Fatalf("failure: %v", err)
		}
		elemChan <- element.Element{
			Client:               r.Client,
			Id:                   r.Id,
			QoS:                  0,
			ThresholdRequirement: el.ThresholdRequirement,
			Timestamp:            0,
			Data:                 r.Data,
		}
	}

	myQueue, _ := New(deqFunc, OptionSetAnalyticsService(100), OptionQueueLength(1000))

	//thread that generates frames per second with size 4KiB for the queue
	enqueueThread := func() {
		for i := 0; i < 500; i++ {
			data := make([]byte, 8192) //8192 KiB long data
			rand.Read(data)
			err := myQueue.Enqueue(element.Element{
				Client: strconv.Itoa(i),
				Id:     strconv.Itoa(i),
				QoS:    0,
				ThresholdRequirement: element.Threshold{
					Type:      element.MaxLatency,
					Current:   0,
					Threshold: 200,
				},
				Timestamp: 0,
				Data:      data,
			})
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	t.ResetTimer()
	begin := time.Now().UnixNano()
	go enqueueThread()
	go externalgRPCApplication()
	for i := 0; i < 500; i++ {
		myQueue.Dequeue()
		<-elemChan
	}
	clientConn.Close()
	serverListener.Close()
	cancel()
	end := time.Now().UnixNano()

	//analytics := myQueue.GetAnalytics()

	//println("Total time: (ns)", end-begin)
	//t.ReportMetric(float64(analytics.AvgPermanenceTime)/float64(t.N), "Avg-Queue-Time")
	//t.ReportMetric(float64(analytics.EnqueueDequeueRatio)/float64(t.N), "EnqueueDequeue-Ratio")
	//t.ReportMetric(float64(analytics.SpaceFull)/float64(t.N), "Space-occupied")
	t.ReportMetric(float64(end-begin)/float64(t.N), "Total-time")

}

func BenchmarkDequeue500UDP(t *testing.B) {

	//thread that simulates an external application receiving and sending udp frames
	externalAppListener, err := net.ListenUDP("udp", &net.UDPAddr{
		Port: 50505,
		IP:   net.ParseIP("0.0.0.0"),
	})
	if err != nil {
		t.Fatal(err)
		return
	}
	externalAppconn, err := net.Dial("udp", "0.0.0.0:50506")
	if err != nil {
		t.Fatal(err)
	}
	externalUdpSocketApplication := func() {
		b := make([]byte, 10000)
		for {
			n, _, err := externalAppListener.ReadFromUDP(b)
			if err != nil {
				//closed listernet, function finished
				return
			}
			_, _ = externalAppconn.Write(b[:n])
		}
	}

	//dequeue function sending the frames to the external application
	dequeueConn, err := net.Dial("udp", "0.0.0.0:50505")
	if err != nil {
		t.Fatal(err)
		return
	}
	deqFunc := func(el *element.Element) {
		msg, err := json.Marshal(el)
		toSend := len(msg)
		sent := 0
		for toSend > 0 {
			sendbuf := msg
			if toSend >= 5000 {
				sendbuf = msg[sent : sent+5000]
				sent = 5000
				toSend -= 5000
			} else {
				sendbuf = msg[sent:]
				sent += toSend
				toSend = 0
			}
			_, err = dequeueConn.Write(sendbuf)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	myQueue, _ := New(deqFunc, OptionSetAnalyticsService(100), OptionQueueLength(1000))

	//thread that generates frames per second with size 4KiB for the queue
	enqueueThread := func() {
		for i := 0; i < 500; i++ {
			data := make([]byte, 8192) //8192 KiB long data
			rand.Read(data)
			err := myQueue.Enqueue(element.Element{
				Client: strconv.Itoa(i),
				Id:     strconv.Itoa(i),
				QoS:    0,
				ThresholdRequirement: element.Threshold{
					Type:      element.MaxLatency,
					Current:   0,
					Threshold: 200,
				},
				Timestamp: 0,
				Data:      data,
			})
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	t.ResetTimer()
	begin := time.Now().UnixNano()
	go enqueueThread()
	go externalUdpSocketApplication()
	l, err := net.ListenUDP("udp", &net.UDPAddr{
		Port: 50506,
		IP:   net.ParseIP("0.0.0.0"),
	})
	defer l.Close()
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 10000)
	for i := 0; i < 500; i++ {
		tmpbuffer := make([]byte, 0)
		myQueue.Dequeue()
		totread := 0
		n, _, err := l.ReadFromUDP(b)
		tmpbuffer = append(tmpbuffer, b[:n]...)
		totread += n
		for n == 5000 {
			n, _, err = l.ReadFromUDP(b)
			tmpbuffer = append(tmpbuffer, b[:n]...)
			totread += n
		}
		if err != nil {
			t.Fatal(err)
		}
		var msg element.Element
		if err = json.Unmarshal(tmpbuffer[:totread], &msg); err != nil {
			t.Fatal(err)
		}
	}
	dequeueConn.Close()
	externalAppListener.Close()
	externalAppconn.Close()
	end := time.Now().UnixNano()

	//analytics := myQueue.GetAnalytics()

	//println("Total time: (ns)", end-begin)
	//t.ReportMetric(float64(analytics.AvgPermanenceTime)/float64(t.N), "Avg-Queue-Time")
	//t.ReportMetric(float64(analytics.EnqueueDequeueRatio)/float64(t.N), "EnqueueDequeue-Ratio")
	//t.ReportMetric(float64(analytics.SpaceFull)/float64(t.N), "Space-occupied")
	t.ReportMetric(float64(end-begin)/float64(t.N), "Total-time")

}
