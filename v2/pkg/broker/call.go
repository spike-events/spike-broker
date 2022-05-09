package broker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/hetiansu5/urlquery"
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
	InternalError(err error)
	NotFound()

	SetReply(reply string)
	SetToken(token string)
	SetProvider(provider Provider)
	SetEndpoint(p rids.Pattern)
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
		Data:            payload,
		EndpointPattern: p,
	}
}

func NewCallFromJSON(callJSON json.RawMessage) (Call, error) {
	// TODO: Support v1 JSON version
	var c call
	err := json.Unmarshal(callJSON, &c)
	if err != nil {
		return nil, err
	}
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
	Data            interface{}  `json:"data"`
	ReplyStr        string       `json:"reply"`
	EndpointPattern rids.Pattern `json:"endpointPattern"`
	Token           string       `json:"token"`

	provider Provider
	err      Error
}

func (c *call) UnmarshalJSON(data []byte) error {
	type callInnerType struct {
		Data            interface{}     `json:"data"`
		ReplyStr        string          `json:"reply"`
		EndpointPattern json.RawMessage `json:"endpointPattern"`
		Token           string          `json:"token"`
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

func (c *call) Endpoint() rids.Pattern {
	return c.EndpointPattern
}

func (c *call) SetEndpoint(p rids.Pattern) {
	c.EndpointPattern = p
}

func (c *call) Provider() Provider {
	return c.provider
}

func (c *call) Reply() string {
	return c.ReplyStr
}

func (c *call) RawData() interface{} {
	return c.Data
}

func (c *call) SetReply(reply string) {
	c.ReplyStr = reply
}

func (c *call) SetProvider(provider Provider) {
	c.provider = provider
}

func (c *call) SetToken(token string) {
	c.Token = token
}

func (c *call) RawToken() string {
	return c.Token
}

// ParseToken logado
func (c *call) ParseToken(t interface{}) {
	token := c.Token
	if len(token) > 0 {
		err := json.Unmarshal([]byte(token), &t)
		if err != nil {
			panic("invalid token on unmarshal")
		}
	}
}

// PathParam retorna parametro map string
func (c *call) PathParam(key string) string {
	params := c.Endpoint().Params()
	if params == nil {
		return ""
	}
	if _, ok := params[key]; !ok {
		return ""
	}
	return params[key].String()
}

// ParseData data
func (c *call) ParseData(v interface{}) error {
	switch c.Data.(type) {
	case []byte:
		return json.Unmarshal(c.Data.(json.RawMessage), v)
	}
	v = c.Data
	return nil
}

// ToJSON CallRequest
func (c *call) ToJSON() json.RawMessage {
	//if c.EndpointPattern.Version() == 1 {
	//
	//	cV1 := &callV1{
	//		Params:   nil,
	//		Data:     nil,
	//		Token:    "",
	//		Query:    "",
	//		Endpoint: "",
	//	}
	//}
	data, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	return data
}

// FromJSON builds the Call struct from raw data
func (c *call) FromJSON(data json.RawMessage, provider Provider, reply string) error {
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

func (c *call) File(f *dataurl.DataURL) {
	if c.ReplyStr == "" {
		return
	}

	var success Success
	success.Code = http.StatusOK
	payload, err := json.Marshal(f)
	if err != nil {
		panic(err)
	}
	success.Data = payload
	data, err := json.Marshal(&success)
	if err != nil {
		panic(err)
	}
	err = c.provider.PublishRaw(c.ReplyStr, data)
}

// OK result
func (c *call) OK(result ...interface{}) {
	if c.ReplyStr == "" {
		return
	}

	var success Success
	success.Code = http.StatusOK

	if len(result) == 0 {
		data, err := json.Marshal(&success)
		if err != nil {
			panic(err)
		}
		err = c.provider.PublishRaw(c.ReplyStr, data)
		if err != nil {
			log.Printf("request: failed to publish OK without result on %s inbox %s: %v",
				c.EndpointPattern, c.ReplyStr, err)
		}
		return
	}

	switch result[0].(type) {
	case string:
		success.Data = []byte(result[0].(string))
		data, err := json.Marshal(&success)
		if err != nil {
			panic(err)
		}
		err = c.provider.PublishRaw(c.ReplyStr, data)
		if err != nil {
			log.Printf("request: failed to publish OK string result on %s inbox %s: %v",
				c.EndpointPattern, c.ReplyStr, err)
		}
		return
	case []byte:
		success.Data = result[0].([]byte)
		data, err := json.Marshal(&success)
		if err != nil {
			panic(err)
		}
		err = c.provider.PublishRaw(c.ReplyStr, data)
		if err != nil {
			log.Printf("request: failed to publish OK []byte result on %s inbox %s: %v",
				c.EndpointPattern, c.ReplyStr, err)
		}
		return
	}

	// Make sure we always marshal pointer structures
	result[0] = spike_utils.PointerFromInterface(result[0])
	payload, err := json.Marshal(result[0])
	if err != nil {
		panic(err)
	}
	success.Data = payload
	data, err := json.Marshal(&success)
	if err != nil {
		panic(err)
	}
	err = c.provider.PublishRaw(c.ReplyStr, data)
	if err != nil {
		log.Printf("request: failed to publish OK JSON on %s inbox %s: %v", c.EndpointPattern, c.ReplyStr, err)
	}
}

func (c *call) GetError() Error {
	return c.err
}

// InternalError result
func (c *call) InternalError(err error) {
	c.Error(InternalError(err))
}

// Error result
func (c *call) Error(err error, msg ...string) {
	if err == nil {
		panic("error request cant be nil")
	}
	if brokerErr, ok := err.(Error); ok {
		c.error(brokerErr)
	} else {
		c.InternalError(err)
	}
}

// Timeout informs the max timeout for this request
func (c *call) Timeout(timeout time.Duration) {
	if c.ReplyStr == "" {
		return
	}

	if timeout <= 100*time.Millisecond {
		timeout = 100 * time.Millisecond
	}

	err := c.provider.PublishRaw(c.ReplyStr, []byte(fmt.Sprintf("timeout:%d", int(timeout))))
	if err != nil {
		log.Printf("request: failed to publish Timeout on %s inbox %s: %v", c.EndpointPattern, c.ReplyStr, err)
	}
}

// Error result
func (c *call) error(err Error) {
	if c.ReplyStr == "" {
		return
	}

	logLevel := os.Getenv("API_LOG_LEVEL")
	if logLevel == "DEBUG" || logLevel == "ERROR" {
		log.Printf("api error: endpoint %s inbox %s: %v", c.EndpointPattern, c.ReplyStr, err)
		if logLevel == "DEBUG" {
			log.Printf("api error debug: %s", debug.Stack())
		}
	}

	err2 := c.provider.PublishRaw(c.ReplyStr, err.ToJSON())
	if err2 != nil {
		log.Printf("request: failed to publish error response on %s inbox %s: %v", c.EndpointPattern,
			c.ReplyStr, err2)
	}
}

func (c *call) ParseQuery(q interface{}) error {
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

func (c *call) NotFound() {
	c.error(ErrorNotFound)
}
