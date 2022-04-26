package socket

import (
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

type WSMessageToken struct {
	WSMessage
}

func (t *WSMessageToken) Handle(ws WSConnection) broker.Error {
	if len(t.Token) == 0 {
		return broker.ErrorInvalidParams
	}

	token, valid := ws.Authenticator().ValidateToken(t.Token)
	if !valid {
		return broker.ErrorStatusUnauthorized
	}

	publish := ws.GetSessionToken() == ""
	ws.SetSessionToken(token)
	ws.SetSessionID(t.ID)
	t.Token = token
	ws.WSConnection().WriteJSON(t)
	if publish {
		go ws.Broker().Publish(rids.Spike().EventSocketConnected(), ws, ws.GetSessionToken())
	}
	return nil
}
