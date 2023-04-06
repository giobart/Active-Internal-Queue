package queue

import (
	"errors"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
	"github.com/giobart/Active-Internal-Queue/pkg/insertStrategies"
	"github.com/giobart/Active-Internal-Queue/pkg/removeStrategies"
	"time"
)

const (
	DefaultLength     = 20
	DefaultMinDequeue = 1
	DefaultMaxDequeue = 1
	DefaultNWorkers   = 1
)

type Queue struct {
	queue          []*element.Element
	insertStrategy insertStrategies.PushPopStrategyActuator
	removeStrategy removeStrategies.RemoveStrategyActuator
	batch          int64
	length         int
	inserted       int
	minDequeue     int
	nWorkers       int
	maxDequeue     int
	waitChannel    chan bool
	dequeueFunc    func(el *element.Element)
}

type Analytics struct {
	AvgPermanenceTime   int64 //expressed in milliseconds
	EnqueueDequeueRatio int64 //<1 dequeue faster than enqueue, ==1 balanced, >1 enqueue faster than dequeue
	Spaceleft           int   //0-100% value expressing the space lect
}

type ActiveInternalQueue interface {
	// Enqueue Add an element to the queue
	Enqueue(el element.Element) error
	// Dequeue Remove from the queue the next frame or waits until there is one.
	// It calls "dequeueFunc" with the corresponding result
	Dequeue()
	// GetAnalytics obtain the internal queue analytics
	GetAnalytics() Analytics
}

func OptionQueueLength(length int) func(*Queue) error {
	return func(q *Queue) error {
		q.queue = make([]*element.Element, length)
		q.length = length
		return nil
	}
}

func OptionQueueInsertStrategy(strategy insertStrategies.InsertStrategy) func(*Queue) error {
	return func(q *Queue) error {
		selector, err := insertStrategies.InsertStrategySelector(strategy)
		if err != nil {
			return err
		}
		q.insertStrategy = selector
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

func New(dequeueFunc func(el *element.Element), options ...func(*Queue) error) (ActiveInternalQueue, error) {
	insertFifo, _ := insertStrategies.InsertStrategySelector(insertStrategies.FIFO)
	removeOldest, _ := removeStrategies.RemoveStrategySelector(removeStrategies.CleanOldest)
	queue := Queue{
		queue:          make([]*element.Element, DefaultLength),
		insertStrategy: insertFifo,
		removeStrategy: removeOldest,
		length:         DefaultLength,
		minDequeue:     DefaultMinDequeue,
		nWorkers:       DefaultNWorkers,
		maxDequeue:     DefaultMaxDequeue,
		dequeueFunc:    dequeueFunc,
		waitChannel:    make(chan bool),
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

	if err := checkElement(el); err != nil {
		return err
	}

	el.Timestamp = time.Now().UnixMilli()

	//push element to queue
	err := q.insertStrategy.Push(&el, &q.queue)
	if err != nil {
		//If queue is full, remove element accordingly to the remove strategy and try again
		if errors.Is(err, &insertStrategies.FullQueue{}) {
			victim, err := q.removeStrategy.FindVictim(&q.queue)
			if err != nil {
				return err
			}
			err = q.insertStrategy.Delete(victim, &q.queue)
			if err != nil {
				return err
			}
			q.inserted--
			return q.Enqueue(el)
		} else {
			return err
		}
	}
	q.inserted++
	return nil
}

func (q *Queue) Dequeue() {

	element, err := q.insertStrategy.Pop(&q.queue)
	if err != nil {

	}
	go q.dequeueFunc(element)
	return
}

func (q *Queue) GetAnalytics() Analytics {
	return Analytics{}
}

func checkElement(el element.Element) error {
	if el.Id == "" {
		return errors.New("empty element id")
	}
	return nil
}
