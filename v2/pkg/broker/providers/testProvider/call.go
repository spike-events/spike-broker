package testProvider

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/hetiansu5/urlquery"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	spikeutils "github.com/spike-events/spike-broker/v2/pkg/spike-utils"
	"github.com/vincent-petithory/dataurl"
)

type callRequest struct {
	Pattern rids.Pattern `json:"pattern"`
	error   broker.Error
	result  broker.Message
	Token   broker.RawData `json:"token"`
	Payload broker.RawData `json:"payload"`
	okF     func(...interface{})
	errF    func(interface{})
	fileF   func(*dataurl.DataURL)
}

func (c *callRequest) UnmarshalJSON(data []byte) error {
	type callInnerType struct {
		Pattern json.RawMessage `json:"pattern"`
		Token   broker.RawData  `json:"token"`
		Payload broker.RawData  `json:"payload"`
	}
	var callInner callInnerType
	err := json.Unmarshal(data, &callInner)
	if err != nil {
		return err
	}

	pattern, err := rids.UnmarshalPattern(callInner.Pattern)
	if err != nil {
		return err
	}
	c.Payload = callInner.Payload
	c.Pattern = pattern
	c.Token = callInner.Token
	return nil
}

func (c *callRequest) GetError() broker.Error {
	return c.error
}

func (c *callRequest) RawToken() []byte {
	return c.Token
}

func (c *callRequest) RawData() []byte {
	return c.Payload
}

func (c *callRequest) Reply() string {
	return ""
}

func (c *callRequest) Provider() broker.Provider {
	return nil
}

func (c *callRequest) Endpoint() rids.Pattern {
	return c.Pattern
}

func (c *callRequest) PathParam(key string) string {
	params := c.Endpoint().Params()
	if params == nil {
		return ""
	}
	if _, ok := params[key]; !ok {
		return ""
	}
	return params[key].String()
}

func (c *callRequest) ParseData(v interface{}) error {
	return json.Unmarshal([]byte(c.Payload), v)
}

func (c *callRequest) ParseQuery(q interface{}) error {
	query := c.Pattern.QueryParams()
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

func (c *callRequest) ToJSON() json.RawMessage {
	data, _ := json.Marshal(c)
	return data
}

func (c *callRequest) Timeout(_ time.Duration) {
	// Do nothing on Timeout call
}

func (c *callRequest) File(f *dataurl.DataURL) {
	c.error = nil
	if c.fileF != nil {
		c.fileF(f)
	}
	var success broker.Message
	success.CodeInt = http.StatusOK
	payload, err := json.Marshal(f)
	if err != nil {
		panic(err)
	}
	success.DataIface = payload
	c.result = success
}

func (c *callRequest) OK(result ...interface{}) {
	c.error = nil
	if c.okF != nil {
		c.okF(result...)
	}
	var success broker.Message
	success.CodeInt = http.StatusOK

	if len(result) > 0 {
		switch result[0].(type) {
		case string:
			success.DataIface = []byte(result[0].(string))
		case []byte:
			success.DataIface = result[0].([]byte)
		default:
			// Make sure we always marshal pointer structures
			result[0] = spikeutils.PointerFromInterface(result[0])
			encoded, err := json.Marshal(result[0])
			if err != nil {
				panic(err)
			}
			success.DataIface = encoded
		}
	}
	c.result = success
}

func (c *callRequest) InternalError(err error) {
	c.error = broker.InternalError(err)
	if c.errF != nil {
		c.errF(c.error)
	}
}

func (c *callRequest) Error(err error, msg ...string) {
	if brokerErr, ok := err.(broker.Error); ok {
		c.error = brokerErr
		if c.errF != nil {
			c.errF(c.error)
		}
		return
	}
	c.InternalError(err)
}

func (c *callRequest) NotFound() {
	c.error = broker.ErrorNotFound
	if c.errF != nil {
		c.errF(c.error)
	}
}

func (c *callRequest) SetReply(reply string) {
	//TODO implement me
	panic("implement me")
}

func (c *callRequest) SetToken(token []byte) {
	c.Token = token
}

func (c *callRequest) SetProvider(provider broker.Provider) {
	//TODO implement me
	panic("implement me")
}

func (c *callRequest) SetEndpoint(p rids.Pattern) {
	//TODO implement me
	panic("implement me")
}

func NewCall(p rids.Pattern, payload interface{}, token []byte,
	okF func(...interface{}), errF func(interface{}), fileF func(*dataurl.DataURL)) broker.Call {
	if okF == nil {
		okF = func(...interface{}) {}
	}
	if errF == nil {
		errF = func(interface{}) {}
	}
	if fileF == nil {
		fileF = func(*dataurl.DataURL) {}
	}
	data, _ := json.Marshal(payload)
	return &callRequest{
		Pattern: p,
		Token:   token,
		Payload: data,
		okF:     okF,
		errF:    errF,
		fileF:   fileF,
	}
}
