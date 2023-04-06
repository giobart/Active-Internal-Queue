package removeStrategies

import (
	"errors"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
)

type cleanOldest struct{}

func (c *cleanOldest) FindVictim(queue *[]*element.Element) (int, error) {
	oldest := -1
	oldestTimestamp := int64(0)
	for i := 0; i < len(*queue); i++ {
		if oldest == -1 {
			oldest = i
			oldestTimestamp = (*queue)[i].Timestamp
		}
		if (*queue)[i].Timestamp < oldestTimestamp {
			oldest = i
			oldestTimestamp = (*queue)[i].Timestamp
		}
	}
	if oldest == -1 {
		return -1, errors.New("not found")
	}
	return oldest, nil
}
