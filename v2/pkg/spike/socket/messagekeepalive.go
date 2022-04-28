package socket

import (
	"github.com/spike-events/spike-broker/v2/pkg/broker"
)

type WSMessageKeepAlive struct {
	WSMessage
}

func (m *WSMessageKeepAlive) Handle(ws WSConnection) broker.Error {
	err := ws.WSConnection().WriteJSON(m)
	if err != nil {
		return broker.ErrorInternalServerError
	}
	return nil
}
