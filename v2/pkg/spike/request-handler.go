package spike

import (
	"fmt"

	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

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
			opts.Service.Logger().Printf("nats: panic on handler: %v", r)
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
