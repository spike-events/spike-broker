package v2

import (
	"encoding/json"

	"github.com/spike-events/spike-broker/v2/pkg/service"
)

type Authenticator struct {
}

func (a Authenticator) ValidateToken(token json.RawMessage) (processedToken json.RawMessage, valid bool) {
	return token, string(token) == "\"token-string\""
}

func NewAuthenticator() service.Authenticator {
	return &Authenticator{}
}
