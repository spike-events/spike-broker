package broker

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"sync"

	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

func NewSpecific(impl SpecificProvider) Provider {
	return &specificProviderBase{
		impl: impl,
	}
}

type specificProviderBase struct {
	impl SpecificProvider
	m    sync.Mutex
}

func (s *specificProviderBase) Close() {
	s.impl.Close()
}

func (s *specificProviderBase) Subscribe(sub Subscription, handler ServiceHandler) (func(), Error) {
	s.m.Lock()
	defer s.m.Unlock()
	return s.impl.SubscribeRaw(sub, sub.Resource.Service(), handler)
}

func (s *specificProviderBase) Monitor(monitoringGroup string, sub Subscription, handler ServiceHandler,
	token ...[]byte) (func(), Error) {
	s.m.Lock()
	defer s.m.Unlock()

	if sub.Resource.Version() > 1 {
		if sub.Resource.Method() != rids.EVENT {
			return nil, InternalError(fmt.Errorf("invalid RID, can only monitor Event Methods"))
		}

		if len(token) > 0 {
			subEncoded, err := json.Marshal(sub.Resource)
			if err != nil {
				return nil, InternalError(err)
			}
			p, err := rids.NewPatternFromString(fmt.Sprintf("%s.validateMonitor", sub.Resource.Service()), rids.INTERNAL)
			if err != nil {
				return nil, InternalError(err)
			}
			rErr := s.Request(p, subEncoded, nil, token...)
			if rErr != nil && rErr.Code() != ErrorServiceUnavailable.Code() {
				// FIXME: Ignore Service Unavailable (compatible with V1)
				return nil, rErr
			}
		}
	}
	return s.impl.SubscribeRaw(sub, monitoringGroup, handler)
}

func (s *specificProviderBase) Get(p rids.Pattern, rs interface{}, token ...[]byte) Error {
	return s.Request(p, nil, rs, token...)
}

func (s *specificProviderBase) Request(p rids.Pattern, payload interface{}, rs interface{}, token ...[]byte) Error {
	c := s.impl.NewCall(p, payload)
	if len(token) > 0 && len(token[0]) > 0 {
		c.SetToken(token[0])
	}

	// Check dependencies
	result, rErr := s.impl.RequestRaw(p.EndpointName(), c.ToJSON())
	if rErr != nil {
		return rErr
	}

	bMsg := NewMessageFromJSON(result)
	if bMsg == nil {
		return NewError("invalid payload", http.StatusInternalServerError, result)
	}

	if math.Abs(float64(bMsg.Code()-http.StatusOK)) >= 100 {
		return bMsg
	}

	switch rs.(type) {
	case nil:
		return nil
	case *RawData:
		rawData := bMsg.Data()
		rv := reflect.ValueOf(rs)
		ro := reflect.ValueOf(rawData)
		rv.Elem().Set(ro)
	case *[]byte:
		if err := json.Unmarshal(bMsg.Data(), rs); err != nil {
			tmp := []byte(bMsg.Data())
			tmp, err = json.Marshal(tmp)
			if err != nil {
				return InternalError(err)
			}
			if err = json.Unmarshal(tmp, rs); err != nil {
				return InternalError(err)
			}
			return nil
		}
	default:
		if err := json.Unmarshal(bMsg.Data(), rs); err != nil {
			return InternalError(err)
		}
	}
	return nil
}

func (s *specificProviderBase) Publish(p rids.Pattern, payload interface{}, token ...[]byte) Error {
	c := s.impl.NewCall(p, payload)
	callMetaData := s.impl.NewCall(p, nil)
	if len(token) > 0 && len(token[0]) > 0 {
		c.SetToken(token[0])
		callMetaData.SetToken(token[0])
	}

	// When publishing on V1 calls we simply ignore all validations
	if p.Version() > 1 {

		// We must check if the token has permission to publish
		if p.Method() != rids.EVENT {
			return InternalError(fmt.Errorf("invalid RID, can only publish on Event Methods"))
		}

		// Validate the request on remote service when there's a token
		if len(token) > 0 {
			subEncoded, err := json.Marshal(p)
			if err != nil {
				return InternalError(err)
			}
			vp, err := rids.NewPatternFromString(fmt.Sprintf("%s.validatePublish", p.Service()), rids.INTERNAL)
			if err != nil {
				return InternalError(err)
			}
			rErr := s.Request(vp, subEncoded, nil, token...)
			if rErr != nil && rErr.Code() != ErrorServiceUnavailable.Code() {
				// FIXME: Ignore Service Unavailable (compatible with V1)
				return rErr
			}
		}
	}

	return s.impl.PublishRaw(p.EndpointNameSpecific(), c.ToJSON())
}

func (s *specificProviderBase) Reply(ep string, payload []byte) Error {
	return s.impl.PublishRaw(ep, payload)
}
