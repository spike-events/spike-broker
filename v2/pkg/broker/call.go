package broker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spike-events/spike-broker/v2/pkg/rids"
	spike_utils "github.com/spike-events/spike-broker/v2/pkg/spike-utils"
	"github.com/vincent-petithory/dataurl"
)

type CallHandler func(c Call)

type Call interface {
	RawToken() string
	RawData() interface{}
	Reply() string
	Provider() Provider
	Endpoint() rids.Pattern

	ParseToken(t interface{})
	PathParam(key string) string
	ParseData(v interface{}) error
	ParseQuery(q interface{}) error

	ToJSON() json.RawMessage
	Timeout(timeout time.Duration)

	File(f *dataurl.DataURL)
	OK(result ...interface{})

	GetError() Error
	Error(err error, msg ...string)

	SetToken(token string)
	SetProvider(provider Provider)
}

// NewCall returns a Call interface instance
func NewCall(p rids.Pattern, data interface{}) Call {
	var payload []byte
	var err error
	switch data.(type) {
	case []byte:
	case nil:
	default:
		data = spike_utils.PointerFromInterface(data)
		payload, err = json.Marshal(data)
		if err != nil {
			panic(err)
		}
	}

	return &call{
		callBase: callBase{
			Data:            payload,
			EndpointPattern: p,
		},
		APIVersion: 2,
	}
}

func NewCallFromJSON(callJSON json.RawMessage, p rids.Pattern, reply string) (Call, error) {
	type apiVersion struct {
		APIVersion int `json:"apiVersion"`
	}

	var version apiVersion
	_ = json.Unmarshal(callJSON, &version)

	if version.APIVersion != 2 {
		// Version 1
		v1Call, err := NewCallV1FromJSON(callJSON, p, reply)
		if err != nil {
			return nil, err
		}
		return v1Call, nil
	}

	var c call
	err := json.Unmarshal(callJSON, &c)
	if err != nil {
		return nil, err
	}
	c.ReplyStr = reply
	return &c, nil
}

func NewHTTPCall(p rids.Pattern, token string, data interface{}, params map[string]fmt.Stringer, query interface{}) Call {
	c := NewCall(p, data).(*call)
	c.EndpointPattern.SetParams(params)
	c.EndpointPattern.Query(query)
	c.Token = token
	return c
}

// CallRequest handler
type call struct {
	callBase
	APIVersion int `json:"apiVersion"`
}

func (c *call) UnmarshalJSON(data []byte) error {
	type callInnerType struct {
		Data            interface{}     `json:"data"`
		ReplyStr        string          `json:"reply"`
		EndpointPattern json.RawMessage `json:"endpointPattern"`
		Token           string          `json:"token"`
		APIVersion      int             `json:"apiVersion"`
	}
	var callInner callInnerType
	err := json.Unmarshal(data, &callInner)
	if err != nil {
		return err
	}

	pattern, err := rids.UnmarshalPattern(callInner.EndpointPattern)
	if err != nil {
		return err
	}
	c.Data = callInner.Data
	c.ReplyStr = callInner.ReplyStr
	c.EndpointPattern = pattern
	c.Token = callInner.Token
	c.APIVersion = 2
	return nil
}

func (c *call) ToJSON() json.RawMessage {
	data, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	return data
}
