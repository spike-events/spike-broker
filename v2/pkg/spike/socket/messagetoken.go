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

	token, valid := ws.Authenticator().ValidateToken([]byte(t.Token))
	if !valid {
		return broker.ErrorStatusUnauthorized
	}

	publish := string(ws.GetSessionToken()) == ""
	ws.SetSessionToken(token)
	ws.SetSessionID(t.ID)
	t.Token = string(token)
	err := ws.WriteJSON(t)
	if err != nil {
		return broker.InternalError(err)
	}
	if publish {
		go ws.Broker().Publish(rids.Spike().EventSocketConnected(), ws, ws.GetSessionToken())
	}
	return nil
}
