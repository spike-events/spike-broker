package testProvider

import (
	"encoding/json"
	"time"

	"github.com/hetiansu5/urlquery"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/vincent-petithory/dataurl"
)

type callRequest struct {
	p       rids.Pattern
	result  broker.Error
	token   broker.RawData
	payload broker.RawData
	okF     func(...interface{})
	errF    func(interface{})
	fileF   func(*dataurl.DataURL)
}

func (c *callRequest) GetError() broker.Error {
	return c.result
}

func (c *callRequest) RawToken() []byte {
	return c.token
}

func (c *callRequest) RawData() []byte {
	return c.payload
}

func (c *callRequest) Reply() string {
	return ""
}

func (c *callRequest) Provider() broker.Provider {
	return nil
}

func (c *callRequest) Endpoint() rids.Pattern {
	return c.p
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
	return json.Unmarshal([]byte(c.payload), v)
}

func (c *callRequest) ParseQuery(q interface{}) error {
	query := c.p.QueryParams()
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

func (c *callRequest) Timeout(timeout time.Duration) {
	//TODO implement me
	panic("implement me")
}

func (c *callRequest) File(f *dataurl.DataURL) {
	c.result = nil
	if c.fileF != nil {
		c.fileF(f)
	}
}

func (c *callRequest) OK(result ...interface{}) {
	c.result = nil
	if c.okF != nil {
		c.okF(result...)
	}
}

func (c *callRequest) InternalError(err error) {
	c.result = broker.InternalError(err)
	if c.errF != nil {
		c.errF(c.result)
	}
}

func (c *callRequest) Error(err error, msg ...string) {
	if brokerErr, ok := err.(broker.Error); ok {
		c.result = brokerErr
		if c.errF != nil {
			c.errF(c.result)
		}
		return
	}
	c.InternalError(err)
}

func (c *callRequest) NotFound() {
	c.result = broker.ErrorNotFound
	if c.errF != nil {
		c.errF(c.result)
	}
}

func (c *callRequest) SetReply(reply string) {
	//TODO implement me
	panic("implement me")
}

func (c *callRequest) SetToken(token []byte) {
	c.token = token
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
		p:       p,
		token:   token,
		payload: data,
		okF:     okF,
		errF:    errF,
		fileF:   fileF,
	}
}
