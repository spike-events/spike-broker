package spike

import (
	"encoding/json"

	"github.com/spike-events/spike-broker/v2/pkg/service"
)

type testAuthenticator struct {
	validator func(message json.RawMessage) (json.RawMessage, bool)
}

func (t testAuthenticator) ValidateToken(token json.RawMessage) (processedToken json.RawMessage, valid bool) {
	return t.validator(token)
}

func NewTestAuthenticator(validator func(message json.RawMessage) (json.RawMessage, bool)) service.Authenticator {
	return &testAuthenticator{
		validator: validator,
	}
}
