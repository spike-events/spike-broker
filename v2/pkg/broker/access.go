package broker

// AccessHandler is a function that validates request's specific situations
type AccessHandler func(a Access)

type Access interface {
	Call
	AccessDenied()
	AccessGranted()
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

func (a *accessRequest) AccessDenied() {
	a.err = ErrorStatusForbidden
}

func (a *accessRequest) AccessGranted() {
	a.err = nil
}

func (a *accessRequest) OK(_ ...interface{}) {

}
