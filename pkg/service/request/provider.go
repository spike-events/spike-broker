package request

import (
	"encoding/json"
	"time"

	"github.com/spike-events/spike-broker/pkg/rids"
)

type Provider interface {
	Close()
	Subscribe(p *rids.Pattern, hc func(msg *CallRequest), access ...func(msg *AccessRequest)) (interface{}, error)
	SubscribeAll(p *rids.Pattern, hc func(msg *CallRequest), access ...func(msg *AccessRequest)) (interface{}, error)
	Monitor(monitoringGroup string, p *rids.Pattern, hc func(msg *CallRequest), access ...func(msg *AccessRequest)) (func(), error)
	RegisterMonitor(p *rids.Pattern) error
	Publish(p *rids.Pattern, payload *CallRequest, token ...string) error
	PublishRaw(subject string, data json.RawMessage) error
	Get(p *rids.Pattern, rs interface{}, token ...string) *ErrorRequest
	Request(p *rids.Pattern, payload *CallRequest, rs interface{}, token ...string) *ErrorRequest
	RequestRaw(subject string, data json.RawMessage, overrideTimeout ...time.Duration) (json.RawMessage, *ErrorRequest)
}
