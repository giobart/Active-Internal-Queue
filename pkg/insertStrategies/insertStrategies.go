package insertStrategies

import (
	"errors"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
)

type InsertStrategy int

const (
	FIFO          InsertStrategy = iota
	LIFO          InsertStrategy = iota
	ThresholdSort InsertStrategy = iota
	QoSAware      InsertStrategy = iota
	Custom        InsertStrategy = iota
)

var InsertStrategyMap = map[InsertStrategy]PushPopStrategyActuator{
	FIFO: &fifo{},
}

type PushPopStrategyActuator interface {
	Push(el *element.Element, queue *[]*element.Element) error
	Pop(queue *[]*element.Element) (*element.Element, error)
	Delete(index int, queue *[]*element.Element) error
	New() PushPopStrategyActuator
}

func InsertStrategySelector(s InsertStrategy) (PushPopStrategyActuator, error) {
	if actuator, exist := InsertStrategyMap[s]; exist && actuator != nil {
		return actuator.New(), nil
	}
	return nil, errors.New("strategy not supported")
}
