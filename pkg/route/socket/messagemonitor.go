package socket

import (
	"fmt"
	"github.com/spike-events/spike-broker/pkg/models"
	"github.com/spike-events/spike-broker/pkg/rids"
	"github.com/spike-events/spike-broker/pkg/service/request"
	"github.com/spike-events/spike-broker/pkg/utils"
	"strings"
)

type WSMessageMonitor struct {
	WSMessage
}

func (m *WSMessageMonitor) Handle(ws *WSConnection) *request.ErrorRequest {

	if len(ws.token) != 0 {
		type rs struct {
			Token string
		}
		var result rs
		rErr := ws.Broker().Request(rids.Route().ValidateToken(), nil, &result, ws.token)
		if rErr != nil {
			return rErr
		}
		ws.token = result.Token
	}

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
		callAuth.Token = string(ws.token)

		if !ws.auth[0].Auth.UserHavePermission(callAuth) {
			return &request.ErrorStatusForbidden
		}
	}

	localHandler := func(r *request.CallRequest) {
		wsMsg := &WSMessage{
			ID:       m.ID,
			Type:     WSMessageTypePublish,
			Endpoint: m.Endpoint,
			Data:     string(r.Data),
		}
		err := ws.WSConnection().WriteJSON(wsMsg)
		if err != nil {
			r.Error(err)
			return
		}
		r.OK()
	}

	unsubscribe, err := ws.Broker().Monitor(ws.ID, endpoint, localHandler)
	if err != nil {
		return request.InternalError(err)
	}

	go func() {
		<-ws.Context().Done()
		unsubscribe()
	}()

	return nil
}
