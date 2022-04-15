package service

import (
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/service/request"
)

// Auth interface
type Auth interface {
	ValidateToken(token string) (processedToken string, valid bool)
	UserHavePermission(r *request.CallRequest) bool
}

// AuthRid auth service
type AuthRid struct {
	Service  string
	Auth     Auth
	Patterns []*rids.Pattern
}
