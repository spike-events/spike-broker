package request

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/hetiansu5/urlquery"
	"github.com/spike-events/spike-broker/pkg/utils"
	"github.com/vincent-petithory/dataurl"
)

type AccessRequest struct {
	*CallRequest
	err     *bool
	get     *bool
	methods []string
}

func (a *AccessRequest) Methods() []string {
	return a.methods
}

func (a *AccessRequest) IsError() bool {
	return a.err == nil || *a.err
}

func (a *AccessRequest) RequestIsGet() *bool {
	return a.get
}

func (a *AccessRequest) Access(get bool, methods ...string) {
	f := false
	a.err = &f
	a.get = &get
	a.methods = methods
}
func (a *AccessRequest) AccessDenied() {
	f := true
	a.err = &f
}

func (a *AccessRequest) AccessGranted() {
	f := false
	a.err = &f
}

// CallRequest handler
type CallRequest struct {
	Params   map[string]string
	Data     []byte
	reply    string
	provider Provider

	PostForm url.Values
	Form     url.Values
	Token    string
	Query    string
	Endpoint string
	Header   http.Header
}

// ErrorRequest error request
type ErrorRequest struct {
	Message      string          `json:"message,omitempty"`
	Code         int             `json:"code,omitempty"`
	Error        error           `json:"error,omitempty"`
	ErrorMessage string          `json:"errorMessage,omitempty"`
	Data         json.RawMessage `json:"data,omitempty"`
	Type         string          `json:"type,omitempty"`
}

type RedirectRequest struct {
	URL string
}

// ParseToken logado
func (c *CallRequest) ParseToken(t interface{}) {
	token := c.Token
	if len(token) > 0 {
		json.Unmarshal([]byte(token), &t)
	}
}

func (c *CallRequest) Redirect(url string) {
	payload, _ := json.Marshal(RedirectRequest{
		URL: url,
	})
	c.provider.PublishRaw(c.reply, payload)
}

func (c *CallRequest) SetProvider(provider Provider) {
	c.provider = provider
}

func (c *CallRequest) SetReply(reply string) {
	c.reply = reply
}

// Parse error request
func (e *ErrorRequest) Parse(payload []byte) error {
	err := json.Unmarshal(payload, e)
	if err != nil {
		return err
	}
	if len(e.ErrorMessage) > 0 {
		e.Error = fmt.Errorf(e.ErrorMessage)
	}
	return nil
}

// ToJSON error request
func (e ErrorRequest) ToJSON() []byte {
	data, err := json.Marshal(e)
	if err != nil {
		panic(err)
	}
	return data
}

// EmptyRequest empty
func EmptyRequest() *CallRequest {
	return &CallRequest{}
}

// NewRequest nova instancia da call request
func NewRequest(data interface{}) *CallRequest {
	var payload []byte
	var err error
	switch data.(type) {
	case []byte:
		panic("invalid data")
	case nil:
	default:
		data = utils.PointerFromInterface(data)
		payload, err = json.Marshal(data)
		if err != nil {
			panic(err)
		}
	}

	// Make sure we always handle pointers
	return &CallRequest{
		Params: nil,
		Data:   payload,
	}
}

// CloneRequest clone request
func (c CallRequest) CloneRequest(data interface{}) *CallRequest {
	switch data.(type) {
	case []byte:
		panic("invalid data")
	}
	var err error
	c.Data, err = json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return &c
}

// PathParam retorna parametro map string
func (c *CallRequest) PathParam(key string) string {
	return c.Params[key]
}

// ParseData data
func (c *CallRequest) ParseData(v interface{}) error {
	return json.Unmarshal(c.Data, v)
}

// ToJSON CallRequest
func (c *CallRequest) ToJSON() []byte {
	data, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	return data
}

// Unmarshal data
func (c *CallRequest) Unmarshal(data []byte, reply string, provider Provider) error {
	if len(data) > 0 {
		err := json.Unmarshal(data, c)
		if err != nil {
			return err
		}
	}
	c.provider = provider
	c.reply = reply
	return nil
}

func (c *CallRequest) File(f *dataurl.DataURL) {
	if c.reply == "" {
		return
	}

	var success ErrorRequest
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
	err = c.provider.PublishRaw(c.reply, data)
	if err != nil {
		log.Printf("request: failed to publish file on %s inbox %s: %v", c.Endpoint, c.reply, err)
	}
}

