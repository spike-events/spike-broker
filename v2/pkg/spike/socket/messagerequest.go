package socket

import (
	"encoding/json"

	"github.com/spike-events/spike-broker/v2/pkg/broker"
)

type WSMessageRequest struct {
	WSMessage
}

func (m *WSMessageRequest) Handle(ws WSConnection) broker.Error {
	if len(ws.GetToken()) != 0 {
		token, valid := ws.Authenticator().ValidateToken(ws.GetToken())
		if !valid {
			return broker.ErrorStatusUnauthorized
		}
		ws.SetSessionToken(token)
	}

	p, brokerErr := PatternFromEndpoint(ws.GetHandlers(), m.SpecificEndpoint())
	if brokerErr != nil {
		return brokerErr
	}
	call := broker.NewCall(p, m.Data)
	call.SetToken(ws.GetToken())
	call.SetProvider(ws.Broker())

	if !ws.Authorizer().HasPermission(call) {
		return broker.ErrorStatusForbidden
	}

	var response broker.RawData
	rErr := ws.Broker().Request(p, m.Data, &response, ws.GetToken())
	if rErr != nil {
		return rErr
	}

	m.Type = WSMessageTypeResponse
	m.Data = json.RawMessage(response)
	err := ws.WriteJSON(m)
	if err != nil {
		return broker.InternalError(err)
	}
	return nil
}
