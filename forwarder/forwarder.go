package forwarder

// Forwarder is the forwarder of the events
type Forwarder interface {
	Produce(topic string, message []byte) (int32, int64, error)
}
