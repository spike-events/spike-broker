package socket

import (
	"github.com/spike-events/spike-broker/pkg/service/request"
)

type WSMessageType string

const (
	WSMessageTypeRequest     WSMessageType = "request"
	WSMessageTypeResponse    WSMessageType = "response"
	WSMessageTypePublish     WSMessageType = "publish"
	WSMessageTypeSubscribe   WSMessageType = "subscribe"
	WSMessageTypeMonitor     WSMessageType = "monitor"
	WSMessageTypeUnsubscribe WSMessageType = "unsubscribe"
	WSMessageTypeToken       WSMessageType = "token"
	WSMessageTypeError       WSMessageType = "error"
	WSMessageTypeKeepAlive   WSMessageType = "keepalive"
)

type WSMessageHandler interface {
	Handle(c *WSConnection) *request.ErrorRequest
}

type WSMessage struct {
	ID       string        `json:"id"`
	Type     WSMessageType `json:"type"`
	Endpoint string        `json:"endpoint,omitempty"`
	Method   string        `json:"method,omitempty"`
	Token    string        `json:"token,omitempty"`
	Query    string        `json:"query,omitempty"`
	Data     interface{}   `json:"data,omitempty"`
}

type wsMessageNilHandler struct {
	WSMessage
}

func (m *wsMessageNilHandler) Handle(c *WSConnection) *request.ErrorRequest {
	return nil
}

func NewMessageHandler(wsMsg *WSMessage) WSMessageHandler {
	switch wsMsg.Type {
	case WSMessageTypeSubscribe:
	case WSMessageTypeMonitor:
		return &WSMessageMonitor{WSMessage: *wsMsg}
	case WSMessageTypeRequest:
		return &WSMessageRequest{WSMessage: *wsMsg}
	case WSMessageTypeError:
	case WSMessageTypeResponse:
	case WSMessageTypePublish:
		return &WSMessagePublish{WSMessage: *wsMsg}
	case WSMessageTypeUnsubscribe:
	case WSMessageTypeToken:
		return &WSMessageToken{WSMessage: *wsMsg}
	case WSMessageTypeKeepAlive:
		return &WSMessageKeepAlive{WSMessage: *wsMsg}
	}
	return &wsMessageNilHandler{WSMessage: *wsMsg}
}
