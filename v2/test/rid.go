package test

import (
	"fmt"

	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

type serviceTestRid struct {
	rids.Base
}

var serviceTestRidImpl *serviceTestRid

func ServiceTestRid() *serviceTestRid {
	if serviceTestRidImpl == nil {
		serviceTestRidImpl = &serviceTestRid{
			Base: rids.NewRid("serviceTest", "Service Test", "api"),
		}
	}
	return serviceTestRidImpl
}

func (s *serviceTestRid) TestReply(id ...fmt.Stringer) rids.Pattern {
	return s.NewMethod("Returns the ID received", "reply.$ID", id...).Get()
}

func (s *serviceTestRid) ForbiddenCall() rids.Pattern {
	return s.NewMethod("Emulate an endpoint that cannot be accessed", "forbidden").Internal()
}

func (s *serviceTestRid) FromMock() rids.Pattern {
	return s.NewMethod("Emulate an endpoint to be used on mock", "from.mock").Get()
}
