package service

import "github.com/spike-events/spike-broker/v2/pkg/broker"

type Authenticator interface {
	ValidateToken(token string) (processedToken string, valid bool)
}

type Authorizer interface {
	HasPermission(c broker.Call) bool
}
