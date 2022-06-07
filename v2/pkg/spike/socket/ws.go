package socket

import (
	"github.com/gorilla/websocket"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

const (
	WebSocketMsgBufferSize = 30
)

// NewConnectionWS socket
func NewConnectionWS(options Options) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var upgrader = websocket.Upgrader{}
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
		upgrader.HandshakeTimeout = time.Minute * 5
		upgrader.ReadBufferSize = 0
		upgrader.WriteBufferSize = 0
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade: %v", err)
			return
		}
		conn := newConnection(c, options)
		go wsHandler(conn)
	}
}

func wsHandler(c WSConnection) {
	var errorMsg *WSMessage
	defer func() {
		log.Printf("ws: defer wsHandler")
		if r := recover(); r != nil {
			log.Printf("ws: stack error, %v", r)
			log.Printf(string(debug.Stack()))
			log.Printf("ws: context done, disconnecting %s", c.GetID())
			err := c.WSConnection().Close()
			if err != nil {
				log.Printf("ws: failed to close connection %s: %v", c.GetID(), err)
			}
		}
	}()
	for {
		if errorMsg != nil {
			log.Printf("ws: error message %s %v", c.GetID(), errorMsg)
			err := c.WSConnection().WriteJSON(errorMsg)
			if err != nil {
				log.Printf("ws: failed to send error message on connection %s with data %v: %v", c.GetID(), errorMsg, err)
				break
			}
			errorMsg = nil
		}

		var wsMsg WSMessage
		err := c.WSConnection().ReadJSON(&wsMsg)
		if err != nil {
			if _, ok := err.(*websocket.CloseError); ok {
				log.Printf("ws: closed connection %s", c.GetID())
				break
			}
			wsMsg.Type = WSMessageTypeError
			wsMsg.Data = broker.InternalError(err)
			errorMsg = &wsMsg
			continue
		}

		wsMsgHandler := NewMessageHandler(&wsMsg)
		if wsMsgHandler == nil {
			wsMsg.Type = WSMessageTypeError
			errorMsg = &wsMsg
			// FIXME: Send error data to client
			continue
		}

		rErr := wsMsgHandler.Handle(c)
		if rErr != nil {
			wsMsg.Type = WSMessageTypeError
			wsMsg.Data = rErr
			errorMsg = &wsMsg
			continue
		}
	}

	log.Printf("ws: context done, disconnecting %s", c.GetID())
	err := c.WSConnection().Close()
	if err != nil {
		log.Printf("ws: failed to close connection %s: %v", c.GetID(), err)
	}

	log.Printf("ws: -> publish EventSocketDisconnected %s", c.GetID())
	c.Broker().Publish(rids.Spike().EventSocketDisconnected(c.GetID()), nil, c.GetSessionToken())
	log.Printf("ws: <- publish EventSocketDisconnected %s", c.GetID())
}
