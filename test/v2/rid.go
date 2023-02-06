package v2

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

func (s *serviceTestRid) RootEP() rids.Pattern {
	return s.NewMethod("Can use root RID", "").Get()
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

func (s *serviceTestRid) CallV1() rids.Pattern {
	return s.NewMethod("Call a Spike V1 service method", "callV1").Get()
}

func (s *serviceTestRid) CallV1Forbidden() rids.Pattern {
	return s.NewMethod("Call a Spike V1 service method that returns error", "callV1.forbidden").Get()
}

func (s *serviceTestRid) CallWithObjPayload() rids.Pattern {
	return s.NewMethod("Call passing a map expecting to turn into an object", "payload.object").Post()
}

func (s *serviceTestRid) NoHandler() rids.Pattern {
	return s.NewMethod("Route without handler to test 503 errors on NATS", "invalid").Get()
}

func (s *serviceTestRid) CallExpectingFile() rids.Pattern {
	return s.NewMethod("Call and get a file on response", "getFile").Get()
}
