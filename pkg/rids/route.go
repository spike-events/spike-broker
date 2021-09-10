package rids

type route struct{ Base }

var routeImp route

// Route func
func Route() *route {
	routeImp.name = "route"
	routeImp.label = "route"
	return &routeImp
}

func (r *route) ValidateToken() *Pattern {
	return r.NewMethod("", "token.validate").Internal()
}

func (r *route) UserHavePermission() *Pattern {
	return r.NewMethod("", "user.have.permission").Internal()
}

func (r *route) Ready() *Pattern {
	return r.NewMethod("", "ready").NoAuth().Get()
}

func (r *route) EventSocketConnected() *Pattern {
	return r.NewMethod("User has authenticated through Socket channel", "socket.connected").Internal()
}

func (r *route) EventSocketDisconnected(id ...string) *Pattern {
	return r.NewMethod("User has disconnected from Socket channel", "socket.disconnected.$Id", id...).Internal()
}
