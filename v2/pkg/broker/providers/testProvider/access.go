package testProvider

import (
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

type accessRequest struct {
	callRequest
}

func (a *accessRequest) AccessDenied() {
	a.result = broker.ErrorAccessDenied
	a.errF(a.result)
}

func (a *accessRequest) AccessGranted() {
	a.okF()
}

func NewAccess(p rids.Pattern, payload interface{}, token string, okF func(...interface{}), errF func(interface{})) broker.Access {
	c := NewCall(p, payload, token, okF, errF)
	return &accessRequest{
		callRequest: *c.(*callRequest),
	}
}
