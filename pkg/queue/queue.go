package queue

import (
	"github.com/giobart/Active-Internal-Queue/pkg/element"
	"github.com/giobart/Active-Internal-Queue/pkg/strategies"
)

const (
	DefaultLength     = 20
	DefaultMinDequeue = 1
	DefaultMaxDequeue = 1
	DefaultNWorkers   = 1
)

type Queue struct {
	queue          []*element.Element
	insertStrategy strategies.Strategy
	removeStrategy strategies.Strategy
	length         int
	minDequeue     int
	nWorkers       int
	maxDequeue     int
	dequeueFunc    func(el element.Element, batchId int)
}

type ActiveInternalQueue interface {
	Enqueue(el element.Element) error
	Finished(batchId int) error
}

func OptionQueueLength(length int) func(*Queue) error {
	return func(q *Queue) error {
		q.queue = make([]*element.Element, length)
		q.length = length
		return nil
	}
}

func OptionQueueStrategy(strategy strategies.Strategy) func(*Queue) error {
	return func(q *Queue) error {
		q.queueStrategy = strategy
		return nil
	}
}

func OptionMinDequeue(minDequeue int) func(queue *Queue) error {
	return func(q *Queue) error {
		q.minDequeue = minDequeue
		return nil
	}
}

func OptionMaxDequeue(maxDequeue int) func(queue *Queue) error {
	return func(q *Queue) error {
		q.maxDequeue = maxDequeue
		return nil
	}
}

func OptionConcurrentWorkers(n int) func(queue *Queue) error {
	return func(q *Queue) error {
		q.nWorkers = n
		return nil
	}
}

func New(dequeueFunc func(el element.Element, batchId int), options ...func(*Queue) error) (ActiveInternalQueue, error) {
	queue := Queue{
		queue:         make([]*element.Element, DefaultLength),
		queueStrategy: strategies.FIFO,
		length:        DefaultLength,
		minDequeue:    DefaultMinDequeue,
		nWorkers:      DefaultNWorkers,
		maxDequeue:    DefaultMaxDequeue,
		dequeueFunc:   dequeueFunc,
	}

	//parsing functional arguments
	for _, op := range options {
		err := op(&queue)
		if err != nil {
			return nil, err
		}
	}

	return &queue, nil
}

func (q *Queue) Enqueue(el element.Element) error {
	return nil
}

func (q *Queue) Finished(batchId int) error {
	return nil
}
