package testProvider

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/vincent-petithory/dataurl"
)

type RequestMock func(c broker.Call)
type Subscription struct {
	sub     broker.Subscription
	handler broker.ServiceHandler
}

type Mocks struct {
	Requests     map[string]RequestMock
	Publishes    map[string]RequestMock
	ExternalAPIs map[string]interface{}
	Repository   interface{}
}

type ResponseHandlers struct {
	OkF   func(...interface{})
	ErrF  func(interface{})
	FileF func(*dataurl.DataURL)
}

type Provider interface {
	broker.Provider
	SetMocks(mocks Mocks)
	SetResponseHandlers(handlers ResponseHandlers)
}

type testProvider struct {
	base          broker.Provider
	ctx           context.Context
	mocks         Mocks
	respHandlers  ResponseHandlers
	subscriptions map[string]Subscription
	monitors      map[string]map[string]Subscription
}

func (t *testProvider) SetResponseHandlers(handlers ResponseHandlers) {
	t.respHandlers = handlers
}

func (t *testProvider) Subscribe(s broker.Subscription, handler broker.ServiceHandler) (func(), broker.Error) {
	return t.base.Subscribe(s, handler)
}

func (t *testProvider) Monitor(monitoringGroup string, s broker.Subscription, handler broker.ServiceHandler,
	token ...[]byte) (func(), broker.Error) {
	return t.base.Monitor(monitoringGroup, s, handler, token...)
}

func (t *testProvider) Get(p rids.Pattern, rs interface{}, token ...[]byte) broker.Error {
	return t.base.Get(p, rs, token...)
}

func (t *testProvider) Request(p rids.Pattern, payload interface{}, rs interface{}, token ...[]byte) broker.Error {
	return t.base.Request(p, payload, rs, token...)
}

func (t *testProvider) Publish(p rids.Pattern, payload interface{}, token ...[]byte) broker.Error {
	return t.base.Publish(p, payload, token...)
}

func (t *testProvider) Reply(replyEndpoint string, payload []byte) broker.Error {
	return t.base.Reply(replyEndpoint, payload)
}

func (t *testProvider) SetMocks(mocks Mocks) {
	t.mocks = mocks
}

var testProviderImpl *testProvider

// NewTestProvider builds the default test infrastructure provider. It implements the broker.Provider interface and
// some helpers to allow unit testing broker.Subscribe
func NewTestProvider(ctx context.Context) Provider {
	if testProviderImpl == nil {
		testProviderImpl = &testProvider{
			ctx: ctx,
			mocks: Mocks{
				Requests:  make(map[string]RequestMock),
				Publishes: make(map[string]RequestMock),
			},
			subscriptions: make(map[string]Subscription),
			monitors:      make(map[string]map[string]Subscription),
		}
		base := broker.NewSpecific(testProviderImpl)
		testProviderImpl.base = base
	}
	return testProviderImpl
}

/* Provider Inteface */

func (t *testProvider) requestMocked(
	mp map[string]RequestMock,
	data []byte,
	overrideTimeout ...time.Duration,
) ([]byte, broker.Error) {
	var c callRequest
	err := json.Unmarshal(data, &c)
	if err != nil {
		return nil, broker.InternalError(err)
	}

	if f, ok := mp[c.Endpoint().EndpointName()]; ok {
		f(&c)
		var response json.RawMessage
		rErr := c.GetError()
		if rErr != nil {
			return nil, rErr
		}
		response, err = json.Marshal(&c.result)
		if err != nil {
			panic(err)
		}

		return response, nil
	}
	return nil, broker.ErrorServiceUnavailable
}

func (t *testProvider) RequestRaw(_ string, data []byte, overrideTimeout ...time.Duration) ([]byte, broker.Error) {
	return t.requestMocked(t.mocks.Requests, data, overrideTimeout...)
}

func (t *testProvider) PublishRaw(_ string, data []byte) broker.Error {
	_, err := t.requestMocked(t.mocks.Publishes, data)
	return err
}

func (t *testProvider) SubscribeRaw(s broker.Subscription, _ string, handler broker.ServiceHandler) (func(), broker.Error) {
	t.subscribe(t.subscriptions, Subscription{
		sub:     s,
		handler: handler,
	})
	return func() { log.Printf("unsubscribed %s", s.Resource.EndpointNameSpecific()) }, nil
}

func (t *testProvider) Close() {}

func (t *testProvider) NewCall(p rids.Pattern, payload interface{}) broker.Call {
	return NewCall(p, payload, nil, t.respHandlers.OkF, t.respHandlers.ErrF, t.respHandlers.FileF)
}

func (t *testProvider) subscribe(m map[string]Subscription, s Subscription) {
	m[s.sub.Resource.EndpointName()] = s
}
