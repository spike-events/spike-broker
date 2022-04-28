package broker

import (
	"encoding/json"
	"net/http"
)

// Error error request
type Error interface {
	Error() string
	Code() int
	Message() string
	Data() interface{}
	ToJSON() []byte
}

type errorRequest struct {
	MessageStr string      `json:"message,omitempty"`
	CodeInt    int         `json:"code,omitempty"`
	DataIface  interface{} `json:"data,omitempty"`
}

func (e *errorRequest) Data() interface{} {
	return e.DataIface
}

func (e *errorRequest) Error() string {
	return e.MessageStr
}

func (e *errorRequest) Code() int {
	return e.CodeInt
}

func (e *errorRequest) Message() string {
	return e.MessageStr
}

type RedirectRequest struct {
	URL string
}

// ToJSON error request
func (e errorRequest) ToJSON() []byte {
	data, err := json.Marshal(e)
	if err != nil {
		panic(err)
	}
	return data
}

// information already exists
var (
	ErrorInformationAlreadyExists Error = &errorRequest{"information already exists", http.StatusAlreadyReported, nil}
	ErrorNotFound                 Error = &errorRequest{"not found", http.StatusNotFound, nil}
	ErrorStatusUnauthorized       Error = &errorRequest{"not authorized", http.StatusUnauthorized, nil}
	ErrorStatusForbidden          Error = &errorRequest{"forbidden", http.StatusForbidden, nil}
	ErrorInvalidParams            Error = &errorRequest{"invalid params", http.StatusBadRequest, nil}
	ErrorInternalServerError      Error = &errorRequest{"internal error", http.StatusInternalServerError, nil}
	ErrorAccessDenied             Error = &errorRequest{"access denied", http.StatusUnauthorized, nil}
	ErrorTimeout                  Error = &errorRequest{"timeout", http.StatusRequestTimeout, nil}
)

func InternalError(err error) Error {
	return &errorRequest{
		MessageStr: err.Error(),
		CodeInt:    http.StatusInternalServerError,
	}
}

func NewInvalidParamsError(msg string) Error {
	return &errorRequest{
		MessageStr: msg,
		CodeInt:    http.StatusBadRequest,
	}
}

func NewError(msg string, code int, data interface{}) Error {
	return &errorRequest{
		MessageStr: msg,
		CodeInt:    code,
		DataIface:  data,
	}
}

func NewErrorFromJSON(j json.RawMessage) Error {
	var parsed errorRequest
	err := json.Unmarshal(j, &parsed)
	if err != nil {
		return nil
	}
	return &parsed
}
