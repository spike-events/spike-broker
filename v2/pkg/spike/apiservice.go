package spike

import (
	"fmt"
	"time"

	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/broker/providers/testProvider"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/service"
)

// Options holds all attributes needed for Spike APIService to initialize
type Options struct {

	// Service holds the service.APIService implementantion that will be served
	Service service.Service

	// Provider holds the broker.Provider that will be used for communication
	Provider broker.Provider

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
	Start(options Options) error
	Stop(timeout time.Duration) error
}

// New returns an APIService implementation that starts an HTTP and WebSocket routing, authentication and authorization
// mechanism using
func New() APIService {
	return &serviceImpl{}
}

// NewTest returns an APIService implementation that runs unit tests based on specified parameters
func NewTest(startedRepository service.Repository, startedRequestMocks testProvider.RequestMock) *testServiceImpl {
	if serviceTestImplInstance == nil {
		serviceTestImplInstance = &testServiceImpl{
			startRepository:   startedRepository,
			startRequestMocks: startedRequestMocks,
		}
	}
	return serviceTestImplInstance
}

// handleRequest
func handleRequest(p rids.Pattern, msg broker.Call, access broker.Access, opts Options) {
	defer func() {
		if r := recover(); r != nil {
			var rErr broker.Error
			err, ok := r.(error)
			if !ok {
				rErr = broker.InternalError(fmt.Errorf("request: %s: %v", p.EndpointName(), r))
			} else {
				rErr = broker.InternalError(err)
			}
			opts.Logger.Printf("nats: panic on handler: %v", r)
			msg.Error(rErr)
		}
	}()

	// Check if handled
	var found bool
	var handler broker.Subscription
	for _, handler = range opts.Service.Handlers() {
		if handler.Resource.EndpointName() == p.EndpointName() {
			found = true
			break
		}
	}

	if !found {
		msg.NotFound()
		return
	}

	// Authenticate and Authorize
	if !p.Public() {
		// Test Token Authentication
		var valid bool
		var token string
		if token, valid = opts.Authenticator.ValidateToken(msg.RawToken()); !valid {
			msg.Error(broker.ErrorStatusUnauthorized)
			return
		}

		msg.SetToken(token)

		// Test Route Authorization
		if !opts.Authorizer.HasPermission(msg) {
			msg.Error(broker.ErrorStatusForbidden)
			return
		}
	}

	if len(handler.Validators) > 0 {
		for _, validator := range handler.Validators {
			validator(access)
			if err := access.GetError(); err != nil {
				msg.Error(err)
				return
			}
		}
	}

	handler.Handler(msg)
}
