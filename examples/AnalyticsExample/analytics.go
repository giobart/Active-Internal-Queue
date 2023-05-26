package main

import (
	"fmt"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
	"github.com/giobart/Active-Internal-Queue/pkg/queue"
	"log"
)

func main() {

	finish := make(chan bool)

	dequeueCallback := func(el *element.Element) {
		//DO SOMETHING HERE WITH THE MESSAGE
		finish <- true
	}

	myQueue, _ := queue.New(dequeueCallback, queue.OptionSetAnalyticsService(20))

	//QUEUE 10 ELEMENTS
	for i := 0; i < 10; i++ {
		err := myQueue.Enqueue(element.Element{
			Id:   fmt.Sprintf("%d", i),
			Data: []byte("Hello World"),
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

	analytics := myQueue.GetAnalytics()
	fmt.Printf("AvgPermanence: %dns\nSpaceUsage: %d%%\nEnqueueDeququeRatio: %d\n", analytics.AvgPermanenceTime, analytics.SpaceFull, analytics.EnqueueDequeueRatio)
}
