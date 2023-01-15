package broker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hetiansu5/urlquery"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	spikeutils "github.com/spike-events/spike-broker/v2/pkg/spike-utils"
	"github.com/vincent-petithory/dataurl"
)

type callBase struct {
	Data            RawData      `json:"Data"` // As V1 uses upper case on JSON notation we keep data this way
	ReplyStr        string       `json:"reply"`
	EndpointPattern rids.Pattern `json:"endpointPattern"`
	Token           RawData      `json:"token"`
	provider        Provider
	err             Error
}

func (c *callBase) Endpoint() rids.Pattern {
	return c.EndpointPattern
}

func (c *callBase) Provider() Provider {
	return c.provider
}

func (c *callBase) Reply() string {
	return c.ReplyStr
}

func (c *callBase) RawData() []byte {
	return c.Data
}

func (c *callBase) SetProvider(provider Provider) {
	c.provider = provider
}

func (c *callBase) SetToken(token []byte) {
	c.Token = token
}

func (c *callBase) RawToken() []byte {
	return c.Token
}

// PathParam retorna parametro map string
func (c *callBase) PathParam(key string) string {
	params := c.Endpoint().Params()
	if params == nil {
		return ""
	}
	if _, ok := params[key]; !ok {
		return ""
	}
	return params[key].String()
}

func (c *callBase) FromJSON(data json.RawMessage, provider Provider, reply string) error {
	if len(data) > 0 {
		err := json.Unmarshal(data, c)
		if err != nil {
			return err
		}
	}
	var ok bool
	c.provider, ok = provider.(Provider)
	if !ok {
		return fmt.Errorf("invalid provider %v", provider)
	}
	c.ReplyStr = reply
	return nil
}

func (c *callBase) File(f *dataurl.DataURL) {
	if c.ReplyStr == "" {
		return
	}

	var success Message
	success.CodeInt = http.StatusOK
	payload, err := json.Marshal(f)
	if err != nil {
		panic(err)
	}
	success.DataIface = payload
	data, err := json.Marshal(&success)
	if err != nil {
		panic(err)
	}
	err = c.provider.Reply(c.ReplyStr, data)
}

func (c *callBase) OK(result ...interface{}) {
	if c.ReplyStr == "" {
		return
	}

	var success Message
	success.CodeInt = http.StatusOK

	if len(result) == 0 {
		data, err := json.Marshal(&success)
		if err != nil {
			panic(err)
		}
		c.provider.Reply(c.ReplyStr, data) // FIXME: Log or return error
		return
	}

	switch result[0].(type) {
	case string:
		success.DataIface = []byte(result[0].(string))
		data, err := json.Marshal(&success)
		if err != nil {
			panic(err)
		}
		c.provider.Reply(c.ReplyStr, data) // FIXME: Log or return error
		return
	case []byte:
		success.DataIface = result[0].([]byte)
		data, err := json.Marshal(&success)
		if err != nil {
			panic(err)
		}
		c.provider.Reply(c.ReplyStr, data) // FIXME: Log or return error
		return
	}

	// Make sure we always marshal pointer structures
	result[0] = spikeutils.PointerFromInterface(result[0])
	encoded, err := json.Marshal(result[0])
	if err != nil {
		panic(err)
	}
	success.DataIface = encoded
	data, err := json.Marshal(&success)
	if err != nil {
		panic(err)
	}
	c.provider.Reply(c.ReplyStr, data) // FIXME: Log or return error
}

func (c *callBase) GetError() Error {
	return c.err
}

func (c *callBase) InternalError(err error) {
	c.Error(InternalError(err))
}

func (c *callBase) Error(err error, msg ...string) {
	if err == nil {
		panic("error request cant be nil")
	}
	err = Trace(err, 1)
	if brokerErr, ok := err.(Error); ok {
		c.error(brokerErr)
	} else {
		c.InternalError(err)
	}
}

// Timeout informs the max timeout for this request
func (c *callBase) Timeout(timeout time.Duration) {
	if c.ReplyStr == "" {
		return
	}

	if timeout <= 100*time.Millisecond {
		timeout = 100 * time.Millisecond
	}

	c.provider.Reply(c.ReplyStr, []byte(fmt.Sprintf("timeout:%d", int(timeout)))) // FIXME: Log or return error
}

// Error result
func (c *callBase) error(err Error) {
	if c.ReplyStr == "" {
		return
	}

	c.provider.Reply(c.ReplyStr, err.ToJSON()) // FIXME: Log or return error
}

func (c *callBase) ParseQuery(q interface{}) error {
	query := c.EndpointPattern.QueryParams()
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
