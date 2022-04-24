package broker

// AccessHandler is a function that validates request's specific situations
type AccessHandler func(a Access)

type Access interface {
	Call
	Methods() []string
	RequestIsGet() *bool
	Access(get bool, methods ...string)
	AccessDenied()
	AccessGranted()
}

func NewAccess(callMsg Call) Access {
	return &accessRequest{
		call: call{
			Data:     callMsg.RawData(),
			ReplyStr: callMsg.Reply(),
			provider: callMsg.Provider(),
			Token:    callMsg.RawToken(),
		},
		get:     nil,
		methods: nil,
	}
}

type accessRequest struct {
	call
	get     *bool
	methods []string
}

func (a *accessRequest) Methods() []string {
	return a.methods
}

func (a *accessRequest) RequestIsGet() *bool {
	return a.get
}

func (a *accessRequest) Access(get bool, methods ...string) {
	a.err = nil
	a.get = &get
	a.methods = methods
}
func (a *accessRequest) AccessDenied() {
	a.err = ErrorStatusForbidden
}

func (a *accessRequest) AccessGranted() {
	a.err = nil
}
