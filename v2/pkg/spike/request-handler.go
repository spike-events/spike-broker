package spike

import (
	"encoding/json"
	"fmt"

	"github.com/spike-events/spike-broker/v2/pkg/broker"
)

// handleRequest
func handleRequest(sub broker.Subscription, msg broker.Call, access broker.Access, opts Options) {
	p := sub.Resource

	defer func() {
		if r := recover(); r != nil {
			var rErr broker.Error
			err, ok := r.(error)
			if !ok {
				rErr = broker.InternalError(fmt.Errorf("request: %s: %v", p.EndpointName(), r))
			} else {
				rErr = broker.InternalError(err)
			}
			opts.Service.Logger().Printf("request: panic on handler: %s %s", p.EndpointName(),
				p.EndpointNameSpecific())
			opts.Service.Logger().Printf("request: panic on handler: %v", r)
			msg.Error(rErr)
		}
	}()

	// Authenticate and Authorize
	if p.Method() != "INTERNAL" && !p.Public() {
		// Test Token Authentication
		var valid bool
		var token json.RawMessage
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

	if len(sub.Validators) > 0 {
		for _, validator := range sub.Validators {
			validator(access)
			if err := access.GetError(); err != nil {
				msg.Error(err)
				return
			}
		}
	}

	sub.Handler(msg)
}
