package service

import (
	"encoding/json"
	"github.com/spike-events/spike-broker/pkg/rids"
	"github.com/spike-events/spike-broker/pkg/service/request"
)

// Auth interface
type Auth interface {
	ValidateToken(token json.RawMessage) (processedToken json.RawMessage, valid bool)
	UserHavePermission(r *request.CallRequest) bool
}

// AuthRid auth service
type AuthRid struct {
	Service  string
	Auth     Auth
	Patterns []*rids.Pattern
}
