package queue

import (
	"sync"
	"time"
)

type Analytics struct {
	AvgPermanenceTime        int64   `json:"AvgPermanenceTime"`   //expressed in milliseconds
	EnqueueDequeueRatio      int     `json:"EnqueueDequeueRatio"` //<1 dequeue faster than enqueue, ==1 balanced, >1 enqueue faster than dequeue
	SpaceFull                int     `json:"SpaceFull"`           //0-100% value expressing the space lect
	ThresholdRatio           float64 `json:"ThresholdRatio"`      //0-1 value representing (N.Thresholded values/Tot number of deletion)
	EPSIn                    float64 `json:"ElementPerSecondIn"`
	EPSOut                   float64 `json:"ElementPerSecondOut"`
	ePSInCounter             int
	ePSOutCounter            int
	ePSInLock                sync.RWMutex
	ePSOutLock               sync.RWMutex
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
	analyticsService := &Analytics{
		AvgPermanenceTime:   0,
		EnqueueDequeueRatio: 0,
		SpaceFull:           0,
		deltaInsertionTime:  make([]int64, windowSize),
		deltaPermanenceTime: make([]int64, windowSize),
		maxQueueSize:        0,
		windowSize:          windowSize,
	}
	go analyticsService.countEPS()
	return analyticsService
}

func (a *Analytics) countEPS() {

	for true {
		select {
		case <-time.After(time.Second):
			a.ePSInLock.Lock()
			a.ePSOutLock.Lock()
			a.EPSIn = float64(a.EPSIn)*0.9 + float64(a.ePSInCounter)*0.1
			a.EPSOut = float64(a.EPSOut)*0.9 + float64(a.ePSOutCounter)*0.1
			a.ePSInCounter = 0
			a.ePSOutCounter = 0
			a.ePSInLock.Unlock()
			a.ePSOutLock.Unlock()
		}
	}

}

// NotifyInsertion used no
func (a *Analytics) NotifyInsertion() {
	//update insertion time window
	now := time.Now().UnixNano()
	if a.lastInsertionTime != 0 {
		a.deltaInsertionTime[a.nextInsertionTimeIndex] = now - a.lastInsertionTime
		a.nextInsertionTimeIndex = (a.nextInsertionTimeIndex + 1) % len(a.deltaInsertionTime)
	}
	a.lastInsertionTime = now
	//update EPS counter
	a.ePSInLock.Lock()
	a.ePSInCounter += 1
	a.ePSInLock.Unlock()
}

// NotifyDeletion used to notify that an item was removed from the queue.
// It requires the former insertion time to keep track of the avg permanence time in the queue
func (a *Analytics) NotifyDeletion(previousInsertionTime int64) {
	//update permanence time window
	deltaPermanenceTime := time.Now().UnixNano() - previousInsertionTime
	a.deltaPermanenceTime[a.nextDeltaPermanenceIndex] = deltaPermanenceTime
	a.nextDeltaPermanenceIndex = (a.nextDeltaPermanenceIndex + 1) % len(a.deltaPermanenceTime)
	a.addDeleted()
	//update EPS counter
	a.ePSOutLock.Lock()
	a.ePSOutCounter += 1
	a.ePSOutLock.Unlock()
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
	if a.totDeleted == 0 {
		a.ThresholdRatio = 0
	} else {
		a.ThresholdRatio = float64(float64(a.totThreshold) / float64(a.totDeleted))
	}

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
