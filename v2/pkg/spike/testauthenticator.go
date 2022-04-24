package spike

import "github.com/spike-events/spike-broker/v2/pkg/service"

type testAuthenticator struct {
}

func (t testAuthenticator) ValidateToken(token string) (processedToken string, valid bool) {
	//TODO implement me
	panic("implement me")
}

func NewTestAuthenticator(validator func(string) (string, bool)) service.Authenticator {
	return &testAuthenticator{}
}
