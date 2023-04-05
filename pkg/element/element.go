package element

type ThresholdType int

const (
	MaxLatency      ThresholdType = iota
	MinimumAccuracy ThresholdType = iota
	None            ThresholdType = iota
)

type Threshold struct {
	Type      ThresholdType `json:"Type"`
	Threshold float64       `json:"Threshold"`
	Current   float64       `json:"Current"`
}

type Element struct {
	Client               string    `json:"Client"`
	Id                   string    `json:"Id"`
	QoS                  int       `json:"QoS"`
	ThresholdRequirement Threshold `json:"ThresholdRequirement"`
	Timestamp            int64     `json:"timestamp"`
	Data                 []byte    `json:"Data"`
}
