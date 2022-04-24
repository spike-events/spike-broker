package testProvider

import (
	"context"
	"encoding/json"
	"time"

	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

type RequestMock func(p rids.Pattern, payload interface{}, res interface{}, token ...string) broker.Error
type RequestRawMock func(payload json.RawMessage, overrideTimeout ...time.Duration) (json.RawMessage, broker.Error)

type Mocks struct {
	requests     map[string]RequestMock
	requestsRaw  map[string]RequestRawMock
	publishes    map[string]RequestMock
	publishesRaw map[string]RequestRawMock
}

type Provider interface {
	broker.Provider
	SetMocks(mocks Mocks)
}

type testProvider struct {
	ctx    context.Context
	cancel context.CancelFunc

	mocks Mocks

	subscriptions map[string]broker.Subscription
	monitors      map[string]map[string]broker.Subscription
}

func (t *testProvider) SetHandler(_ func(p rids.Pattern, payload []byte)) {}

func (t *testProvider) SetMocks(mocks Mocks) {
	t.mocks = mocks
}

var testProviderImpl *testProvider

// NewTestProvider builds the default test infrastructure provider. It implements the broker.Provider interface and
// some helpers to allow unit testing broker.Subscribe
func NewTestProvider(ctx context.Context, cancel context.CancelFunc) *testProvider {
	if testProviderImpl == nil {
		testProviderImpl = &testProvider{
			ctx:    ctx,
			cancel: cancel,
			mocks: Mocks{
				requests:     make(map[string]RequestMock),
				requestsRaw:  make(map[string]RequestRawMock),
				publishes:    make(map[string]RequestMock),
				publishesRaw: make(map[string]RequestRawMock),
			},
			subscriptions: make(map[string]broker.Subscription),
			monitors:      make(map[string]map[string]broker.Subscription),
		}
	}
	return testProviderImpl
}

/* Provider Inteface */

func (t *testProvider) request(mp map[string]RequestMock, p rids.Pattern, payload interface{}, rs interface{}, token ...string) broker.Error {
	if f, ok := mp[p.EndpointName()]; ok {
		return f(p, payload, rs, token...)
	}
	return broker.ErrorNotFound
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
	return nil, broker.ErrorNotFound
}

func (t *testProvider) Request(p rids.Pattern, payload interface{}, rs interface{}, token ...string) broker.Error {
	return t.request(t.mocks.requests, p, payload, rs, token...)
}

func (t *testProvider) RequestRaw(subject string, data json.RawMessage, overrideTimeout ...time.Duration) (json.RawMessage, broker.Error) {
	return t.requestRaw(t.mocks.requestsRaw, subject, data, overrideTimeout...)
}

func (t *testProvider) Publish(p rids.Pattern, payload interface{}, token ...string) error {
	return t.request(t.mocks.publishes, p, payload, nil, token...)
}

func (t *testProvider) PublishRaw(subject string, data json.RawMessage) error {
	_, err := t.requestRaw(t.mocks.publishesRaw, subject, data)
	return err
}

func (t *testProvider) Get(p rids.Pattern, rs interface{}, token ...string) broker.Error {
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

func (t *testProvider) Monitor(monitoringGroup string, s broker.Subscription) (interface{}, error) {
	var ok bool
	var m map[string]broker.Subscription
	if m, ok = t.monitors[monitoringGroup]; !ok {
		m = make(map[string]broker.Subscription)
	}
	t.subscribe(m, s)
	return t, nil
}
