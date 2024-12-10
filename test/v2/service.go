package v2

import (
	"context"
	"encoding/json"
	"os"

	"github.com/gofrs/uuid/v5"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/service"
	"github.com/vincent-petithory/dataurl"
)

type LocalPayload struct {
	Attr1 int    `json:"attr1"`
	Attr2 string `json:"attr2"`
}

type ServiceTest struct {
	key          uuid.UUID
	ctx          context.Context
	repo         interface{}
	externalAPIs map[string]interface{}
	broker       broker.Provider
	logger       service.Logger
}

func (s *ServiceTest) SetExternalAPI(api string, implementation interface{}) error {
	if s.externalAPIs == nil {
		s.externalAPIs = make(map[string]interface{})
	}
	s.externalAPIs[api] = implementation
	return nil
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
			Resource: ServiceTestRid().RootEP(),
			Handler:  s.rootRID,
		},
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
		{
			Resource: ServiceTestRid().CallExpectingFile(),
			Handler:  s.callExpectingFile,
		},
	}
}

func (s *ServiceTest) Monitors() map[string][]broker.Subscription {
	return map[string][]broker.Subscription{
		s.Rid().Name(): {
			{
				Resource: ServiceTestRid().EventOneTest(),
				Handler:  s.handleEventOne,
			},
			{
				Resource: ServiceTestRid().EventTwoTest(),
				Handler:  s.handleEventTwo,
			},
		},
	}
}

func (s *ServiceTest) Events() []broker.Event {
	return []broker.Event{
		{
			Resource: ServiceTestRid().EventOneTest(),
			PublishValidators: []broker.AccessHandler{
				s.validatePublishEventOne,
			},
			MonitorValidators: []broker.AccessHandler{
				s.validateMonitorEventOne,
			},
		},
		{
			Resource: ServiceTestRid().EventTwoTest(),
		},
	}
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
	a.AccessGranted()
}

func (s *ServiceTest) validatePublishEventOne(a broker.Access) {
	s.logger.Printf("validated access for publishing event one")
	a.AccessGranted()
}

func (s *ServiceTest) validateMonitorEventOne(a broker.Access) {
	s.logger.Printf("validated access for monitoring event one")
	a.AccessGranted()
}

func (s *ServiceTest) rootRID(c broker.Call) {
	s.logger.Printf("accessed root RID")
	c.OK(struct {
		Field string
		value int
	}{
		Field: "field",
		value: 10,
	})
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
	err := s.Broker().Request(V2RidForV1Service().AnswerV2Service(id), &id, &rID, c.RawToken())
	if err != nil {
		c.Error(err)
		return
	}
	if rID != id {
		c.Error(broker.ErrorInvalidParams)
		return
	}
	c.OK(&rID)
}

func (s *ServiceTest) callV1Forbidden(c broker.Call) {

}

func (s *ServiceTest) callWithObjPayload(c broker.Call) {
	var payload LocalPayload
	err := json.Unmarshal(c.RawData(), &payload)
	if err != nil {
		c.Error(err)
		return
	}
	c.OK(&payload)
}

func (s *ServiceTest) callExpectingFile(c broker.Call) {
	f, err := os.ReadFile("th.webp")
	if err != nil {
		c.Error(err)
		return
	}
	data := dataurl.New(f, "image/webp", "filename", "linux.webp")
	c.File(data)
}

func (s *ServiceTest) handleEventOne(c broker.Call) {
	s.logger.Printf("event one called")
	c.OK()
}

func (s *ServiceTest) handleEventTwo(c broker.Call) {
	s.logger.Printf("event two called")
	c.OK()
}

func NewServiceTest(broker broker.Provider, logger service.Logger) service.Service {
	return &ServiceTest{
		broker: broker,
		logger: logger,
	}
}
