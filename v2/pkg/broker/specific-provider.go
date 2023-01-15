package broker

import (
	"time"
)

// SpecificProvider is the base interface that Provider implementations must implement. Do not use it directly on
// application level, it is used only inside Spike Broker

type SpecificProvider interface {
	//Close Implements the underlying close operations
	Close()

	// SubscribeRaw implements the underlying subscription operations
	SubscribeRaw(sub Subscription, group string, handler ServiceHandler) (func(), Error)

	// RequestRaw calls a low level subject with a []byte payload and an optional timeout expecting a payload as response
	RequestRaw(subject string, data []byte, overrideTimeout ...time.Duration) ([]byte, Error)

	// PublishRaw publishes a low-level event with a []byte on a subject
	PublishRaw(subject string, data []byte) Error
}
