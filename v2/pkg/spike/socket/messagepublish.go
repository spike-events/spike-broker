package socket

import (
	"github.com/spike-events/spike-broker/v2/pkg/broker"
)

type WSMessagePublish struct {
	WSMessage
}

func (m *WSMessagePublish) Handle(ws WSConnection) broker.Error {
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

	err := ws.Broker().Publish(p, m.Data, ws.GetToken())
	if err != nil {
		return broker.InternalError(err)
	}
	return nil
}
