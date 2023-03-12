package spike

import (
	"encoding/json"

	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/broker/providers/testProvider"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/vincent-petithory/dataurl"
)

type APITestRequestOrPublish struct {
	Pattern    rids.Pattern
	Repository interface{} // DEPRECATED in favor of Mocks.Repository
	Payload    interface{}
	Token      json.RawMessage
	Mocks      testProvider.Mocks
	Ok         func(...interface{})
	Err        func(interface{})
	File       func(f *dataurl.DataURL)
}

type APITestAccess struct {
	Pattern    rids.Pattern
	Repository interface{} // DEPRECATED in favor of Mocks.Repository
	Payload    interface{}
	Token      json.RawMessage
	Mocks      testProvider.Mocks
	Ok         func()
	Err        func(int, interface{})
}

type APITestService interface {
	APIService
	TestAccess(params APITestAccess) broker.Error
	TestRequestOrPublish(params APITestRequestOrPublish) broker.Error
	TestRequestOrPublishBurst(params []APITestRequestOrPublish) broker.Error
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
