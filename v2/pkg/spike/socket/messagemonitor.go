package socket

import (
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	spikeutils "github.com/spike-events/spike-broker/v2/pkg/spike-utils"
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

	p := spikeutils.PatternFromEndpoint(ws.GetHandlers(), m.SpecificEndpoint())
	call := broker.NewCall(p, m.Data)
	call.SetToken(ws.GetToken())
	call.SetProvider(ws.Broker())

	if !ws.Authorizer().HasPermission(call) {
		return broker.ErrorStatusForbidden
	}

	localHandler := func(r broker.Call) {
		wsMsg := &WSMessage{
			ID:       m.ID,
			Type:     WSMessageTypePublish,
			Endpoint: m.Endpoint,
			Data:     string(r.RawData()),
		}
		err := ws.WSConnection().WriteJSON(wsMsg)
		if err != nil {
			r.InternalError(err)
			return
		}
		r.OK()
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

	return nil
}
