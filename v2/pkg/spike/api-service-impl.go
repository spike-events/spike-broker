package spike

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/service"
)

type serviceImpl struct {
	opts     *Options
	wsPrefix string
	ctx      context.Context
	cancel   context.CancelFunc
	id       uuid.UUID
	broker   broker.Provider
	logger   service.Logger
}

func (s *serviceImpl) RegisterService(options Options) error {
	s.opts = &options
	s.ctx, s.cancel = context.WithTimeout(context.Background(), options.Timeout)
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	s.id = id
	s.broker = options.Service.Broker()
	s.logger = options.Service.Logger()
	return nil
}

func (s *serviceImpl) StartService() error {
	if s.opts == nil {
		return fmt.Errorf("API not initialized")
	}

	s.broker.SetHandler(s.opts.Service.Rid().Name(), func(sub broker.Subscription, payload []byte, replyEndpoint string) {
		call, err := broker.NewCallFromJSON(payload, sub.Resource, replyEndpoint)
		if err == nil {
			call.SetProvider(s.broker)
			access := broker.NewAccess(call)
			handleRequest(sub, call, access, *s.opts)
		}
	})

	if s.opts.Service == nil {
		return fmt.Errorf("no service specified on options")
	}

	if s.opts.Service.Handlers() == nil {
		return fmt.Errorf("service must implement a valid Handlers() function")
	}

	if s.broker == nil {
		return fmt.Errorf("no broker specified on options")
	}

	if s.logger == nil {
		return fmt.Errorf("no logger specified on options")
	}

	for _, sub := range s.opts.Service.Handlers() {
		if _, err := s.broker.Subscribe(sub); err != nil {
			return err
		}
	}

	if s.opts.Service.Monitors() != nil {
		for group, sub := range s.opts.Service.Monitors() {
			if _, err := s.broker.Monitor(group, sub); err != nil {
				return err
			}
		}
	}

	if err := s.opts.Service.Start(s.id, s.ctx); err != nil {
		return err
	}
	return nil
}

func (s *serviceImpl) Stop() error {
	if s.opts == nil {
		return fmt.Errorf("API not initialized")
	}

	<-s.opts.Service.Stop()
	s.cancel()
	return nil
}