// OK result
func (c *CallRequest) OK(result ...interface{}) {
	if c.reply == "" {
		return
	}

	var success ErrorRequest
	success.Code = http.StatusOK

	if len(result) == 0 {
		data, err := json.Marshal(&success)
		if err != nil {
			panic(err)
		}
		err = c.provider.PublishRaw(c.reply, data)
		if err != nil {
			log.Printf("request: failed to publish OK without result on %s inbox %s: %v", c.Endpoint, c.reply, err)
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
		err = c.provider.PublishRaw(c.reply, data)
		if err != nil {
			log.Printf("request: failed to publish OK string result on %s inbox %s: %v", c.Endpoint, c.reply, err)
		}
		return
	case []byte:
		success.Data = result[0].([]byte)
		data, err := json.Marshal(&success)
		if err != nil {
			panic(err)
		}
		err = c.provider.PublishRaw(c.reply, data)
		if err != nil {
			log.Printf("request: failed to publish OK []byte result on %s inbox %s: %v", c.Endpoint, c.reply, err)
		}
		return
	}

	// Make sure we always marshal pointer structures
	result[0] = utils.PointerFromInterface(result[0])
	payload, err := json.Marshal(result[0])
	if err != nil {
		panic(err)
	}
	success.Data = payload
	data, err := json.Marshal(&success)
	if err != nil {
		panic(err)
	}
	err = c.provider.PublishRaw(c.reply, data)
	if err != nil {
		log.Printf("request: failed to publish OK JSON on %s inbox %s: %v", c.Endpoint, c.reply, err)
	}
}

// Error result
func (c *CallRequest) Error(err error, typeError ...string) {
	if len(typeError) == 0 {
		typeError = append(typeError, "")
	}
	c.error(ErrorRequest{err.Error(), http.StatusInternalServerError, err, err.Error(), nil, typeError[0]})
}

// ErrorRequest result
func (c *CallRequest) ErrorRequest(err *ErrorRequest, msg ...string) {
	if err == nil {
		panic("error request cant be nil")
	}
	errCopy := *err
	if len(msg) > 0 {
		errCopy.Message = msg[0]
	}
	c.error(errCopy)
}

// Timeout informs the max timeout for this request
func (c *CallRequest) Timeout(timeout time.Duration) {
	if c.reply == "" {
		return
	}

	if timeout <= 100*time.Millisecond {
		timeout = 100 * time.Millisecond
	}

	err := c.provider.PublishRaw(c.reply, []byte(fmt.Sprintf("timeout:%d", int(timeout))))
	if err != nil {
		log.Printf("request: failed to publish Timeout on %s inbox %s: %v", c.Endpoint, c.reply, err)
	}
}

// Error result
func (c *CallRequest) error(err ErrorRequest) {
	if c.reply == "" {
		return
	}

	if err.Error != nil {
		err.ErrorMessage = err.Error.Error()
	}

	logLevel := os.Getenv("API_LOG_LEVEL")
	if logLevel == "DEBUG" || logLevel == "ERROR" {
		log.Printf("api error: endpoint %s inbox %s: %v", c.Endpoint, c.reply, err)
		if logLevel == "DEBUG" {
			log.Printf("api error debug: %s", debug.Stack())
		}
	}

	err2 := c.provider.PublishRaw(c.reply, err.ToJSON())
	if err2 != nil {
		log.Printf("request: failed to publish error response on %s inbox %s: %v", c.Endpoint, c.reply, err2)
	}
}

func (c *CallRequest) ParseQuery(q interface{}) error {
	if c.Query != "" {
		return urlquery.Unmarshal([]byte(c.Query), q)
	}
	return nil
}

// information already exists
var (
	ErrorInformationAlreadyExists = ErrorRequest{"information already exists", http.StatusAlreadyReported, fmt.Errorf("information already exists"), "", nil, ""}
	ErrorNotFound                 = ErrorRequest{"not found", http.StatusNotFound, fmt.Errorf("not found"), "", nil, ""}
	ErrorStatusUnauthorized       = ErrorRequest{"not authorized", http.StatusUnauthorized, fmt.Errorf("not authorized"), "", nil, ""}
	ErrorStatusForbidden          = ErrorRequest{"forbidden", http.StatusForbidden, fmt.Errorf("forbidden"), "", nil, ""}
	ErrorInvalidParams            = ErrorRequest{"invalid params", http.StatusBadRequest, fmt.Errorf("invalid params"), "", nil, ""}
	ErrorInternalServerError      = ErrorRequest{"internal error", http.StatusInternalServerError, fmt.Errorf("internal error"), "", nil, ""}
	ErrorAccessDenied             = ErrorRequest{"access denied", http.StatusUnauthorized, fmt.Errorf("access denied"), "", nil, ""}
	ErrorTimeout                  = ErrorRequest{"timeout", http.StatusRequestTimeout, fmt.Errorf("timeout"), "", nil, ""}
)

// Error result
func (c *CallRequest) NotFound() {
	c.error(ErrorNotFound)
}

func InternalError(err error) *ErrorRequest {
	return &ErrorRequest{
		Message:      err.Error(),
		Code:         http.StatusInternalServerError,
		Error:        err,
		ErrorMessage: err.Error(),
	}
}

// Search query de search
func (c *CallRequest) Search(columns ...string) string {
	search := c.Form.Get("search")
	if search == "" {
		return "deleted_at IS NULL"
	}

	for i := range columns {
		columns[i] = columns[i] + " LIKE ?"
	}

	words := strings.Split(search, " ")

	var andConditions []string
	for i := 0; i < len(words); i++ {
		andConditions = append(andConditions, "("+strings.Join(columns, " OR ")+")")
	}

	var wordsParams []string
	for _, w := range words {
		for range columns {
			wordsParams = append(wordsParams, "'%"+w+"%'")
		}
	}

	where := "deleted_at IS NULL AND (" + strings.Join(andConditions, " AND ") + ")"
	for strings.Contains(where, "?") {
		where = strings.Replace(where, "?", wordsParams[0], 1)
		wordsParams = wordsParams[1:]
	}

	log.Println(where)

	return where
}
