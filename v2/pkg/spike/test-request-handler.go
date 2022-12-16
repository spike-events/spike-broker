package spike

import (
	"encoding/json"
	"fmt"

	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

func handleAccessForTest(p rids.Pattern, msg broker.Call, access broker.Access, handleOk func(), handleErr func(int, interface{}), opts Options) {
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
			access.Error(rErr)
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
		access.Error(broker.ErrorServiceUnavailable)
		return
	}

	// Authenticate and Authorize
	if !p.Public() {
		// Test Token Authentication
		var valid bool
		var token json.RawMessage
		if token, valid = opts.Authenticator.ValidateToken(access.RawToken()); !valid {
			handleErr(-1, broker.ErrorStatusUnauthorized)
			return
		}

		msg.SetToken(token)

		// Test Route Authorization
		if !opts.Authorizer.HasPermission(msg) {
			handleErr(-2, broker.ErrorStatusForbidden)
			return
		}
	}

	if len(handler.Validators) > 0 {
		for i, validator := range handler.Validators {
			validator(access)
			if err := access.GetError(); err != nil {
				handleErr(i, err)
				return
			}
		}
	}

	handleOk()
}

func handleRequestForTest(p rids.Pattern, msg broker.Call, opts Options) {
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
		msg.Error(broker.ErrorServiceUnavailable)
		return
	}

	handler.Handler(msg)
}
