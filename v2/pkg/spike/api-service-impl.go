package spike

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
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

func (s *serviceImpl) Setup(options Options) error {
	s.opts = &options
	s.ctx, s.cancel = context.WithCancel(context.Background())
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	s.id = id
	s.broker = options.Service.Broker()
	s.logger = options.Service.Logger()
	return nil
}

func (s *serviceImpl) RegisterService(options Options) error {
	options.Service.Logger().Printf("use Setup instead of RegisterService")
	return s.Setup(options)
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

	// Subscribe Live method
	if _, err := s.broker.Subscribe(broker.Subscription{
		Resource: s.opts.Service.Rid().Live(),
		Handler:  func(c broker.Call) { c.OK() },
	}, handler); err != nil {
		return err
	}

	if withMonitors, ok := s.opts.Service.(service.WithMonitors); ok && withMonitors.Monitors() != nil {
		s.monitorSubs = make([]func(), 0)
		for group, subs := range withMonitors.Monitors() {
			for _, sub := range subs {
				ctxGroup := fmt.Sprintf("%s-%s", s.opts.Service.Rid().Name(), group)
				if unsubscribe, err := s.broker.Monitor(ctxGroup, sub, handler); err == nil {
					s.monitorSubs = append(s.monitorSubs, unsubscribe)
				} else {
					return err
				}
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
				a := broker.NewAccess(call)
				s.validateMonitor(a)
				if err = a.GetError(); err != nil {
					call.Error(err)
					return
				}
				call.OK()
			}
		}
		_, err := s.broker.Subscribe(monitorValidateSub, eventHandlerForValidation)
		if err != nil {
			return err
		}

		// We also need to create a handler to check if Publish Validators allow publishing.
		publishValidateSub := broker.Subscription{
			Resource: s.opts.Service.Rid().ValidatePublish(),
		}
		eventHandlerForValidation = func(sub broker.Subscription, payload []byte, replyEndpoint string) {
			call, err := broker.NewCallFromJSON(payload, sub.Resource, replyEndpoint)
			if err == nil {
				call.SetProvider(s.broker)
				a := broker.NewAccess(call)
				s.validatePublish(a)
				if err = a.GetError(); err != nil {
					call.Error(err)
					return
				}
				call.OK()
			}
		}
		_, err = s.broker.Subscribe(publishValidateSub, eventHandlerForValidation)
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
		s.logger.Printf("stopping: unsubscribed from all monitors")
	} else {
		s.logger.Printf("stopping: no monitor to unsubscribe")
	}

	<-s.opts.Service.Stop()
	s.logger.Printf("stopping: service stopped, cancelling context")
	s.cancel()
	return nil
}

func (s *serviceImpl) validateMonitor(access broker.Access) {
	p, err := rids.UnmarshalPattern(access.RawData())
	if err != nil {
		access.AccessDenied(broker.InternalError(fmt.Errorf("invalid rids.Pattern on payload: %s", err)))
		return
	}

	withEvents, ok := s.opts.Service.(service.WithEvents)
	if !ok {
		access.AccessDenied(broker.InternalError(fmt.Errorf("service does not declare events")))
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

	access.AccessDenied(broker.InternalError(fmt.Errorf("endpoint %s not declared on Events arrays", p.EndpointName())))
}

func (s *serviceImpl) validatePublish(access broker.Access) {
	p, err := rids.UnmarshalPattern(access.RawData())
	if err != nil {
		access.AccessDenied(broker.InternalError(fmt.Errorf("invalid rids.Pattern on payload: %s", err)))
		return
	}

	withEvents, ok := s.opts.Service.(service.WithEvents)
	if !ok {
		access.AccessDenied(broker.InternalError(fmt.Errorf("service does not declare events")))
		return
	}

	for _, event := range withEvents.Events() {
		if event.Resource.EndpointName() == p.EndpointName() {
			if len(event.PublishValidators) == 0 {
				access.AccessGranted()
				return
			}

			for _, validator := range event.PublishValidators {
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

	access.AccessDenied(broker.InternalError(fmt.Errorf("endpoint %s not declared on Events arrays", p.EndpointName())))
}
