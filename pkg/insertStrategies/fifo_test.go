package insertStrategies

import (
	"github.com/giobart/Active-Internal-Queue/pkg/element"
	"strconv"
	"testing"
)

func TestFifo_Push(t *testing.T) {
	queue := make([]*element.Element, 20)
	strategy, _ := InsertStrategySelector(FIFO)
	for i := 0; i < 21; i++ {
		// Adding elements
		currentElement := element.Element{
			Client:               strconv.Itoa(i),
			Id:                   strconv.Itoa(i),
			QoS:                  0,
			ThresholdRequirement: element.Threshold{},
			Timestamp:            0,
			Data:                 []byte("test"),
		}
		err := strategy.Push(&currentElement, &queue)

		// check if element in correct position
		if i < 20 {
			if queue[i].Id != strconv.Itoa(i) {
				t.Fatal("Element n", i, " should be ", currentElement, "instead we have ", queue[i])
			}
		}

		if err != nil {
			// an error whould happen for element 21, but not before
			if i < 20 {
				t.Error("The queue has size 20, it should not be full")
				t.Fatal(err)
			}
			return
		}
	}
	// no error for element 21 -> something is not working
	t.Fatal("The queue should have thrown an error before")
}

func TestFifo_Push2(t *testing.T) {
	queue := make([]*element.Element, 1)
	strategy, _ := InsertStrategySelector(FIFO)

	// adding two elements in a one element queue
	for i := 0; i < 2; i++ {

		currentElement := element.Element{
			Client:               strconv.Itoa(i),
			Id:                   strconv.Itoa(i),
			QoS:                  0,
			ThresholdRequirement: element.Threshold{},
			Timestamp:            0,
			Data:                 []byte("test"),
		}
		err := strategy.Push(&currentElement, &queue)
		if err != nil {
			if i == 0 {
				t.Error("The queue has size 1, it should not be full")
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatal("The queue should have thrown an error before")
}

func TestFifo_Pop(t *testing.T) {
	queue := make([]*element.Element, 20)
	strategy, _ := InsertStrategySelector(FIFO)

	// cheking many times if push and pop break something over the time
	for j := 0; j < 21; j++ {
		for i := 0; i < 20; i++ {
			currentElement := element.Element{
				Client:               strconv.Itoa(i),
				Id:                   strconv.Itoa(i),
				QoS:                  0,
				ThresholdRequirement: element.Threshold{},
				Timestamp:            0,
				Data:                 []byte("test"),
			}
			err := strategy.Push(&currentElement, &queue)
			if err != nil {
				t.Fatal(err)
			}
		}
		for i := 0; i < 20; i++ {
			currentElement, err := strategy.Pop(&queue)
			if err != nil {
				t.Fatal(err)
			}
			if currentElement.Id != strconv.Itoa(i) {
				t.Fatal("Element n", i, " should be ", currentElement, "instead we have ", queue[i])
			}
		}
	}

}

func TestFifo_Delete(t *testing.T) {
	queue := make([]*element.Element, 20)
	strategy, _ := InsertStrategySelector(FIFO)

	//adding 20 elements to the queue of size 20
	for i := 0; i < 20; i++ {
		currentElement := element.Element{
			Client:               strconv.Itoa(i),
			Id:                   strconv.Itoa(i),
			QoS:                  0,
			ThresholdRequirement: element.Threshold{},
			Timestamp:            0,
			Data:                 []byte("test"),
		}
		err := strategy.Push(&currentElement, &queue)
		if err != nil {
			t.Error("The queue has size 20, it should not be full")
			t.Fatal(err)
		}
	}

	//removing element 0
	err := strategy.Delete(0, &queue)
	if err != nil {
		t.Error(err, "Should not be an error")
		t.Fatal(err)
	}

	//push new element with id 100
	newElement := element.Element{
		Client:               "100",
		Id:                   "100",
		QoS:                  0,
		ThresholdRequirement: element.Threshold{},
		Timestamp:            0,
		Data:                 []byte("test"),
	}
	err = strategy.Push(&newElement, &queue)
	if err != nil {
		t.Error("We deleted an element it should not be full")
		t.Fatal(err)
	}

	//the first element in the fifo should be 1
	result, err := strategy.Pop(&queue)
	if err != nil {
		t.Error("Should not be empty")
		t.Fatal(err)
	}

	if result.Id != "1" {
		t.Error("First element should be 1, current element is: ", result.Id)
		t.Fatal(err)
	}

}

func TestFifo_Delete2(t *testing.T) {
	queue := make([]*element.Element, 20)
	strategy, _ := InsertStrategySelector(FIFO)
	for i := 0; i < 21; i++ {
		currentElement := element.Element{
			Client:               strconv.Itoa(i),
			Id:                   strconv.Itoa(i),
			QoS:                  0,
			ThresholdRequirement: element.Threshold{},
			Timestamp:            0,
			Data:                 []byte("test"),
		}
		err := strategy.Push(&currentElement, &queue)
		if err != nil {
			if i < 20 {
				t.Error("The queue has size 20, it should not be full")
				t.Fatal(err)
			}
		}
	}

	//removing element 5 with ID==5
	err := strategy.Delete(5, &queue)
	if err != nil {
		t.Error(err, "Should not be an error")
		t.Fatal(err)
	}

	//removing element 5 with ID==6
	err = strategy.Delete(5, &queue)
	if err != nil {
		t.Error(err, "Should not be an error")
		t.Fatal(err)
	}

	newElement := element.Element{
		Client:               "100",
		Id:                   "100",
		QoS:                  0,
		ThresholdRequirement: element.Threshold{},
		Timestamp:            0,
		Data:                 []byte("test"),
	}
	err = strategy.Push(&newElement, &queue)
	if err != nil {
		t.Error("We deleted an element it should not be full")
		t.Fatal(err)
	}
	result, err := strategy.Pop(&queue)
	if err != nil {
		t.Error("Should not be empty")
		t.Fatal(err)
	}

	if result.Id != "0" {
		t.Error("First element should be 1, current element is: ", result.Id)
		t.Fatal(err)
	}

	for i := 0; i < 18; i++ {
		result, err := strategy.Pop(&queue)
		if err != nil {
			t.Error("Should not be empty")
			t.Fatal(err)
		}
		if result.Id == "5" {
			t.Fatal("Element: ", result, " should not be there")
		}
		if result.Id == "6" {
			t.Fatal("Element: ", result, " should not be there")
		}
	}
	result, err = strategy.Pop(&queue)
	if err == nil {
		t.Fatal("Should be empty")
	}

}
