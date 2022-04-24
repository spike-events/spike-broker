package socket

import "github.com/spike-events/spike-broker/v2/pkg/service/request"

type WSMessageKeepAlive struct {
	WSMessage
}

func (m *WSMessageKeepAlive) Handle(ws *WSConnection) *request.Error {
	err := ws.WSConnection().WriteJSON(m)
	if err != nil {
		return &request.ErrorInternalServerError
	}
	return nil
}
