package testProvider

import (
	"context"
	"encoding/json"
	"time"

	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

type RequestMock func(p rids.Pattern, payload interface{}, res interface{}, token ...json.RawMessage) broker.Error
type RequestRawMock func(payload json.RawMessage, overrideTimeout ...time.Duration) (json.RawMessage, broker.Error)

type Mocks struct {
	Requests     map[string]RequestMock
	RequestsRaw  map[string]RequestRawMock
	Publishes    map[string]RequestMock
	PublishesRaw map[string]RequestRawMock
}

type Provider interface {
	broker.Provider
	SetMocks(mocks Mocks)
}

type testProvider struct {
	ctx context.Context

	mocks Mocks

	subscriptions map[string]broker.Subscription
	monitors      map[string]map[string]broker.Subscription
}

func (t *testProvider) SetHandler(_ string, _ broker.ServiceHandler) {
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
				Requests:     make(map[string]RequestMock),
				RequestsRaw:  make(map[string]RequestRawMock),
				Publishes:    make(map[string]RequestMock),
				PublishesRaw: make(map[string]RequestRawMock),
			},
			subscriptions: make(map[string]broker.Subscription),
			monitors:      make(map[string]map[string]broker.Subscription),
		}
	}
	return testProviderImpl
}

/* Provider Inteface */

func (t *testProvider) request(mp map[string]RequestMock, p rids.Pattern, payload interface{}, rs interface{}, token ...json.RawMessage) broker.Error {
	if f, ok := mp[p.EndpointName()]; ok {
		return f(p, payload, rs, token...)
	}
	return broker.ErrorServiceUnavailable
}

func (t *testProvider) requestRaw(
	mp map[string]RequestRawMock,
	subject string,
	data json.RawMessage,
	overrideTimeout ...time.Duration,
) (json.RawMessage, broker.Error) {
	if f, ok := mp[subject]; ok {
		return f(data, overrideTimeout...)
	}
	return nil, broker.ErrorServiceUnavailable
}

func (t *testProvider) Request(p rids.Pattern, payload interface{}, rs interface{}, token ...json.RawMessage) broker.Error {
	return t.request(t.mocks.Requests, p, payload, rs, token...)
}

func (t *testProvider) RequestRaw(subject string, data json.RawMessage, overrideTimeout ...time.Duration) (json.RawMessage, broker.Error) {
	return t.requestRaw(t.mocks.RequestsRaw, subject, data, overrideTimeout...)
}

func (t *testProvider) Publish(p rids.Pattern, payload interface{}, token ...json.RawMessage) error {
	return t.request(t.mocks.Publishes, p, payload, nil, token...)
}

func (t *testProvider) PublishRaw(subject string, data json.RawMessage) error {
	_, err := t.requestRaw(t.mocks.PublishesRaw, subject, data)
	return err
}

func (t *testProvider) Get(p rids.Pattern, rs interface{}, token ...json.RawMessage) broker.Error {
	return t.Request(p, nil, rs, token...)
}

// Provider Interface not supported

func (t *testProvider) Close() {}

func (t *testProvider) subscribe(m map[string]broker.Subscription, s broker.Subscription) {
	m[s.Resource.EndpointName()] = s
}

func (t *testProvider) Subscribe(s broker.Subscription) (interface{}, error) {
	t.subscribe(t.subscriptions, s)
	return t, nil
}

func (t *testProvider) SubscribeAll(s broker.Subscription) (interface{}, error) {
	t.subscribe(t.subscriptions, s)
	return t, nil
}

func (t *testProvider) Monitor(monitoringGroup string, s broker.Subscription) (func(), error) {
	var ok bool
	var m map[string]broker.Subscription
	if m, ok = t.monitors[monitoringGroup]; !ok {
		m = make(map[string]broker.Subscription)
	}
	t.subscribe(m, s)
	return func() {}, nil
}
