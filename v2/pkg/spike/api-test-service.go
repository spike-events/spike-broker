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
	RequestOk  func(...interface{})
	AccessOk   func(...interface{})
	RequestErr func(interface{})
	AccessErr  func(interface{})
	Mocks      testProvider.Mocks
}

type APITestService interface {
	APIService
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
