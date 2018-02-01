package forwarder

// Forwarder is the forwarder of the events
type Forwarder interface {
	start()
	Forward(event []byte, partition string) error
}
