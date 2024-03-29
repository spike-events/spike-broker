package spike

import (
	"time"

	"github.com/spike-events/spike-broker/v2/pkg/service"
)

// Options holds all attributes needed for Spike APIService to initialize
type Options struct {
	// Service holds the service.APIService implementantion that will be served
	Service service.Service

	// Authenticator implements the authenticator interface that will be used by the service to check token validity
	Authenticator service.Authenticator

	// Authorizer implements the authorizer interface that will verify if the token can access the route
	Authorizer service.Authorizer

	// Timeout is the default timeout used internally
	Timeout time.Duration
}

// APIService interface for starting and stopping the service.Service instance. It is defined as an interface to allow the
// creation of a TestBroker implementation which in turns allow the service.Service to be unit tested
type APIService interface {
	// Deprecated: RegisterService registers the Service with Spike API.
	// Use Setup because RegisterService seems to give the impression one can register multiple services
	RegisterService(options Options) error

	// Setup sets the API Service Options before starting
	Setup(options Options) error

	// StartService starts the service specified on provided Options
	StartService() error

	// Stop requests all services to close connections and HTTP server to stop in case it was stated
	Stop() error
}

// NewAPIService returns an APIService implementation that starts an HTTP and WebSocket routing, authentication and authorization
// mechanism using
func NewAPIService() APIService {
	return &serviceImpl{}
}
