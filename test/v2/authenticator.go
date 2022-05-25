package v2

import "github.com/spike-events/spike-broker/v2/pkg/service"

type Authenticator struct {
}

func (a Authenticator) ValidateToken(token string) (processedToken string, valid bool) {
	return token, token == "token-string"
}

func NewAuthenticator() service.Authenticator {
	return &Authenticator{}
}
