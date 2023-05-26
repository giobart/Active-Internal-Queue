package main

import (
	"fmt"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
	"github.com/giobart/Active-Internal-Queue/pkg/insertStrategies"
	"github.com/giobart/Active-Internal-Queue/pkg/queue"
	"github.com/giobart/Active-Internal-Queue/pkg/removeStrategies"
	"log"
)

func main() {

	finish := make(chan bool)

	dequeueCallback := func(el *element.Element) {
		//DO SOMETHING HERE WITH THE MESSAGE
		finish <- true
	}

	myQueue, _ := queue.New(
		dequeueCallback,
		queue.OptionQueueLength(20),
		queue.OptionQueueRemoveStrategy(removeStrategies.CleanOldest),
		queue.OptionQueueInsertStrategy(insertStrategies.FIFO),
	)

	//QUEUE 10 ELEMENTS
	for i := 0; i < 10; i++ {
		err := myQueue.Enqueue(element.Element{
			Id:   fmt.Sprintf("%d", i),
			Data: []byte("Hello World"),
			ThresholdRequirement: element.Threshold{
				Type:      element.MaxLatency,
				Threshold: 200, //latency in ms
				Current:   0,   //You can initialize the current cumulative latency or leave it empty
			},
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	//ASK DEQUEUE 5 TIMES
	for i := 0; i < 5; i++ {
		myQueue.Dequeue()
	}

	//Wait until the callback is called 5 times
	for i := 0; i < 5; i++ {
		<-finish
	}
}
