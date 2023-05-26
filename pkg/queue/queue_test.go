package queue

import (
	"fmt"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
	"github.com/giobart/Active-Internal-Queue/pkg/insertStrategies"
	"github.com/giobart/Active-Internal-Queue/pkg/removeStrategies"
	"log"
	"strconv"
	"testing"
	"time"
)

func TestQueue_Enqueue(t *testing.T) {
	deqFunc := func(el *element.Element) {
		print("Works")
	}
	myQueue, err := New(
		deqFunc,
		OptionQueueInsertStrategy(insertStrategies.FIFO),
		OptionQueueRemoveStrategy(removeStrategies.CleanOldest),
		OptionConcurrentWorkers(1),
		OptionMaxDequeue(1),
		OptionMinDequeue(1),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = myQueue.Enqueue(element.Element{
		Client:               "a",
		Id:                   "1",
		QoS:                  0,
		ThresholdRequirement: element.Threshold{},
		Timestamp:            0,
		Data:                 []byte("test"),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestQueue_Enqueue_Full(t *testing.T) {
	deqFunc := func(el *element.Element) {
		print("Works")
	}
	myQueue, err := New(
		deqFunc,
		OptionQueueInsertStrategy(insertStrategies.FIFO),
		OptionQueueRemoveStrategy(removeStrategies.CleanOldest),
		OptionQueueLength(10),
	)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		err = myQueue.Enqueue(element.Element{
			Client:               "a",
			Id:                   "1",
			QoS:                  0,
			ThresholdRequirement: element.Threshold{},
			Timestamp:            0,
			Data:                 []byte("test"),
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestQueue_Dequeue(t *testing.T) {
	dequeueNext := make(chan bool)
	finished := make(chan bool)
	last := -1
	deqFunc := func(el *element.Element) {
		intId, _ := strconv.Atoi(el.Id)
		if intId-1 != last {
			t.Fatal("The next id should be: ", last+1, " instead we have ", intId)
		}
		last = intId
		dequeueNext <- true
	}

	myQueue, _ := New(deqFunc)

	enqueueThread := func() {
		for i := 0; i < 100; i++ {
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
			time.Sleep(time.Millisecond * 10)
		}
	}

	dequeueThread := func() {
		time.Sleep(time.Millisecond * 10)
		for i := 0; i < 100; i++ {
			myQueue.Dequeue()
			<-dequeueNext
		}
		finished <- true
	}
	go dequeueThread()
	go enqueueThread()
	<-finished
}

func TestQueue_ThresholdLatency(t *testing.T) {
	dequeueNext := make(chan int)
	finished := make(chan bool)
	last := -1
	deqFunc := func(el *element.Element) {
		intId, _ := strconv.Atoi(el.Id)
		if intId-2 != last {
			t.Fatal("The next id should be: ", last+2, " instead we have ", intId)
		}
		if intId%2 == 0 {
			t.Fatal("Found ID: ", intId, " But frames with even IDs should have been discarded because of latency req.")
		}
		last = intId
		dequeueNext <- intId
	}

	myQueue, _ := New(deqFunc, OptionQueueLength(100), OptionSetAnalyticsService(10))
	initialanalytics := myQueue.GetAnalytics()
	if initialanalytics.totDeleted != 0 && initialanalytics.SpaceFull != 0 {
		t.Error("Initial Analytics should be 0")
	}

	enqueueThread := func() {
		time.Sleep(time.Millisecond)
		for i := 0; i < 100; i++ {
			err := myQueue.Enqueue(element.Element{
				Client: strconv.Itoa(i),
				Id:     strconv.Itoa(i),
				QoS:    0,
				ThresholdRequirement: element.Threshold{
					Type:      element.MaxLatency,
					Current:   0,
					Threshold: float64((i % 2) * 1000),
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
		time.Sleep(time.Millisecond * 9)
		for i := 0; i < 50; i++ {
			myQueue.Dequeue()
			<-dequeueNext
		}
		finished <- true
	}
	go dequeueThread()
	go enqueueThread()
	<-finished
	analytics := myQueue.GetAnalytics()
	if analytics.ThresholdRatio != 0.5 {
		log.Fatalln("Threshold ratio should be 0.5, current ratio: ", analytics.ThresholdRatio)
	}
}

func TestQueue_SimpleExample(t *testing.T) {
	dequeueCallback := func(el *element.Element) {
		//DO SOMETHING HERE WITH THE MESSAGE
		fmt.Println(el)
	}

	myQueue, _ := New(dequeueCallback)

	//QUEUE 10 ELEMENTS
	for i := 0; i < 10; i++ {
		err := myQueue.Enqueue(element.Element{
			Id:   fmt.Sprintf("%d", i),
			Data: []byte("test"),
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	//ASK DEQUEUE 10 TIMES
	for i := 0; i < 10; i++ {
		myQueue.Dequeue()
	}
}

func TestQueue_DequeueFirst(t *testing.T) {

	finish := make(chan bool, 10)

	dequeueCallback := func(el *element.Element) {
		finish <- true
	}

	myQueue, _ := New(dequeueCallback)

	//ASK DEQUEUE 10 TIMES BEFORE ENQUEUE
	for i := 0; i < 10; i++ {
		myQueue.Dequeue()
	}

	//QUEUE 10 ELEMENTS
	for i := 0; i < 10; i++ {
		err := myQueue.Enqueue(element.Element{
			Id:   fmt.Sprintf("%d", i),
			Data: []byte("test"),
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	for i := 0; i < 10; i++ {
		<-finish
	}

}

func TestQueue_InvalidElement(t *testing.T) {

	dequeueCallback := func(el *element.Element) {}

	myQueue, _ := New(dequeueCallback)

	//QUEUE 10 ELEMENTS
	for i := 0; i < 10; i++ {
		err := myQueue.Enqueue(element.Element{
			Data: []byte("test"),
		})
		if err == nil {
			t.Error("Elements without ID should throw an error")
		}
	}

}

func TestQueue_InvalidInsertStrategyOption(t *testing.T) {
	deqFunc := func(el *element.Element) {
		print("Works")
	}
	_, err := New(
		deqFunc,
		OptionQueueInsertStrategy(insertStrategies.Custom),
		OptionQueueRemoveStrategy(removeStrategies.CleanOldest),
		OptionConcurrentWorkers(1),
		OptionMaxDequeue(1),
		OptionMinDequeue(1),
	)
	if err == nil {
		t.Error("Cuatom strategy not defined, it should return an error")
	}
}

func TestQueue_InvalidRemoveStrategyOption(t *testing.T) {
	deqFunc := func(el *element.Element) {
		print("Works")
	}
	_, err := New(
		deqFunc,
		OptionQueueInsertStrategy(insertStrategies.FIFO),
		OptionQueueRemoveStrategy(removeStrategies.Custom),
		OptionConcurrentWorkers(1),
		OptionMaxDequeue(1),
		OptionMinDequeue(1),
	)
	if err == nil {
		t.Error("Cuatom strategy not defined, it should return an error")
	}
}
