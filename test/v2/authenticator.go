package v2

import (
	"github.com/spike-events/spike-broker/v2/pkg/service"
)

type Authenticator struct {
}

func (a Authenticator) ValidateToken(token []byte) (processedToken []byte, valid bool) {
	return token, string(token) == "token-string"
}

func NewAuthenticator() service.Authenticator {
	return &Authenticator{}
}
