package removeStrategies

import (
	"errors"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
)

type RemoveStrategy int

const (
	CleanOldest RemoveStrategy = iota
	Custom      RemoveStrategy = iota
)

var RemoveStrategyMap = map[RemoveStrategy]RemoveStrategyActuator{
	CleanOldest: &cleanOldest{},
}

type RemoveStrategyActuator interface {
	FindVictim(queue *[]*element.Element) (int, error)
	New() RemoveStrategyActuator
}

func RemoveStrategySelector(s RemoveStrategy) (RemoveStrategyActuator, error) {
	if actuator, exist := RemoveStrategyMap[s]; exist && actuator != nil {
		return actuator.New(), nil
	}
	return nil, errors.New("strategy not supported")
}
