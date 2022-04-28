package test

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/service"
)

type ServiceTest struct {
	key  uuid.UUID
	ctx  context.Context
	conf service.Config
}

func (s ServiceTest) SetConfig(config service.Config) error {
	s.conf = config
	return nil
}

func (s ServiceTest) Start(key uuid.UUID, ctx context.Context) error {
	s.key = key
	s.ctx = ctx
	return nil
}

func (s ServiceTest) Stop() chan bool {
	return make(chan bool, 1)
}

func (s ServiceTest) Handlers() []broker.Subscription {
	return []broker.Subscription{
		{
			Resource:   ServiceTestRid().TestReply(),
			Handler:    s.reply,
			Validators: []broker.AccessHandler{s.validateReply},
		},
	}
}

func (s ServiceTest) Monitors() map[string]broker.Subscription {
	return nil
}

func (s ServiceTest) Key() uuid.UUID {
	return s.key
}

func (s ServiceTest) Rid() rids.Resource {
	return ServiceTestRid()
}

func (s ServiceTest) Broker() broker.Provider {
	return s.conf.Broker
}

func (s ServiceTest) Repository() service.Repository {
	return s.conf.Repository
}

func (s ServiceTest) Logger() service.Logger {
	return s.conf.Logger
}

func (s *ServiceTest) validateReply(a broker.Access) {
	if a.RawToken() == "" {
		a.AccessDenied()
		return
	}
	a.OK()
}

func (s *ServiceTest) reply(c broker.Call) {
	id, err := uuid.FromString(c.PathParam("ID"))
	if err != nil {
		c.InternalError(err)
		return
	}
	c.OK(&id)
}

func NewServiceTest() service.Service {
	return &ServiceTest{}
}
