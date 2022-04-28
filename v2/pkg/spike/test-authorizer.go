package spike

import (
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/service"
)

type testAuthorizer struct {
	authorizer func(c broker.Call) bool
}

func (t testAuthorizer) HasPermission(c broker.Call) bool {
	return t.authorizer(c)
}

func NewTestAuthorizer(authorizer func(c broker.Call) bool) service.Authorizer {
	return &testAuthorizer{
		authorizer: authorizer,
	}
}
