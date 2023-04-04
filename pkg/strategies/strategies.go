package strategies

import (
	"errors"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
)

type Strategy int

const (
	FIFO          Strategy = iota
	LIFO          Strategy = iota
	ThresholdSort Strategy = iota
	QoSAware      Strategy = iota
)

type PushPopStrategyActuator interface {
	Push(el *element.Element, queue *[]*element.Element) error
	Pop(queue *[]*element.Element) (*element.Element, error)
	Delete(index int, queue *[]*element.Element) error
}

type CleanStrategyActuator interface {
	NextIndexToClean(queue *[]*element.Element) error
}

func StrategySelector(s Strategy) (PushPopStrategyActuator, error) {
	switch s {
	case FIFO:
		return &fifo{start: 0, end: 0}, nil
	case LIFO:
		return nil, errors.New("not yet implemented")
	case ThresholdSort:
		return nil, errors.New("not yet implemented")
	case QoSAware:
		return nil, errors.New("not yet implemented")
	}
	return nil, errors.New("strategy supported")
}
