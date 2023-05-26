package removeStrategies

import (
	"github.com/giobart/Active-Internal-Queue/pkg/element"
	"strconv"
	"testing"
)

func TestFifo_InvalidStrategy(t *testing.T) {
	_, err := RemoveStrategySelector(Custom)
	if err == nil {
		t.Error("Strategy not declared, it should throw an error")
	}
}

func TestCleanOldest_FindVictim(t *testing.T) {
	queue := make([]*element.Element, 20)
	cleanStrategy, _ := RemoveStrategySelector(CleanOldest)

	// create a full queue
	for i := 0; i < 20; i++ {
		queue[i] = &element.Element{
			Client:               strconv.Itoa(i),
			Id:                   strconv.Itoa(i),
			QoS:                  0,
			ThresholdRequirement: element.Threshold{},
			Timestamp:            int64(1000 - i),
			Data:                 []byte("test"),
		}
	}

	//set element 5 as the oldest
	queue[5].Timestamp = 1

	//check if FindVictim picks element 5
	index, err := cleanStrategy.FindVictim(&queue)

	if err != nil {
		t.Fatal(err)
	}

	if index != 5 {
		t.Fatal("Expected index: 5, found:", index)
	}
}
