package testProvider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hetiansu5/urlquery"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	spikeutils "github.com/spike-events/spike-broker/v2/pkg/spike-utils"
	"github.com/vincent-petithory/dataurl"
)

type callRequestMock struct {
	Data            broker.RawData `json:"Data"`
	ReplyStr        string         `json:"reply"`
	EndpointPattern rids.Pattern   `json:"endpointPattern"`
	Token           broker.RawData `json:"token"`
	errorResult     broker.Error
	successResult   broker.Message
	token           broker.RawData
}

func (c *callRequestMock) UnmarshalJSON(data []byte) error {
	type callInnerType struct {
		Data            broker.RawData  `json:"data"`
		ReplyStr        string          `json:"reply"`
		EndpointPattern json.RawMessage `json:"endpointPattern"`
		Token           broker.RawData  `json:"token"`
		TokenV1         string          `json:"Token"`
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
	return nil
}

func (c *callRequestMock) GetError() broker.Error {
	return c.errorResult
}

func (c *callRequestMock) RawToken() []byte {
	return c.token
}

func (c *callRequestMock) RawData() []byte {
	return c.Data
}

func (c *callRequestMock) Reply() string {
	return ""
}

func (c *callRequestMock) Provider() broker.Provider {
	panic("not implemented")
}

func (c *callRequestMock) Endpoint() rids.Pattern {
	return c.EndpointPattern
}

func (c *callRequestMock) PathParam(key string) string {
	params := c.Endpoint().Params()
	if params == nil {
		return ""
	}
	if _, ok := params[key]; !ok {
		return ""
	}
	return params[key].String()
}

func (c *callRequestMock) ParseQuery(q interface{}) error {
	query := c.Endpoint().QueryParams()
	switch query.(type) {
	case string:
		return urlquery.Unmarshal([]byte(query.(string)), q)
	case nil:
		return nil
	default:
		data, err := json.Marshal(query)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, q)
	}
}

func (c *callRequestMock) ToJSON() json.RawMessage {
	data, _ := json.Marshal(c)
	return data
}

func (c *callRequestMock) Timeout(_ time.Duration) {
	panic("not implemented, if you needed this please open a bug report")
}

func (c *callRequestMock) File(f *dataurl.DataURL) {
	c.errorResult = nil
	var success broker.Message
	success.CodeInt = http.StatusOK
	payload, err := json.Marshal(f)
	if err != nil {
		panic(err)
	}
	success.DataIface = payload
	c.successResult = success
}

func (c *callRequestMock) OK(result ...interface{}) {
	c.errorResult = nil
	var success broker.Message
	success.CodeInt = http.StatusOK

	if len(result) > 0 {
		switch result[0].(type) {
		case string:
			success.DataIface = []byte(result[0].(string))
			return
		case []byte:
			success.DataIface = result[0].([]byte)
			return
		}

		// Make sure we always marshal pointer structures
		result[0] = spikeutils.PointerFromInterface(result[0])
		encoded, err := json.Marshal(result[0])
		if err != nil {
			panic(err)
		}
		success.DataIface = encoded
	}
	c.successResult = success
}

func (c *callRequestMock) InternalError(err error) {
	c.errorResult = broker.InternalError(err)
}

func (c *callRequestMock) Error(err error, msg ...string) {
	if brokerErr, ok := err.(broker.Error); ok {
		c.errorResult = brokerErr
		return
	}
	c.InternalError(err)
}

func (c *callRequestMock) SetToken(token []byte) {
	c.token = token
}

func (c *callRequestMock) SetProvider(provider broker.Provider) {
	//TODO implement me
	panic("implement me")
}

func NewCallFromRequestMock(data json.RawMessage) *callRequestMock {
	var c callRequestMock
	err := json.Unmarshal(data, &c)
	if err != nil {
		panic(fmt.Sprintf("invalid call data: %s", err))
	}
	return &c
}
