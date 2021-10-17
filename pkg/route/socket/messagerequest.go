package socket

import (
	"encoding/json"
	"fmt"
	"github.com/spike-events/spike-broker/pkg/models"
	"github.com/spike-events/spike-broker/pkg/rids"
	"github.com/spike-events/spike-broker/pkg/service/request"
	"github.com/spike-events/spike-broker/pkg/utils"
	"strings"
)

type WSMessageRequest struct {
	WSMessage
}

func (m *WSMessageRequest) Handle(ws *WSConnection) *request.ErrorRequest {
	values := strings.Split(m.Endpoint, ".")
	permission := models.HavePermissionRequest{
		Service:  values[0],
		Endpoint: strings.Join(values[1:], "."),
		Method:   m.Method,
	}

	var endpoint *rids.Pattern
	for _, auth := range ws.auth {
		if auth.Service != permission.Service {
			continue
		}

		endpoint = utils.PatternFromEndpoint(auth.Patterns, &permission)
		break
	}

	if endpoint == nil || m.Method == "INTERNAL" {
		return &request.ErrorStatusForbidden
	}

	values = strings.Split(endpoint.Endpoint, ".")
	permission = models.HavePermissionRequest{
		Service:  permission.Service,
		Endpoint: fmt.Sprintf("%s.%s", permission.Service, endpoint.Endpoint),
		Method:   m.Method,
	}
	if endpoint.Authenticated && len(ws.auth) > 0 {

		callAuth := request.NewRequest(permission)
		callAuth.Token = ws.token

		if !ws.auth[0].Auth.UserHavePermission(callAuth) {
			return &request.ErrorStatusForbidden
		}
	}

	var response interface{}
	rErr := ws.Broker().Request(endpoint, request.NewRequest(m.Data), &response, ws.token)
	if rErr != nil {
		return rErr
	}

	data, err := json.Marshal(response)
	if err != nil {
		return request.InternalError(err)
	}
	m.Type = WSMessageTypeResponse
	m.Data = string(data)
	err = ws.WSConnection().WriteJSON(m)
	if err != nil {
		return request.InternalError(err)
	}
	return nil
}
