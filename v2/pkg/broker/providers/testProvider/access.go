package testProvider

import (
	"encoding/json"

	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

type accessRequest struct {
	callRequest
}

func (a *accessRequest) AccessDenied(err ...broker.Error) {
	if len(err) > 0 {
		a.result = err[0]
	} else {
		a.result = broker.ErrorAccessDenied
	}
	if a.errF != nil {
		a.errF(a.result)
	}
}

func (a *accessRequest) AccessGranted() {
	if a.okF != nil {
		a.okF()
	}
}

func NewAccess(p rids.Pattern, payload interface{}, token json.RawMessage, okF func(...interface{}), errF func(interface{})) broker.Access {
	c := NewCall(p, payload, token, okF, errF, nil)
	return &accessRequest{
		callRequest: *c.(*callRequest),
	}
}
