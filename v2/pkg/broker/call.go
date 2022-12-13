package broker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hetiansu5/urlquery"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	spikeUtils "github.com/spike-events/spike-broker/v2/pkg/spike-utils"
	"github.com/vincent-petithory/dataurl"
)

type CallHandler func(c Call)

type Call interface {
	RawToken() json.RawMessage
	RawData() json.RawMessage
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

	SetToken(token json.RawMessage)
	SetProvider(provider Provider)
}

// NewCall returns a Call interface instance
func NewCall(p rids.Pattern, data interface{}) Call {
	var payload []byte
	var err error
	switch data.(type) {
	case nil:
	case []byte:
		payload = data.([]byte)
	default:
		data = spikeUtils.PointerFromInterface(data)
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

func NewHTTPCall(p rids.Pattern, token json.RawMessage, data interface{}, params map[string]fmt.Stringer, query interface{}) Call {
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
		Data            json.RawMessage `json:"data"`
		ReplyStr        string          `json:"reply"`
		EndpointPattern json.RawMessage `json:"endpointPattern"`
		Token           json.RawMessage `json:"token"`
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
	type callV1Compatible struct {
		callBase
		Params     map[string]string `json:"Params"`
		TokenV1    json.RawMessage   `json:"Token"`
		QueryV1    string            `json:"Query"`
		APIVersion int               `json:"apiVersion"`
	}

	params := make(map[string]string)
	for i, param := range c.EndpointPattern.Params() {
		params[i] = param.String()
	}

	query := c.EndpointPattern.QueryParams()
	var queryStr string
	switch query.(type) {
	case string:
		queryStr = query.(string)
	case nil:
	default:
		if q, err := urlquery.Marshal(query); err == nil {
			queryStr = string(q)
		}
	}

	toSend := &callV1Compatible{
		callBase:   c.callBase,
		Params:     params,
		TokenV1:    c.Token,
		QueryV1:    queryStr,
		APIVersion: c.APIVersion,
	}
	//switch toSend.Data.(type) {
	//case []byte:
	//	toSend.Data = string(toSend.Data.([]byte))
	//}
	data, err := json.Marshal(toSend)
	if err != nil {
		panic(err)
	}
	return data
}
