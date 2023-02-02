package broker

import (
	"encoding/json"
	"time"

	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

// AccessHandler is a function that validates request's specific situations
type AccessHandler func(a Access)

type Access interface {
	RawToken() []byte
	RawData() []byte
	Reply() string
	Provider() Provider
	Endpoint() rids.Pattern

	PathParam(key string) string
	ParseQuery(q interface{}) error

	ToJSON() json.RawMessage
	Timeout(timeout time.Duration)

	AccessDenied(err ...Error)
	AccessGranted()

	GetError() Error

	SetToken(token []byte)
	SetProvider(provider Provider)
}

func NewAccess(callMsg Call) Access {
	return &accessRequest{
		call: call{
			callBase: callBase{
				Data:            callMsg.RawData(),
				ReplyStr:        callMsg.Reply(),
				EndpointPattern: callMsg.Endpoint(),
				Token:           callMsg.RawToken(),
				provider:        callMsg.Provider(),
			},
			APIVersion: 2,
		},
	}
}

type accessRequest struct {
	call
}

func (a *accessRequest) AccessDenied(err ...Error) {
	if len(err) > 0 {
		a.err = err[0]
		return
	}
	a.err = ErrorAccessDenied
}

func (a *accessRequest) AccessGranted() {
	a.err = nil
}
