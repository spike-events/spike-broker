package spike

import (
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/broker/providers/testProvider"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

type APITestRequestOrPublish struct {
	Pattern    rids.Pattern
	Repository interface{}
	Payload    interface{}
	Token      string
	Mocks      testProvider.Mocks
	Ok         func(...interface{})
	Err        func(interface{})
}

type APITestAccess struct {
	Pattern    rids.Pattern
	Repository interface{}
	Payload    interface{}
	Token      string
	Mocks      testProvider.Mocks
	Ok         func()
	Err        func(int, interface{})
}

type APITestService interface {
	APIService
	TestAccess(params APITestAccess) broker.Error
	TestRequestOrPublish(params APITestRequestOrPublish) broker.Error
}

// NewAPITestService returns an APIService implementation that runs unit tests based on specified parameters
func NewAPITestService(startRequestMocks testProvider.Mocks) APITestService {
	if serviceTestImplInstance == nil {
		serviceTestImplInstance = &testServiceImpl{
			startRequestMocks: startRequestMocks,
		}
	}
	return serviceTestImplInstance
}
