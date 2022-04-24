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
	result  *broker.Error
	token   string
	payload interface{}
	okF     func(interface{})
	errF    func(interface{})
}

func (c callRequest) GetError() broker.Error {
	//TODO implement me
	panic("implement me")
}

func (c callRequest) RawToken() string {
	return c.token
}

func (c callRequest) RawData() []byte {
	data, err := json.Marshal(c.payload)
	if err != nil {
		panic("invalid data on marshal")
	}
	return data
}

func (c callRequest) Reply() string {
	return ""
}

func (c callRequest) Provider() broker.Provider {
	return nil
}

func (c callRequest) Endpoint() rids.Pattern {
	return c.p
}

func (c callRequest) ParseToken(t interface{}) {
	token := c.token
	if len(token) > 0 {
		err := json.Unmarshal([]byte(token), &t)
		if err != nil {
			panic("invalid token on unmarshal")
		}
	}
}

func (c callRequest) PathParam(key string) string {
	params := c.Endpoint().Params()
	if params == nil {
		return ""
	}
	if _, ok := params[key]; !ok {
		return ""
	}
	return params[key].String()
}

func (c callRequest) ParseData(v interface{}) error {
	data := c.RawData()
	return json.Unmarshal(data, v)
}

func (c callRequest) ParseQuery(q interface{}) error {
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

func (c callRequest) ToJSON() []byte {
	data, _ := json.Marshal(c)
	return data
}

func (c callRequest) FromJSON(data json.RawMessage, provider broker.Provider, reply string) error {
	//TODO implement me
	panic("implement me")
}

func (c callRequest) Timeout(timeout time.Duration) {
	//TODO implement me
	panic("implement me")
}

func (c callRequest) File(f *dataurl.DataURL) {
	//TODO implement me
	panic("implement me")
}

func (c callRequest) OK(result ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (c callRequest) InternalError(err error) {
	//TODO implement me
	panic("implement me")
}

func (c callRequest) Error(err broker.Error, msg ...string) {
	//TODO implement me
	panic("implement me")
}

func (c callRequest) NotFound() {
	//TODO implement me
	panic("implement me")
}

func (c callRequest) SetReply(reply string) {
	//TODO implement me
	panic("implement me")
}

func (c callRequest) SetToken(token string) {
	//TODO implement me
	panic("implement me")
}

func (c callRequest) SetProvider(provider broker.Provider) {
	//TODO implement me
	panic("implement me")
}

func (c callRequest) SetEndpoint(p rids.Pattern) {
	//TODO implement me
	panic("implement me")
}

func NewCall(p rids.Pattern, payload interface{}, token string, okF func(interface{}), errF func(interface{})) broker.Call {
	return &callRequest{}
}
