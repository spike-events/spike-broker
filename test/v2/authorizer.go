package v2

import (
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/service"
)

type Authorizer struct {
}

func (a Authorizer) HasPermission(c broker.Call) bool {
	switch c.Endpoint().EndpointName() {
	case ServiceTestRid().TestReply().EndpointName(),
		ServiceTestRid().FromMock().EndpointName(),
		ServiceTestRid().CallWithObjPayload().EndpointName(),
		ServiceTestRid().CallV1().EndpointName(),
		ServiceTestRid().CallExpectingFile().EndpointName():
		return true
	}
	return false
}

func NewAuthorizer() service.Authorizer {
	return &Authorizer{}
}
