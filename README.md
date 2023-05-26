# AIQ-Active-Internal-Queue
A Simple Internal Queue library for Go to support latency sensitive tasks 


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
