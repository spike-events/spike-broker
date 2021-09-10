package socket

import "github.com/spike-events/spike-broker/pkg/service/request"

type WSMessageKeepAlive struct {
	WSMessage
}

func (m *WSMessageKeepAlive) Handle(ws *WSConnection) *request.ErrorRequest {
	err := ws.WSConnection().WriteJSON(m)
	if err != nil {
		return &request.ErrorInternalServerError
	}
	return nil
}
