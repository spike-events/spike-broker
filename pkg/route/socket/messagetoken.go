package socket

import (
	"encoding/json"
	"github.com/spike-events/spike-broker/pkg/rids"
	"github.com/spike-events/spike-broker/pkg/service/request"
)

type WSMessageToken struct {
	WSMessage
}

func (t *WSMessageToken) Handle(ws *WSConnection) *request.ErrorRequest {
	var result json.RawMessage
	err := ws.Broker().Request(rids.Route().ValidateToken(), request.NewRequest(t.Token), &result)
	if err != nil {
		return &request.ErrorStatusUnauthorized
	}

	publish := ws.GetSessionToken() == nil
	ws.SetSessionToken(result)
	ws.SetSessionID(t.ID)
	t.Token = string(result)
	ws.WSConnection().WriteJSON(t)
	if publish {
		go ws.Broker().Publish(rids.Route().EventSocketConnected(), request.NewRequest(ws), ws.GetSessionToken())
	}
	return nil
}
