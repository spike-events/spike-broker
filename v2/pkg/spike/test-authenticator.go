package spike

import "github.com/spike-events/spike-broker/v2/pkg/service"

type testAuthenticator struct {
	validator func(string) (string, bool)
}

func (t testAuthenticator) ValidateToken(token string) (processedToken string, valid bool) {
	return t.validator(token)
}

func NewTestAuthenticator(validator func(string) (string, bool)) service.Authenticator {
	return &testAuthenticator{
		validator: validator,
	}
}
