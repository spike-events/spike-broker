package v2

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/service"
)

type LocalPayload struct {
	Attr1 int    `json:"attr1"`
	Attr2 string `json:"attr2"`
}

type ServiceTest struct {
	key    uuid.UUID
	ctx    context.Context
	repo   interface{}
	broker broker.Provider
	logger service.Logger
}

func (s *ServiceTest) SetRepository(repo interface{}) error {
	s.repo = repo
	return nil
}

func (s *ServiceTest) Start(key uuid.UUID, ctx context.Context) error {
	s.key = key
	s.ctx = ctx
	return nil
}

func (s *ServiceTest) Stop() chan bool {
	c := make(chan bool, 1)
	c <- true
	return c
}

func (s *ServiceTest) Handlers() []broker.Subscription {
	return []broker.Subscription{
		{
			Resource:   ServiceTestRid().TestReply(),
			Handler:    s.reply,
			Validators: []broker.AccessHandler{s.validateReply},
		},
		{
			Resource:   ServiceTestRid().FromMock(),
			Handler:    s.fromMock,
			Validators: nil,
		},
		{
			Resource:   ServiceTestRid().CallV1(),
			Handler:    s.callV1,
			Validators: nil,
		},
		{
			Resource: ServiceTestRid().CallV1Forbidden(),
			Handler:  s.callV1Forbidden,
		},
		{
			Resource: ServiceTestRid().CallWithObjPayload(),
			Handler:  s.callWithObjPayload,
		},
	}
}

func (s *ServiceTest) Monitors() map[string]broker.Subscription {
	return nil
}

func (s *ServiceTest) Key() uuid.UUID {
	return s.key
}

func (s *ServiceTest) Rid() rids.Resource {
	return ServiceTestRid()
}

func (s *ServiceTest) Broker() broker.Provider {
	return s.broker
}

func (s *ServiceTest) Logger() service.Logger {
	return s.logger
}

func (s *ServiceTest) validateReply(a broker.Access) {
	if len(a.RawToken()) == 0 {
		a.AccessDenied()
		return
	}
	a.OK()
}

func (s *ServiceTest) reply(c broker.Call) {
	id, err := uuid.FromString(c.PathParam("ID"))
	if err != nil {
		c.Error(err)
		return
	}
	c.OK(&id)
}

func (s *ServiceTest) fromMock(c broker.Call) {
	id, _ := uuid.NewV4()
	var replyID uuid.UUID
	err := s.Broker().Request(ServiceTestRid().TestReply(id), id, &replyID, c.RawToken())
	if err != nil {
		c.Error(err)
		return
	}

	c.OK(&replyID)
}

func (s *ServiceTest) callV1(c broker.Call) {
	id, _ := uuid.NewV4()
	var rID uuid.UUID
	err := s.Broker().Request(V2RidForV1Service().AnswerV2Service(id), &id, &rID, json.RawMessage("\"token-string\""))
	if err != nil {
		c.Error(err)
		return
	}
	c.OK(&rID)
}

func (s *ServiceTest) callV1Forbidden(c broker.Call) {

}

func (s *ServiceTest) callWithObjPayload(c broker.Call) {
	var payload LocalPayload
	err := c.ParseData(&payload)
	if err != nil {
		c.Error(err)
		return
	}
	c.OK(&payload)
}

func NewServiceTest(broker broker.Provider, logger service.Logger) service.Service {
	return &ServiceTest{
		broker: broker,
		logger: logger,
	}
}
