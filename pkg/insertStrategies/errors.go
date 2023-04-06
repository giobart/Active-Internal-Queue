package insertStrategies

type FullQueue struct{}

func (m *FullQueue) Error() string {
	return "Full Queue"
}

type EmptyQueue struct{}

func (m *EmptyQueue) Error() string {
	return "Empty Queue"
}
