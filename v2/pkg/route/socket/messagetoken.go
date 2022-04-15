package socket

import (
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/service/request"
)

type WSMessageToken struct {
	WSMessage
}

func (t *WSMessageToken) Handle(ws *WSConnection) *request.ErrorRequest {
	type rs struct {
		Token string
	}
	var result rs
	err := ws.Broker().Request(rids.Route().ValidateToken(), nil, &result, t.Token)
	if err != nil {
		return &request.ErrorStatusUnauthorized
	}

	publish := ws.GetSessionToken() == ""
	ws.SetSessionToken(result.Token)
	ws.SetSessionID(t.ID)
	t.Token = result.Token
	ws.WSConnection().WriteJSON(t)
	if publish {
		go ws.Broker().Publish(rids.Route().EventSocketConnected(), request.NewRequest(ws), ws.GetSessionToken())
	}
	return nil
}
