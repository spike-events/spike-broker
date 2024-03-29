package broker

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Error error request
type Error interface {
	Error() string
	Code() int
	Data() RawData
	ToJSON() []byte
}

type errorMessage struct {
	Message
	MessageStr string `json:"message,omitempty"`
}

func (e *errorMessage) Data() RawData {
	return e.DataIface
}

func (e *errorMessage) Error() string {
	return e.MessageStr
}

func (e *errorMessage) Code() int {
	return e.CodeInt
}

type RedirectRequest struct {
	URL string
}

// ToJSON error request
func (e *errorMessage) ToJSON() []byte {
	data, err := json.Marshal(e)
	if err != nil {
		panic(err)
	}
	return data
}

// information already exists
var (
	ErrorInformationAlreadyExists Error = &errorMessage{Message{http.StatusAlreadyReported, nil}, "information already exists"}
	ErrorNotFound                 Error = &errorMessage{Message{http.StatusNotFound, nil}, "not found"}
	ErrorStatusUnauthorized       Error = &errorMessage{Message{http.StatusUnauthorized, nil}, "not authorized"}
	ErrorStatusForbidden          Error = &errorMessage{Message{http.StatusForbidden, nil}, "forbidden"}
	ErrorInvalidParams            Error = &errorMessage{Message{http.StatusBadRequest, nil}, "invalid params"}
	ErrorInternalServerError      Error = &errorMessage{Message{http.StatusInternalServerError, nil}, "internal error"}
	ErrorAccessDenied             Error = &errorMessage{Message{http.StatusForbidden, nil}, "access denied"}
	ErrorTimeout                  Error = &errorMessage{Message{http.StatusRequestTimeout, nil}, "timeout"}
	ErrorServiceUnavailable       Error = &errorMessage{Message{http.StatusServiceUnavailable, nil}, "service unavailable"}
)

func InternalError(err error) Error {
	if err == nil {
		return ErrorInternalServerError
	}

	return &errorMessage{
		MessageStr: err.Error(),
		Message: Message{
			CodeInt: http.StatusInternalServerError,
		},
	}
}

func NewInvalidParamsError(msg string) Error {
	return &errorMessage{
		MessageStr: msg,
		Message:    Message{CodeInt: http.StatusBadRequest},
	}
}

func NewServiceUnavailableError(endpoint string) Error {
	return &errorMessage{
		MessageStr: fmt.Sprintf("service %s unavailable", endpoint),
		Message:    Message{CodeInt: http.StatusServiceUnavailable},
	}
}

func NewError(msg string, code int, data []byte) Error {
	return &errorMessage{
		MessageStr: msg,
		Message: Message{
			CodeInt:   code,
			DataIface: data,
		},
	}
}

func NewMessageFromJSON(j json.RawMessage) Error {
	var parsed errorMessage
	err := json.Unmarshal(j, &parsed)
	if err != nil {
		return nil
	}
	return &parsed
}
