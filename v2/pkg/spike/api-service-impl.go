package spike

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/service"
)

type serviceImpl struct {
	opts        *Options
	wsPrefix    string
	ctx         context.Context
	cancel      context.CancelFunc
	id          uuid.UUID
	broker      broker.Provider
	logger      service.Logger
	monitorSubs []func()
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

	handler := func(sub broker.Subscription, payload []byte, replyEndpoint string) {
		call, err := broker.NewCallFromJSON(payload, sub.Resource, replyEndpoint)
		if err == nil {
			call.SetProvider(s.broker)
			access := broker.NewAccess(call)
			handleRequest(sub, call, access, *s.opts)
		}
	}

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
		if _, err := s.broker.Subscribe(sub, handler); err != nil {
			return err
		}
	}

	if withMonitors, ok := s.opts.Service.(service.WithMonitors); ok && withMonitors.Monitors() != nil {
		s.monitorSubs = make([]func(), 0)
		for group, sub := range withMonitors.Monitors() {
			if unsubscribe, err := s.broker.Monitor(group, sub, handler); err == nil {
				s.monitorSubs = append(s.monitorSubs, unsubscribe)
			} else {
				return err
			}
		}
	}

	if withEvents, ok := s.opts.Service.(service.WithEvents); ok && withEvents.Events() != nil {
		// In case the service has Events, we need to create a handler that allows the websocket implementation
		// to check if Monitor Validators allow the websocket monitor.
		monitorValidateSub := broker.Subscription{
			Resource: s.opts.Service.Rid().ValidateMonitor(),
		}
		eventHandlerForValidation := func(sub broker.Subscription, payload []byte, replyEndpoint string) {
			call, err := broker.NewCallFromJSON(payload, sub.Resource, replyEndpoint)
			if err == nil {
				call.SetProvider(s.broker)
				s.validateMonitor(broker.NewAccess(call))
			}
		}
		_, err := s.broker.Subscribe(monitorValidateSub, eventHandlerForValidation)
		if err != nil {
			return err
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

	if s.monitorSubs != nil && len(s.monitorSubs) > 0 {
		for _, unsubscribe := range s.monitorSubs {
			unsubscribe()
		}
	}

	<-s.opts.Service.Stop()
	s.cancel()
	return nil
}

func (s *serviceImpl) validateMonitor(access broker.Access) {
	var p rids.Pattern
	err := json.Unmarshal(access.RawData(), &p)
	if err != nil {
		access.Error(broker.ErrorInvalidParams, fmt.Sprintf("Invalid rids.Pattern on payload: %s", err))
		return
	}

	withEvents, ok := s.opts.Service.(service.WithEvents)
	if !ok {
		access.Error(broker.ErrorInternalServerError, "Service does not declare events")
		return
	}

	for _, event := range withEvents.Events() {
		if event.Resource.EndpointName() == p.EndpointName() {
			if len(event.MonitorValidators) == 0 {
				access.AccessGranted()
				return
			}

			for _, validator := range event.MonitorValidators {
				validator(access)
				if access.GetError() != nil {
					// Test failed and message sent
					return
				}
			}
			// All validators executed without error
			return
		}
	}

	access.Error(broker.ErrorInvalidParams, fmt.Sprintf("Endpoint %s not declared on Events arrays", p.EndpointName()))
}
