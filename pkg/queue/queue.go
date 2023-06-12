package queue

import (
	"errors"
	"github.com/giobart/Active-Internal-Queue/pkg/element"
	"github.com/giobart/Active-Internal-Queue/pkg/insertStrategies"
	"github.com/giobart/Active-Internal-Queue/pkg/removeStrategies"
	"sync"
	"time"
)

const (
	DefaultLength     = 20
	DefaultMinDequeue = 1
	DefaultMaxDequeue = 1
	DefaultNWorkers   = 1
)

type Queue struct {
	queue              []*element.Element
	insertStrategy     insertStrategies.PushPopStrategyActuator
	removeStrategy     removeStrategies.RemoveStrategyActuator
	batch              int64
	length             int
	inserted           int
	minDequeue         int
	nWorkers           int
	maxDequeue         int
	dequeueWaitChannel chan *element.Element
	dequeueFunc        func(el *element.Element)
	rwlock             sync.Mutex
	daqueueWaiting     int //number of dequeue functions waiting to be executed
	analyticsService   AnalyticsGenerator
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

func OptionQueueRemoveStrategy(strategy removeStrategies.RemoveStrategy) func(*Queue) error {
	return func(q *Queue) error {
		selector, err := removeStrategies.RemoveStrategySelector(strategy)
		if err != nil {
			return err
		}
		q.removeStrategy = selector
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

func OptionSetAnalyticsService(windowSize int) func(queue *Queue) error {
	return func(q *Queue) error {
		q.analyticsService = NewAnalyticsGenerator(windowSize)
		return nil
	}
}

func New(dequeueFunc func(el *element.Element), options ...func(*Queue) error) (ActiveInternalQueue, error) {
	insertFifo, _ := insertStrategies.InsertStrategySelector(insertStrategies.FIFO)
	removeOldest, _ := removeStrategies.RemoveStrategySelector(removeStrategies.CleanOldest)
	queue := Queue{
		queue:              make([]*element.Element, DefaultLength),
		insertStrategy:     insertFifo,
		removeStrategy:     removeOldest,
		length:             DefaultLength,
		minDequeue:         DefaultMinDequeue,
		nWorkers:           DefaultNWorkers,
		maxDequeue:         DefaultMaxDequeue,
		dequeueFunc:        dequeueFunc,
		dequeueWaitChannel: make(chan *element.Element, 100),
		analyticsService:   nil,
	}

	//parsing functional arguments
	for _, op := range options {
		err := op(&queue)
		if err != nil {
			return nil, err
		}
	}

	//set an async goroutine to handle the dequeue function call to avoid blocking the thread
	go func() {
		for true {
			select {
			case returnElement := <-queue.dequeueWaitChannel:
				queue.dequeueFunc(returnElement)
			}
		}
	}()

	return &queue, nil
}

func (q *Queue) Enqueue(el element.Element) error {

	q.rwlock.Lock()
	defer q.rwlock.Unlock()

	if err := checkElement(el); err != nil {
		return err
	}

	el.Timestamp = time.Now().UnixNano()

	//If dequeue function is waiting then immediately return the item
	if q.daqueueWaiting > 0 {
		q.daqueueWaiting--
		q.callDequeueFunction(&el)
		if q.analyticsService != nil {
			//notify quick insertion deletion for analytics purposes
			q.analyticsService.NotifyInsertion()
			q.analyticsService.NotifyDeletion(el.Timestamp)
		}
		return nil
	}

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
			q.rwlock.Unlock()
			err = q.Enqueue(el)
			q.rwlock.Lock()
			return err
		} else {
			return err
		}
	}

	// update number of inserted elements
	q.inserted++

	//notify the analytics service
	if q.analyticsService != nil {
		q.analyticsService.NotifyInsertion()
		q.analyticsService.NotifyCurrentSpace(q.inserted, q.length)
	}

	return nil
}

func (q *Queue) Dequeue() {

	q.rwlock.Lock()
	defer q.rwlock.Unlock()

	// try to fetch the element
	returnElement, err := q.insertStrategy.Pop(&q.queue)
	if err != nil {
		// if no element available update the waiting queue. New elements will be returned immediately
		q.daqueueWaiting = q.daqueueWaiting + 1
		return
	}

	// update queue size
	q.inserted--

	// notify analytics
	if q.analyticsService != nil {
		q.analyticsService.NotifyCurrentSpace(q.inserted, q.length)
		q.analyticsService.NotifyDeletion(returnElement.Timestamp)
	}

	//if threshold requirements not met discard element
	isThresholdValid := checkAndUpdateThreshold(returnElement)
	if !isThresholdValid {
		q.analyticsService.NotifyThreshold()
		go q.Dequeue()
		return
	}

	// async call to dequeue function
	q.callDequeueFunction(returnElement)

	return
}

func (q *Queue) callDequeueFunction(el *element.Element) {
	q.dequeueWaitChannel <- el
}

func (q *Queue) GetAnalytics() Analytics {
	if q.analyticsService != nil {
		return q.analyticsService.GetAnalytics()
	}
	return Analytics{}
}

func checkElement(el element.Element) error {
	if el.Id == "" {
		return errors.New("empty element id")
	}
	return nil
}

func checkAndUpdateThreshold(el *element.Element) bool {
	if el.ThresholdRequirement.Type == element.MaxLatency {
		totLatency := el.ThresholdRequirement.Current + float64((time.Now().UnixNano()-el.Timestamp)/int64(time.Millisecond))
		el.ThresholdRequirement.Current = totLatency
		if totLatency > el.ThresholdRequirement.Threshold {
			return false
		}
	}
	return true
}
