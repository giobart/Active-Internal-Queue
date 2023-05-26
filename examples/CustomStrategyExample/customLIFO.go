package main

import (
	"errors"
	"fmt"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
	"github.com/giobart/Active-Internal-Queue/pkg/insertStrategies"
	"github.com/giobart/Active-Internal-Queue/pkg/queue"
	"github.com/giobart/Active-Internal-Queue/pkg/removeStrategies"
	"log"
)

type myCustomLIFOStrategy struct {
	totElements int
}

func (f *myCustomLIFOStrategy) New() insertStrategies.PushPopStrategyActuator {
	return &myCustomLIFOStrategy{}
}

func (f *myCustomLIFOStrategy) Push(el *element.Element, queue *[]*element.Element) error {
	size := len(*queue)
	if f.totElements == size {
		return &insertStrategies.FullQueue{}
	}
	(*queue)[f.totElements] = el
	f.totElements++
	return nil
}

func (f *myCustomLIFOStrategy) Pop(queue *[]*element.Element) (*element.Element, error) {
	if f.totElements == 0 {
		return nil, &insertStrategies.EmptyQueue{}
	}
	f.totElements--
	retvalue := (*queue)[f.totElements]
	(*queue)[f.totElements] = nil
	return retvalue, nil
}

func (f *myCustomLIFOStrategy) Delete(index int, queue *[]*element.Element) error {
	size := len(*queue)
	if index >= size {
		return errors.New("out of bound")
	}
	if index >= 0 || index <= f.totElements {
		(*queue)[index] = nil
		f.totElements--
		for i, el := range (*queue)[index:] {
			if el != nil {
				(*queue)[index+i-1] = el
			}
		}
	} else {
		return errors.New("out of range")
	}
	return nil
}

func main() {

	finish := make(chan bool)

	dequeueCallback := func(el *element.Element) {
		//DO SOMETHING HERE WITH THE MESSAGE
		log.Println(string(el.Data))
		finish <- true
	}

	insertStrategies.InsertStrategyMap[insertStrategies.Custom] = &myCustomLIFOStrategy{}

	myQueue, _ := queue.New(
		dequeueCallback,
		queue.OptionQueueLength(5),
		queue.OptionQueueRemoveStrategy(removeStrategies.CleanOldest),
		queue.OptionQueueInsertStrategy(insertStrategies.Custom),
	)

	//QUEUE 10 ELEMENTS
	for i := 0; i < 10; i++ {
		err := myQueue.Enqueue(element.Element{
			Id:   fmt.Sprintf("%d", i),
			Data: []byte(fmt.Sprintf("Element %d", i)),
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
