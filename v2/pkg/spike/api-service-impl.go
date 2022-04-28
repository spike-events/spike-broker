package spike

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/service"
)

type serviceImpl struct {
	opts     *Options
	wsPrefix string
	ctx      context.Context
	cancel   context.CancelFunc
	id       uuid.UUID
}

func (s *serviceImpl) Initialize(options Options) error {
	s.opts = &options
	s.ctx, s.cancel = context.WithTimeout(context.Background(), options.Timeout)
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	s.id = id
	return nil
}

func (s *serviceImpl) StartService() error {
	if s.opts == nil {
		return fmt.Errorf("API not initialized")
	}

	s.opts.Broker.SetHandler(func(p rids.Pattern, payload []byte, replyEndpoint string) {
		call, err := broker.NewCallFromJSON(payload)
		if err == nil {
			call.SetProvider(s.opts.Broker)
			call.SetReply(replyEndpoint)
			access := broker.NewAccess(call)
			handleRequest(p, call, access, *s.opts)
		}
	})

	if s.opts.Service == nil {
		return fmt.Errorf("no service specified on options")
	}

	if s.opts.Service.Handlers() == nil {
		return fmt.Errorf("service must implement a valid Handlers() function")
	}

	if s.opts.Broker == nil {
		return fmt.Errorf("no broker specified on options")
	}

	if s.opts.Logger == nil {
		return fmt.Errorf("no logger specified on options")
	}

	err := s.opts.Service.SetConfig(service.Config{
		Repository: s.opts.Repository,
		Broker:     s.opts.Broker,
		Logger:     s.opts.Logger,
	})
	if err != nil {
		return err
	}

	for _, sub := range s.opts.Service.Handlers() {
		if _, err = s.opts.Broker.Subscribe(sub); err != nil {
			return err
		}
	}

	if s.opts.Service.Monitors() != nil {
		for group, sub := range s.opts.Service.Monitors() {
			if _, err = s.opts.Broker.Monitor(group, sub); err != nil {
				return err
			}
		}
	}

	if err = s.opts.Service.Start(s.id, s.ctx); err != nil {
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
