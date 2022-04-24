package testProvider

import (
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

type accessRequest struct {
	callRequest
}

func (a accessRequest) GetError() broker.Error {
	//TODO implement me
	panic("implement me")
}

func (a accessRequest) Methods() []string {
	//TODO implement me
	panic("implement me")
}

func (a accessRequest) IsError() bool {
	//TODO implement me
	panic("implement me")
}

func (a accessRequest) RequestIsGet() *bool {
	//TODO implement me
	panic("implement me")
}

func (a accessRequest) Access(get bool, methods ...string) {
	//TODO implement me
	panic("implement me")
}

func (a accessRequest) AccessDenied() {
	//TODO implement me
	panic("implement me")
}

func (a accessRequest) AccessGranted() {
	//TODO implement me
	panic("implement me")
}

func NewAccess(p rids.Pattern, payload interface{}, token string, okF func(interface{}), errF func(interface{})) broker.Access {
	return &accessRequest{}
}
