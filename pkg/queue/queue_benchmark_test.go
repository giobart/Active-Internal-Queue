package queue

import (
	"encoding/json"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
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
				Client:               strconv.Itoa(i),
				Id:                   strconv.Itoa(i),
				QoS:                  0,
				ThresholdRequirement: element.Threshold{},
				Timestamp:            0,
				Data:                 []byte("test"),
			})
			if err != nil {
				t.Fatal(err)
			}
			if i%300 == 0 {
				analytics := myQueue.GetAnalytics()
				println("#### Benchmark iteration n,", i, " #####")
				println("Avg Queue Time: (ns)", analytics.AvgPermanenceTime)
				println("EnqueueDequeue Ratio: ", analytics.EnqueueDequeueRatio)
				println("Space left: %", analytics.SpaceFull)
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

	begin := time.Now().UnixNano()
	go dequeueThread()
	go enqueueThread()
	<-finished
	end := time.Now().UnixNano()

	analytics := myQueue.GetAnalytics()

	println("#### Benchmark Results ####")
	println("Avg Queue Time: (ns)", analytics.AvgPermanenceTime)
	println("EnqueueDequeue Ratio: ", analytics.EnqueueDequeueRatio)
	println("Final Space left: %", analytics.SpaceFull)
	println("Total time: (ns)", end-begin)
}

func Benchmark30Fps(t *testing.B) {
	dequeueNext := make(chan bool)
	finished := make(chan bool)

	deqFunc := func(el *element.Element) {
		dequeueNext <- true
	}

	myQueue, _ := New(deqFunc, OptionSetAnalyticsService(100), OptionQueueLength(1000))

	enqueueThread := func() {
		for i := 0; i < 300; i++ {
			err := myQueue.Enqueue(element.Element{
				Client:               strconv.Itoa(i),
				Id:                   strconv.Itoa(i),
				QoS:                  0,
				ThresholdRequirement: element.Threshold{},
				Timestamp:            0,
				Data:                 []byte("test"),
			})
			if err != nil {
				t.Fatal(err)
			}
			if i%100 == 0 {
				analytics := myQueue.GetAnalytics()
				println("#### Benchmark iteration n,", i, " #####")
				println("Avg Queue Time: (ns)", analytics.AvgPermanenceTime)
				println("EnqueueDequeue Ratio: ", analytics.EnqueueDequeueRatio)
				println("Space occupied: %", analytics.SpaceFull)
			}
			time.Sleep(time.Millisecond * 30)
		}
	}

	dequeueThread := func() {
		for i := 0; i < 300; i++ {
			myQueue.Dequeue()
			<-dequeueNext
		}
		finished <- true
	}

	begin := time.Now().UnixNano()
	go dequeueThread()
	go enqueueThread()
	<-finished
	end := time.Now().UnixNano()

	analytics := myQueue.GetAnalytics()

	println("#### Benchmark Results ####")
	println("Avg Queue Time: (ns)", analytics.AvgPermanenceTime)
	println("EnqueueDequeue Ratio: ", analytics.EnqueueDequeueRatio)
	println("Final Space occupied: %", analytics.SpaceFull)
	println("Total time: (ns)", end-begin)
}

func Benchmark30FpsUdpSockets(t *testing.B) {
	finished := make(chan bool)

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
	defer externalAppListener.Close()
	externalUdpSocketApplication := func() {

		b := make([]byte, 7000)
		for {
			n, _, err := externalAppListener.ReadFromUDP(b)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = externalAppconn.Write(b[:n])
		}
	}

	//dequeue function sending the frames to the external application
	deqFunc := func(el *element.Element) {
		conn, err := net.Dial("udp", "0.0.0.0:50505")
		if err != nil {
			t.Fatal(err)
			return
		}
		msg, err := json.Marshal(el)
		_, err = conn.Write(msg)
		if err != nil {
			t.Fatal(err)
		}
		conn.Close()
	}

	myQueue, _ := New(deqFunc, OptionSetAnalyticsService(100), OptionQueueLength(1000))

	//thread that generates 30 frames per second with size 4KiB for the queue
	enqueueThread := func() {
		time.Sleep(time.Millisecond * 100)
		for i := 0; i < 300; i++ {
			data := make([]byte, 4096) //4096 KiB long data
			rand.Read(data)
			err := myQueue.Enqueue(element.Element{
				Client:               strconv.Itoa(i),
				Id:                   strconv.Itoa(i),
				QoS:                  0,
				ThresholdRequirement: element.Threshold{},
				Timestamp:            0,
				Data:                 data,
			})
			if err != nil {
				t.Fatal(err)
			}
			if i%100 == 0 {
				analytics := myQueue.GetAnalytics()
				println("#### Benchmark iteration n,", i, " #####")
				println("Avg Queue Time: (ns)", analytics.AvgPermanenceTime)
				println("EnqueueDequeue Ratio: ", analytics.EnqueueDequeueRatio)
				println("Space occupied: %", analytics.SpaceFull)
			}
			time.Sleep(time.Millisecond * 1) //1000FPS
		}
	}

	//thread that sends the frames to the external applications and awaits for the response
	// after each response forwards the next frame
	dequeueThread := func() {
		l, err := net.ListenUDP("udp", &net.UDPAddr{
			Port: 50506,
			IP:   net.ParseIP("0.0.0.0"),
		})
		defer l.Close()
		if err != nil {
			t.Fatal(err)
		}
		b := make([]byte, 7000)
		for i := 0; i < 300; i++ {
			myQueue.Dequeue()
			n, _, err := l.ReadFromUDP(b)
			if err != nil {
				t.Fatal(err)
			}
			var msg element.Element
			if err = json.Unmarshal(b[:n], &msg); err != nil {
				t.Fatal(err)
			}
		}
		finished <- true
	}

	begin := time.Now().UnixNano()
	go dequeueThread()
	go enqueueThread()
	go externalUdpSocketApplication()
	<-finished
	end := time.Now().UnixNano()

	analytics := myQueue.GetAnalytics()

	println("#### Benchmark Results ####")
	println("Avg Queue Time: (ns)", analytics.AvgPermanenceTime)
	println("EnqueueDequeue Ratio: ", analytics.EnqueueDequeueRatio)
	println("Final Space occupied: %", analytics.SpaceFull)
	println("Total time: (ns)", end-begin)
}
