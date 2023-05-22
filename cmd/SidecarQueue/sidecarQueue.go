package main

import (
	"context"
	"fmt"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
	"github.com/giobart/Active-Internal-Queue/pkg/gRPCspec"
	"github.com/giobart/Active-Internal-Queue/pkg/queue"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"time"
)

func main() {

	quit := make(chan bool, 0)
	generatedQueue := make(chan *queue.ActiveInternalQueue, 0)
	framesChan := make(chan element.Element, 5)
	go startQueueClient(quit, generatedQueue, 50505, framesChan)
	queue := <-generatedQueue
	go ProcessFrame(framesChan, (*queue).Dequeue)

	//start forwarding server
	//start incoming frames server

}

func startQueueClient(quit <-chan bool, generatedQueue chan<- *queue.ActiveInternalQueue, port int, forwardChan chan element.Element) {
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

	generatedQueue <- &myQueue
	<-quit
	cancel()
}

func ProcessFrame(frames chan element.Element, dequeue func()) {
	for true {
		frame := <-frames
		dequeue()
		ForwardFrame(frame)
	}
}
