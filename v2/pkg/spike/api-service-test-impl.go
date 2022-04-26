package spike

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/broker/providers/testProvider"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/service"
)

var serviceTestImplInstance *testServiceImpl

type testServiceImpl struct {
	opts              *Options
	testProvider      testProvider.Provider
	startRepository   service.Repository
	startRequestMocks testProvider.RequestMock
	ctx               context.Context
	cancel            context.CancelFunc
	id                uuid.UUID
}

func (s *testServiceImpl) Initialize(options Options) error {
	s.opts = &options
	s.ctx, s.cancel = context.WithTimeout(context.Background(), options.Timeout)
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	s.id = id
	return nil
}

func (s *testServiceImpl) StartService() error {
	if s.opts == nil {
		return fmt.Errorf("API not initialized")
	}

	tp := testProvider.NewTestProvider(s.ctx, s.cancel)
	s.opts.Broker = tp
	s.opts.Repository = s.startRepository
	config := service.Config{
		Broker:     tp,
		Logger:     s.opts.Logger,
		Repository: s.startRepository,
	}

	if err := s.opts.Service.SetConfig(config); err != nil {
		return err
	}

	// TODO: Test Migrator

	// Initialize Handlers
	handlers := s.opts.Service.Handlers()
	for _, h := range handlers {
		_, err := tp.Subscribe(h)
		if err != nil {
			return err
		}
	}

	// Initialize Monitors
	monitos := s.opts.Service.Monitors()
	for g, m := range monitos {
		_, err := tp.Monitor(g, m)
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

	s.opts.Broker.Close()
	return nil
}

func (s *testServiceImpl) TestRequestOrPublish(
	p rids.Pattern,
	repository interface{},
	payload interface{},
	token string,
	requestOk func(interface{}),
	accessOk func(interface{}),
	requestErr func(interface{}),
	accessErr func(interface{}),
	mocks testProvider.Mocks,
) broker.Error {
	if s.opts == nil {
		return broker.InternalError(fmt.Errorf("API not initialized"))
	}

	err := s.opts.Service.SetConfig(service.Config{
		Repository: repository,
		Broker:     s.opts.Broker,
		Logger:     s.opts.Logger,
	})
	if err != nil {
		return broker.InternalError(err)
	}

	s.testProvider.SetMocks(mocks)

	call := testProvider.NewCall(p, payload, token, requestOk, requestErr)
	access := testProvider.NewAccess(p, payload, token, accessOk, accessErr)
	handleRequest(p, call, access, *s.opts)
	if access.GetError() != nil {
		return access.GetError()
	}

	return call.GetError()
}
