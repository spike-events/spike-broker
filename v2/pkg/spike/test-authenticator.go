package spike

import (
	"github.com/spike-events/spike-broker/v2/pkg/service"
)

type testAuthenticator struct {
	validator func(message []byte) ([]byte, bool)
}

func (t testAuthenticator) ValidateToken(token []byte) (processedToken []byte, valid bool) {
	return t.validator(token)
}

func NewTestAuthenticator(validator func(message []byte) ([]byte, bool)) service.Authenticator {
	return &testAuthenticator{
		validator: validator,
	}
}
