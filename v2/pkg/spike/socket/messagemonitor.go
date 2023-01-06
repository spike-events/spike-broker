package socket

import (
	"encoding/json"

	"github.com/spike-events/spike-broker/v2/pkg/broker"
)

type WSMessageMonitor struct {
	WSMessage
}

func (m *WSMessageMonitor) Handle(ws WSConnection) broker.Error {

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

	localHandler := func(c broker.Call) {
		wsMsg := &WSMessage{
			ID:       m.ID,
			Type:     WSMessageTypePublish,
			Endpoint: m.Endpoint,
			Data:     json.RawMessage(c.RawData()),
		}
		err := ws.WSConnection().WriteJSON(wsMsg)
		if err != nil {
			c.Error(err)
			return
		}
		c.OK()
	}

	sub := broker.Subscription{
		Resource: p,
		Handler:  localHandler,
	}
	unsubscribe, err := ws.Broker().Monitor(ws.GetID().String(), sub)
	if err != nil {
		return broker.InternalError(err)
	}

	go func() {
		<-ws.Context().Done()
		unsubscribe()
	}()

	m.Type = WSMessageTypeResponse
	m.Data = nil
	err = ws.WSConnection().WriteJSON(m)
	if err != nil {
		return broker.InternalError(err)
	}
	return nil
}
