package spike

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/broker/providers/testProvider"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/service"
)

var serviceTestImplInstance *testServiceImpl

type testServiceImpl struct {
	opts              Options
	testProvider      testProvider.Provider
	startRepository   service.Repository
	startRequestMocks testProvider.RequestMock
}

func (b *testServiceImpl) Start(options Options) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.opts.Timeout)
	tp := testProvider.NewTestProvider(ctx, cancel)

	b.opts = options
	b.opts.Provider = tp
	b.opts.Repository = b.startRepository
	config := service.Config{
		Broker:     tp,
		Logger:     options.Logger,
		Repository: b.startRepository,
	}

	if err := b.opts.Service.SetConfig(config); err != nil {
		return err
	}

	// TODO: Test Migrator

	// Initialize Handlers
	handlers := b.opts.Service.Handlers()
	for _, h := range handlers {
		_, err := tp.Subscribe(h)
		if err != nil {
			return err
		}
	}

	// Initialize Monitors
	monitos := b.opts.Service.Monitors()
	for g, m := range monitos {
		_, err := tp.Monitor(g, m)
		if err != nil {
			return err
		}
	}

	// Start the service
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}

	err = b.opts.Service.Start(id, ctx)
	if err != nil {
		return err
	}

	return nil
}

func (b *testServiceImpl) Stop(timeout time.Duration) error {
	b.opts.Provider.Close()
	return nil
}

func (b *testServiceImpl) TestRequestOrPublish(
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

	err := b.opts.Service.SetConfig(service.Config{
		Repository: repository,
		Broker:     b.opts.Provider,
		Logger:     b.opts.Logger,
	})
	if err != nil {
		return broker.InternalError(err)
	}

	b.testProvider.SetMocks(mocks)

	call := testProvider.NewCall(p, payload, token, requestOk, requestErr)
	access := testProvider.NewAccess(p, payload, token, accessOk, accessErr)
	handleRequest(p, call, access, b.opts)
	if access.GetError() != nil {
		return access.GetError()
	}

	return call.GetError()
}
