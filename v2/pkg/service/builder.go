package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

type ResourceDefinition struct {
	RID    string
	Public bool
	Method rids.MethodType
}

type MonitorDefinition struct {
	Pattern rids.Pattern
	Handler broker.CallHandler
}

type EventDefinition struct {
	PublishValidators []broker.AccessHandler
	MonitorValidators []broker.AccessHandler
}

func BuildResources(name string, httpPrefix string, handlers map[string]string,
	events map[string]string, public []string) *ResourceBuilder {
	ridsLabel := fmt.Sprintf("%s built by generated", name)
	resource := &ResourceBuilder{
		Base:      rids.NewRid(name, ridsLabel, httpPrefix),
		endpoints: make(map[string]ResourceDefinition),
		handlers:  make(map[string]broker.Subscription),
		monitors:  make(map[string][]broker.Subscription),
	}

	for n, rid := range handlers {
		r := resourceDefinitionFromString(rid)
		if slices.Contains(public, n) {
			r.Public = true
		}
		resource.endpoints[n] = r
	}

	for n, rid := range events {
		if _, ok := resource.endpoints[n]; ok {
			panic(fmt.Sprintf("%s as event is duplicate", n))
		}
		r := ResourceDefinition{
			RID:    rid,
			Method: rids.EVENT,
		}
		if slices.Contains(public, n) {
			r.Public = true
		}
		resource.endpoints[n] = r
	}

	return resource
}

type Generated interface {
	Service

	Context() context.Context
}

func Build(provider broker.Provider, logger Logger, resource *ResourceBuilder,
	handlers map[string]broker.CallHandler, events map[string]EventDefinition,
	validators map[string][]broker.AccessHandler,
	monitors map[string][]MonitorDefinition) Generated {

	for name, handler := range handlers {
		var v []broker.AccessHandler
		if validators != nil {
			v, _ = validators[name]
		}
		resource.addHandler(name, handler, v)
	}

	for group, monitor := range monitors {
		resource.addMonitorGroup(group, monitor)
	}

	for name, evt := range events {
		resource.addEvent(name, evt)
	}

	var svc Generated
	base := generated{
		provider: provider,
		logger:   logger,
		resource: resource,
	}
	if len(resource.monitors) > 0 && len(resource.events) > 0 {
		svc = &generatedWithMonitorsAndEvents{
			generated: base,
		}
	} else if len(resource.monitors) > 0 {
		svc = &generatedWithMonitors{
			generated: base,
		}
	} else if len(resource.events) > 0 {
		svc = &generatedWithEvents{
			generated: base,
		}
	} else {
		svc = &base
	}

	return svc
}

type ResourceBuilder struct {
	rids.Base
	endpoints map[string]ResourceDefinition
	handlers  map[string]broker.Subscription
	monitors  map[string][]broker.Subscription
	events    []broker.Event
}

func (r *ResourceBuilder) addHandler(name string, handler broker.CallHandler,
	validators []broker.AccessHandler) {
	rid := r.endpoints[name]
	h := broker.Subscription{
		Resource: r.patternFromRID(name, rid.RID,
			rid.Method, rid.Public),
		Handler:    handler,
		Validators: validators,
	}
	r.handlers[name] = h
}

func (r *ResourceBuilder) addMonitorGroup(group string, monitors []MonitorDefinition) {
	for _, monitor := range monitors {
		r.monitors[group] = append(r.monitors[group],
			broker.Subscription{
				Resource: monitor.Pattern,
				Handler:  monitor.Handler,
			})
	}
}

func (r *ResourceBuilder) addEvent(name string, evt EventDefinition) {
	rid := r.endpoints[name]
	r.events = append(r.events, broker.Event{
		Resource: r.patternFromRID(name, rid.RID, rids.EVENT,
			rid.Public),
		PublishValidators: evt.PublishValidators,
		MonitorValidators: evt.MonitorValidators,
	})
}

func (r *ResourceBuilder) Endpoint(name string, params ...fmt.Stringer) rids.Pattern {
	ep := r.endpoints[name]
	return r.patternFromRID(name, ep.RID, ep.Method, ep.Public, params...)
}

func (r *ResourceBuilder) patternFromRID(name, rid string, method rids.MethodType,
	public bool, params ...fmt.Stringer) rids.Pattern {
	m := r.NewMethod(name, rid, params...)
	if public {
		m = m.Public()
	}
	switch method {
	case rids.INTERNAL:
		return m.Internal()
	case rids.EVENT:
		return m.Event()
	case rids.GET:
		return m.Get()
	case rids.DELETE:
		return m.Delete()
	case rids.PATCH:
		return m.Patch()
	case rids.POST:
		return m.Post()
	case rids.PUT:
		return m.Put()
	default:
		panic(fmt.Sprintf("invalid %s method", method))
	}
}

type generated struct {
	key      uuid.UUID
	ctx      context.Context
	cancel   context.CancelFunc
	provider broker.Provider
	logger   Logger
	resource *ResourceBuilder
}

func (s *generated) Context() context.Context {
	return s.ctx
}

func (s *generated) Start(key uuid.UUID, ctx context.Context) error {
	s.key = key
	s.ctx, s.cancel = context.WithCancel(ctx)
	return nil
}

func (s *generated) Stop() chan bool {
	s.cancel()
	c := make(chan bool, 1)
	c <- true
	return c
}

func (s *generated) Handlers() []broker.Subscription {
	var handlers []broker.Subscription
	for _, handler := range s.resource.handlers {
		handlers = append(handlers, handler)
	}
	return handlers
}

func (s *generated) Key() uuid.UUID {
	return s.key
}

func (s *generated) Rid() rids.Resource {
	return s.resource
}

func (s *generated) Broker() broker.Provider {
	return s.provider
}

func (s *generated) Logger() Logger {
	return s.logger
}

type generatedWithMonitors struct {
	generated
}

func (s *generatedWithMonitors) Monitors() map[string][]broker.Subscription {
	return s.resource.monitors
}

type generatedWithEvents struct {
	generated
}

func (s *generatedWithEvents) Events() []broker.Event {
	return s.resource.events
}

type generatedWithMonitorsAndEvents struct {
	generated
}

func (s *generatedWithMonitorsAndEvents) Monitors() map[string][]broker.Subscription {
	return s.resource.monitors
}
func (s *generatedWithMonitorsAndEvents) Events() []broker.Event {
	return s.resource.events
}

func resourceDefinitionFromString(rid string) ResourceDefinition {
	parts := strings.Split(rid, " ")
	if len(parts) != 2 {
		panic("RIDS must be defined as METHOD PATH")
	}
	return ResourceDefinition{
		RID:    parts[1],
		Method: rids.MethodType(parts[0]),
	}
}
