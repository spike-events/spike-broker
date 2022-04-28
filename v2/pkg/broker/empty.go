package broker

import (
	"encoding/json"
	"time"

	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/vincent-petithory/dataurl"
)

// EmptyRequest empty
func EmptyRequest() Call {
	return &empty{}
}

type empty struct {
	provider Provider
	endpoint rids.Pattern
}

func (e *empty) GetError() Error {
	return nil
}

func (e *empty) RawData() interface{} {
	return ""
}

func (e *empty) Reply() string {
	return ""
}

func (e *empty) Provider() Provider {
	return e.provider
}

func (e *empty) Endpoint() rids.Pattern {
	return e.endpoint
}

func (e *empty) SetEndpoint(p rids.Pattern) {
	e.endpoint = p
}

func (e *empty) SetToken(token string) {}

func (e *empty) SetProvider(provider Provider) {
	e.provider = provider
}

func (e *empty) RawToken() string {
	return ""
}

func (e *empty) ParseQuery(q interface{}) error {
	return nil
}

func (e *empty) FromJSON(data json.RawMessage, provider Provider, reply string) error {
	return nil
}

func (e *empty) InternalError(err error) {}

func (e *empty) Error(err Error, msg ...string) {}

func (e *empty) ParseToken(t interface{}) {}

func (e *empty) SetReply(reply string) {}

func (e *empty) PathParam(key string) string {
	return ""
}

func (e *empty) ParseData(v interface{}) error {
	return nil
}

func (e *empty) ToJSON() json.RawMessage {
	return json.RawMessage{}
}

func (e *empty) File(f *dataurl.DataURL) {}

func (e *empty) OK(result ...interface{}) {}

func (e *empty) Timeout(timeout time.Duration) {}

func (e *empty) NotFound() {}
