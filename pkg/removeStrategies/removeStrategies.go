package removeStrategies

import (
	"errors"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
)

type RemoveStrategy int

const (
	CleanOldest RemoveStrategy = iota
)

type RemoveStrategyActuator interface {
	FindVictim(queue *[]*element.Element) (int, error)
}

func RemoveStrategySelector(s RemoveStrategy) (RemoveStrategyActuator, error) {
	switch s {
	case CleanOldest:
		return &cleanOldest{}, nil
	}
	return nil, errors.New("strategy supported")
}
