package broker

import (
	"time"

	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

// SpecificProvider is the base interface that Provider implementations must implement. Do not use it directly on
// application level, it is used only inside Spike Broker

type SpecificProvider interface {
	//Close Implements the underlying close operations
	Close()

	// SubscribeRaw implements the underlying subscription operations. It is fundamental that the specific endpoint is
	// used. In this case, if the subscription resource has parameters they must be used instead of generic values
	SubscribeRaw(sub Subscription, group string, handler ServiceHandler) (func(), Error)

	// RequestRaw calls a low level subject with a []byte payload and an optional timeout expecting a payload as response
	RequestRaw(subject string, data []byte, overrideTimeout ...time.Duration) ([]byte, Error)

	// PublishRaw publishes a low-level event with a []byte payload on a subject
	PublishRaw(subject string, data []byte) Error

	// NewCall creates a Call to be used internally
	NewCall(p rids.Pattern, payload interface{}) Call
}
