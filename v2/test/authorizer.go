package test

import (
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/service"
)

type Authorizer struct {
}

func (a Authorizer) HasPermission(c broker.Call) bool {
	if c.Endpoint().EndpointName() == ServiceTestRid().TestReply().EndpointName() {
		return true
	}
	return false
}

func NewAuthorizer() service.Authorizer {
	return &Authorizer{}
}
