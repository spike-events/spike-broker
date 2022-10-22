package broker

import (
	"encoding/json"
	"time"

	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

type ProviderType string

// Subscription has the Resource, the Call handler and the optional Access validators
type Subscription struct {
	Resource   rids.Pattern
	Handler    CallHandler
	Validators []AccessHandler
}

type ServiceHandler func(p rids.Pattern, payload []byte, replyEndpoint string)

// Provider interface implements a multiservice communication broker that allows to listen and execute requests to the
// any Service announced on Spike network
type Provider interface {
	// SetHandler sets the Spike handler for arriving messages on specific Service
	SetHandler(service string, handler ServiceHandler)

	// Close ends the connection to the Provider
	Close()

	// Subscribe requests the Provider to handle a rids.Resource balancing the requests
	Subscribe(s Subscription) (interface{}, error)

	// SubscribeAll requests the Provider to handle a rids.Resource at all times
	SubscribeAll(s Subscription) (interface{}, error)

	// Monitor informs the Provider how to handle a rids.Resource event
	Monitor(monitoringGroup string, s Subscription) (func(), error)

	// Get calls a rids.Resource through the Provider without a paylod
	Get(p rids.Pattern, rs interface{}, token ...string) Error

	// Request calls a rids.Resource through the Provider passing a payload
	Request(p rids.Pattern, payload interface{}, rs interface{}, token ...string) Error

	// RequestRaw calls a low level subject with a json.RawMessage payload and an optional timeout
	RequestRaw(subject string, data json.RawMessage, overrideTimeout ...time.Duration) (json.RawMessage, Error)

	// Publish informs the Provider that a rids.Resource event has happened
	Publish(p rids.Pattern, payload interface{}, token ...string) error

	// PublishRaw publishes a low-level event with a json.RawMessage on a subject
	PublishRaw(subject string, data json.RawMessage) error
}
