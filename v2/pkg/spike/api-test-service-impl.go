package spike

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/broker/providers/testProvider"
	"github.com/spike-events/spike-broker/v2/pkg/service"
)

var serviceTestImplInstance *testServiceImpl

type testServiceImpl struct {
	opts              *Options
	startRepository   interface{}
	startRequestMocks testProvider.Mocks
	ctx               context.Context
	cancel            context.CancelFunc
	id                uuid.UUID
	broker            testProvider.Provider
	logger            service.Logger
}

func (s *testServiceImpl) Setup(options Options) error {
	s.opts = &options
	s.ctx, s.cancel = context.WithTimeout(context.Background(), options.Timeout)
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	s.id = id
	var valid bool
	if s.broker, valid = options.Service.Broker().(testProvider.Provider); !valid {
		panic("must use testProver.Provider as broker on unit tests")
	}
	s.logger = options.Service.Logger()
	return nil
}

func (s *testServiceImpl) RegisterService(options Options) error {
	return s.Setup(options)
}

func (s *testServiceImpl) StartService() error {
	if s.opts == nil {
		return fmt.Errorf("API not initialized")
	}

	// Initialize Handlers
	handlers := s.opts.Service.Handlers()
	for _, h := range handlers {
		_, err := s.broker.Subscribe(h, nil)
		if err != nil {
			return err
		}
	}

	// Initialize Monitors
	if withMonitors, ok := s.opts.Service.(service.WithMonitors); ok {
		monitors := withMonitors.Monitors()
		for g, ms := range monitors {
			for _, m := range ms {
				_, err := s.broker.Monitor(fmt.Sprintf("%s-%s", s.opts.Service.Rid().Name(), g), m, nil)
				if err != nil {
					return err
				}
			}
		}
	}

	// StartService the service
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}

	err = s.opts.Service.Start(id, s.ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *testServiceImpl) Stop() error {
	if s.opts == nil {
		return fmt.Errorf("API not initialized")
	}

	s.broker.Close()
	return nil
}

func (s *testServiceImpl) TestAccess(params APITestAccess) broker.Error {
	if s.opts == nil {
		return broker.InternalError(fmt.Errorf("API not initialized"))
	}

	if withRepository, ok := s.opts.Service.(service.WithRepository); ok {
		var repo interface{}
		if params.Repository != nil {
			repo = params.Repository
		} else if params.Mocks.Repository != nil {
			repo = params.Repository
		}
		err := withRepository.SetRepository(repo)
		if err != nil {
			return broker.InternalError(err)
		}
	}

	if params.Mocks.ExternalAPIs != nil {
		if withExternalAPI, ok := s.opts.Service.(service.WithExternalAPI); ok {
			for api, impl := range params.Mocks.ExternalAPIs {
				err := withExternalAPI.SetExternalAPI(api, impl)
				if err != nil {
					return broker.InternalError(err)
				}
			}
		}
	}
	s.broker.SetMocks(params.Mocks)

	call := testProvider.NewCall(params.Pattern, params.Payload, params.Token, nil, nil, nil)
	access := testProvider.NewAccess(params.Pattern, params.Payload, params.Token, nil, nil)
	handleAccessForTest(params.Pattern, call, access, params.Ok, params.Err, *s.opts)
	return access.GetError()
}

func (s *testServiceImpl) TestRequestOrPublish(params APITestRequestOrPublish) broker.Error {
	if s.opts == nil {
		return broker.InternalError(fmt.Errorf("API not initialized"))
	}

	if withRepository, ok := s.opts.Service.(service.WithRepository); ok {
		var repo interface{}
		if params.Repository != nil {
			repo = params.Repository
		} else if params.Mocks.Repository != nil {
			repo = params.Repository
		}
		err := withRepository.SetRepository(repo)
		if err != nil {
			return broker.InternalError(err)
		}
	}

	if params.Mocks.ExternalAPIs != nil {
		if withExternalAPI, ok := s.opts.Service.(service.WithExternalAPI); ok {
			for api, impl := range params.Mocks.ExternalAPIs {
				err := withExternalAPI.SetExternalAPI(api, impl)
				if err != nil {
					return broker.InternalError(err)
				}
			}
		}
	}

	s.broker.SetMocks(params.Mocks)

	call := testProvider.NewCall(params.Pattern, params.Payload, params.Token, params.Ok, params.Err, params.File)
	handleRequestForTest(params.Pattern, call, *s.opts)
	return call.GetError()
}
