# AIQ-Active-Internal-Queue
![Coverage](https://img.shields.io/badge/Coverage-92.2%25-brightgreen) [![Queue Tests](https://github.com/giobart/Active-Internal-Queue/actions/workflows/queue-tests.yml/badge.svg)](https://github.com/giobart/Active-Internal-Queue/actions/workflows/queue-tests.yml)

A Simple Go Library to define an Internal Queue for latency critical tasks.

![Arch](img/arch.jpg)


# HelloWorld Example

```go
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
		fmt.Println(string(el.Data[:]))
		finish <- true
	}

	myQueue, _ := queue.New(dequeueCallback)

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
}
```

In this example we initialized a queue with the callback function `dequeueCallback`.
Then we inserted 10 elements and asked for a dequeue 5 times. As a result the callback is called 5 times.

The result will look something like this:

```
Hello World
Hello World
Hello World
Hello World
Hello World
```

# Obtain Queue Analytics 

```
myQueue, _ := queue.New(dequeueCallback, queue.OptionSetAnalyticsService(20))
```

Set an optional analytics service with a custom window size (>1). The bigger the window size the more precise the results, but the more memory usage will be higher. 

```
analytics := myQueue.GetAnalytics()
fmt.Printf("AvgPermanence: %dns\n", analytics.AvgPermanenceTime)
fmt.Printf("SpaceUsage: %d%%\n", analytics.SpaceFull)
fmt.Printf("EnqueueDeququeRatio: %d\n", analytics.EnqueueDequeueRatio)
```

# Set Queue Size and insert/remove strategies

```go
myQueue, _ := queue.New(
    dequeueCallback,
    queue.OptionQueueLength(20),
    queue.OptionQueueRemoveStrategy(removeStrategies.CleanOldest),
    queue.OptionQueueInsertStrategy(insertStrategies.FIFO),
)
```

In this example we created a queue with size `20`, `FIFO` enqueue and dequeue strategy and `Clean Oldest` remove strategy.
* The queue size is the maximum length of the queue. 
* Insert Strategy is the strategy used to push and pop elements from the queue. 
* Remove strategy is the strategy used to DROP an element when trying to insert a new one on a full queue. 

# Set Data Threshold

You can set a threshold to the queue elements. 

```go
err := myQueue.Enqueue(element.Element{
			Id:   fmt.Sprintf("%d", i),
			Data: []byte("Hello World"),
			ThresholdRequirement: element.Threshold{
				Type:      element.MaxLatency,
				Threshold: 200, //latency in ms
				Current:   0, //initialize the current cumulative latency or leave it empty
			},
		})
```

In this case, frames where `(current)+(permanence time in the queue) > Threshold` will be discarded before callind the `dequeueCallback` function.

# Define a custom Insert Strategy

You can define a custom Insert strategy extending the interface `insertStrategies.PushPopStrategyActuator` defined as follows.

```go
type PushPopStrategyActuator interface {
	Push(el *element.Element, queue *[]*element.Element) error
	Pop(queue *[]*element.Element) (*element.Element, error)
	Delete(index int, queue *[]*element.Element) error
	New() PushPopStrategyActuator
}
```

Example:

```go

import (
	"errors"
	"fmt"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
	"github.com/giobart/Active-Internal-Queue/pkg/insertStrategies"
	"github.com/giobart/Active-Internal-Queue/pkg/queue"
	"github.com/giobart/Active-Internal-Queue/pkg/removeStrategies"
	"log"
)

type myCustomInsertStrategy struct {
	totElements int
}

func (f *myCustomInsertStrategy) New() insertStrategies.PushPopStrategyActuator {
	return &myCustomInsertStrategy{}
}

func (f *myCustomInsertStrategy) Push(el *element.Element, queue *[]*element.Element) error {
	...
}

func (f *myCustomInsertStrategy) Pop(queue *[]*element.Element) (*element.Element, error) {
	...
}

func (f *myCustomInsertStrategy) Delete(index int, queue *[]*element.Element) error {
	...
}

...

func main() {
	
	dequeueCallback := func(el *element.Element) {
		//DO SOMETHING HERE WITH THE MESSAGE
	}

	//Insert my custom defined strategy in the strategy selector
	insertStrategies.InsertStrategyMap[insertStrategies.Custom] = &myCustomLIFOStrategy{}

	//Initialize the queue with a custom insert strategy
	myQueue, _ := queue.New(
		dequeueCallback,
		queue.OptionQueueLength(5),
		queue.OptionQueueRemoveStrategy(removeStrategies.CleanOldest),
		queue.OptionQueueInsertStrategy(insertStrategies.Custom),
	)
	
}
```

A full example with a custom LIFO strategy is available inside `examples/CustomStrategyExample`.


