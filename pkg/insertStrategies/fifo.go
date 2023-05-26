package insertStrategies

import (
	"errors"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
)

type fifo struct {
	start int
	end   int
	count int
}

func (f *fifo) New() PushPopStrategyActuator {
	return &fifo{start: 0, end: 0}
}

func (f *fifo) Push(el *element.Element, queue *[]*element.Element) error {
	size := len(*queue)
	if f.end == f.start && f.count == size {
		return &FullQueue{}
	}
	(*queue)[f.end] = el
	f.end = (f.end + 1) % size
	f.count++
	return nil
}

func (f *fifo) Pop(queue *[]*element.Element) (*element.Element, error) {
	size := len(*queue)
	if f.start == f.end && f.count == 0 {
		return nil, &EmptyQueue{}
	}
	retvalue := (*queue)[f.start]
	(*queue)[f.start] = nil
	f.start = (f.start + 1) % size
	f.count--
	return retvalue, nil
}

func (f *fifo) Delete(index int, queue *[]*element.Element) error {
	size := len(*queue)
	if index >= size {
		return errors.New("out of bound")
	}
	if index >= f.start || index <= f.end {
		(*queue)[index] = nil

		for i := 0; i <= f.count; i++ {
			tmp, err := f.Pop(queue)
			if err != nil {
				return err
			}
			if tmp != nil {
				_ = f.Push(tmp, queue)
			}
		}

		return nil
	} else {
		return errors.New("out of range")
	}
}
