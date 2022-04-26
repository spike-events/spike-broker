package spike

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
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

	s.opts.Broker.SetHandler(func(p rids.Pattern, payload []byte) {
		call := broker.NewCall(p, payload)
		access := broker.NewAccess(call)
		handleRequest(p, call, access, *s.opts)
	})

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
