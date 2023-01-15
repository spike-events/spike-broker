package broker

import (
	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

type ProviderType string

// Subscription has the Resource, the Call handler and the optional Access validators
type Subscription struct {
	Resource   rids.Pattern
	Handler    CallHandler
	Validators []AccessHandler
}

// Event is used to declare Service events and their validators
type Event struct {
	Resource          rids.Pattern
	PublishValidators []AccessHandler
	MonitorValidators []AccessHandler
}

type ServiceHandler func(sub Subscription, payload []byte, replyEndpoint string)

// Provider interface implements a multiservice communication broker that allows to listen and execute requests to
// any Service announced on Spike network
type Provider interface {
	// Close ends the connection to the Provider
	Close()

	// Subscribe requests the Provider to handle a rids.Resource balancing the requests returning unsubscribe function
	Subscribe(s Subscription, handler ServiceHandler) (func(), Error)

	// Monitor checks if a rids.Resource event can be subscribed by the specified token, subscribe returning unsubscribe function
	Monitor(monitoringGroup string, s Subscription, handler ServiceHandler, token ...[]byte) (func(), Error)

	// Get calls a rids.Resource through the Provider without a paylod
	Get(p rids.Pattern, rs interface{}, token ...[]byte) Error

	// Request calls a rids.Resource through the Provider passing a payload
	Request(p rids.Pattern, payload interface{}, rs interface{}, token ...[]byte) Error

	// Publish informs the Provider that a rids.Resource event has happened
	Publish(p rids.Pattern, payload interface{}, token ...[]byte) Error

	// Reply returns the response using a reply endpoint
	Reply(replyEndpoint string, payload []byte) Error
}
