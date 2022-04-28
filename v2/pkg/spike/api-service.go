package spike

import (
	"time"

	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/broker/providers/testProvider"
	"github.com/spike-events/spike-broker/v2/pkg/service"
)

// Options holds all attributes needed for Spike APIService to initialize
type Options struct {
	// Service holds the service.APIService implementantion that will be served
	Service service.Service

	// Broker holds the broker.Provider that will be used for communication
	Broker broker.Provider

	// Repository holds the service.Repository implementation that will be used by the service.APIService
	Repository service.Repository

	// Logger holds the service.Logger that will handle the logging calls
	Logger service.Logger

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
	// Initialize the APIService with all options
	Initialize(options Options) error

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

// NewTest returns an APIService implementation that runs unit tests based on specified parameters
func NewTest(startRepository service.Repository, startRequestMocks testProvider.RequestMock) *testServiceImpl {
	if serviceTestImplInstance == nil {
		serviceTestImplInstance = &testServiceImpl{
			startRepository:   startRepository,
			startRequestMocks: startRequestMocks,
		}
	}
	return serviceTestImplInstance
}
