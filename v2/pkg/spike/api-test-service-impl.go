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

func (s *testServiceImpl) RegisterService(options Options) error {
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

func (s *testServiceImpl) StartService() error {
	if s.opts == nil {
		return fmt.Errorf("API not initialized")
	}

	// Initialize Handlers
	handlers := s.opts.Service.Handlers()
	for _, h := range handlers {
		_, err := s.broker.Subscribe(h)
		if err != nil {
			return err
		}
	}

	// Initialize Monitors
	monitos := s.opts.Service.Monitors()
	for g, m := range monitos {
		_, err := s.broker.Monitor(g, m)
		if err != nil {
			return err
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

	err := s.opts.Service.SetRepository(params.Repository)
	if err != nil {
		return broker.InternalError(err)
	}

	s.broker.SetMocks(params.Mocks)

	call := testProvider.NewCall(params.Pattern, params.Payload, params.Token, nil, nil)
	access := testProvider.NewAccess(params.Pattern, params.Payload, params.Token, nil, nil)
	handleAccessForTest(params.Pattern, call, access, params.Ok, params.Err, *s.opts)
	return access.GetError()
}

func (s *testServiceImpl) TestRequestOrPublish(params APITestRequestOrPublish) broker.Error {
	if s.opts == nil {
		return broker.InternalError(fmt.Errorf("API not initialized"))
	}

	err := s.opts.Service.SetRepository(params.Repository)
	if err != nil {
		return broker.InternalError(err)
	}

	s.broker.SetMocks(params.Mocks)

	call := testProvider.NewCall(params.Pattern, params.Payload, params.Token, params.Ok, params.Err)
	handleRequestForTest(params.Pattern, call, *s.opts)
	return call.GetError()
}
