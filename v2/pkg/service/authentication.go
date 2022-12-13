package service

import (
	"encoding/json"

	"github.com/spike-events/spike-broker/v2/pkg/broker"
)

type Authenticator interface {
	ValidateToken(token json.RawMessage) (processedToken json.RawMessage, valid bool)
}

type Authorizer interface {
	HasPermission(c broker.Call) bool
}
