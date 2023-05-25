package queue

import (
	"github.com/giobart/Active-Internal-Queue/pkg/element"
	"strconv"
	"testing"
	"time"
)

func TestQueue_Enqueue(t *testing.T) {
	deqFunc := func(el *element.Element) {
		print("Works")
	}
	myQueue, err := New(deqFunc)
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

	myQueue, _ := New(deqFunc, OptionQueueLength(100))

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
}
