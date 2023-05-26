package queue

import (
	"time"
)

type Analytics struct {
	AvgPermanenceTime        int64   //expressed in milliseconds
	EnqueueDequeueRatio      int     //<1 dequeue faster than enqueue, ==1 balanced, >1 enqueue faster than dequeue
	SpaceFull                int     //0-100% value expressing the space lect
	ThresholdRatio           float64 //0-1 value representing (N.Thresholded values/Tot number of deletion)
	deltaInsertionTime       []int64
	deltaPermanenceTime      []int64
	avgDeltaInsertionTime    int64
	lastInsertionTime        int64
	nextDeltaPermanenceIndex int
	nextInsertionTimeIndex   int
	totDeleted               int
	totThreshold             int
	maxQueueSize             int
	windowSize               int
}

type AnalyticsGenerator interface {
	NotifyInsertion()
	NotifyDeletion(previousInsertionTime int64)
	NotifyThreshold()
	NotifyCurrentSpace(numberOfStoredElements int, maxSize int)
	GetAnalytics() Analytics
}

// NewAnalyticsGenerator create new analytics structure with window size
func NewAnalyticsGenerator(windowSize int) *Analytics {
	return &Analytics{
		AvgPermanenceTime:   0,
		EnqueueDequeueRatio: 0,
		SpaceFull:           0,
		deltaInsertionTime:  make([]int64, windowSize),
		deltaPermanenceTime: make([]int64, windowSize),
		maxQueueSize:        0,
		windowSize:          windowSize,
	}
}

// NotifyInsertion used no
func (a *Analytics) NotifyInsertion() {
	now := time.Now().UnixNano()
	if a.lastInsertionTime != 0 {
		a.deltaInsertionTime[a.nextInsertionTimeIndex] = now - a.lastInsertionTime
		a.nextInsertionTimeIndex = (a.nextInsertionTimeIndex + 1) % len(a.deltaInsertionTime)
	}
	a.lastInsertionTime = now
}

// NotifyDeletion used to notify that an item was removed from the queue.
// It requires the former insertion time to keep track of the avg permanence time in the queue
func (a *Analytics) NotifyDeletion(previousInsertionTime int64) {
	deltaPermanenceTime := time.Now().UnixNano() - previousInsertionTime
	a.deltaPermanenceTime[a.nextDeltaPermanenceIndex] = deltaPermanenceTime
	a.nextDeltaPermanenceIndex = (a.nextDeltaPermanenceIndex + 1) % len(a.deltaPermanenceTime)
	a.addDeleted()
}

func (a *Analytics) NotifyCurrentSpace(numberOfStoredElements int, maxSize int) {
	a.SpaceFull = int(numberOfStoredElements * 100 / maxSize)
}

// GetAnalytics calculate the analytics and returns the results
func (a *Analytics) GetAnalytics() Analytics {
	a.AvgPermanenceTime = 0
	a.EnqueueDequeueRatio = 0
	a.avgDeltaInsertionTime = 0

	// calculate avg permanence time
	a.AvgPermanenceTime = calcAvg(&a.deltaPermanenceTime)

	// calc avg insertion time
	a.avgDeltaInsertionTime = calcAvg(&a.deltaInsertionTime)

	// calc queuedequeueratio
	if a.avgDeltaInsertionTime == 0 {
		a.EnqueueDequeueRatio = 1
	} else {
		a.EnqueueDequeueRatio = int(a.AvgPermanenceTime / a.avgDeltaInsertionTime)
	}

	//calc threshold
	a.ThresholdRatio = float64(float64(a.totThreshold) / float64(a.totDeleted))

	return *a
}

func calcAvg(arr *[]int64) int64 {
	// calculate avg permanence time
	avgItems := 0
	avg := int64(0)
	for _, time := range *arr {
		if time != 0 {
			avg += time
			avgItems++
		}
	}
	if avgItems > 0 {
		avg = avg / int64(avgItems)
	}
	return avg
}

func (a *Analytics) addDeleted() {
	a.totDeleted++
	if a.totDeleted > a.windowSize {
		a.totDeleted = a.totDeleted / 2
		a.totThreshold = a.totThreshold / 2
	}
}

func (a *Analytics) NotifyThreshold() {
	a.totThreshold++
}
