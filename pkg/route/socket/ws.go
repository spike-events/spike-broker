package socket

import (
	"context"
	"github.com/gorilla/websocket"
	"github.com/spike-events/spike-broker/pkg/rids"
	"github.com/spike-events/spike-broker/pkg/service"
	"github.com/spike-events/spike-broker/pkg/service/request"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	WebSocketMsgBufferSize = 30
)

// NewConnectionWS socket
func NewConnectionWS(srvBase *service.Base, oauth ...*service.AuthRid) func(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		cancel()
	}()
	return func(w http.ResponseWriter, r *http.Request) {
		var upgrader = websocket.Upgrader{}
		upgrader.ReadBufferSize = 0
		upgrader.WriteBufferSize = 0
		upgrader.HandshakeTimeout = time.Minute * 5
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade: %v", err)
			return
		}
		conn := newConnection(c, srvBase.Broker(), oauth...)
		go wsHandler(ctx, conn)
	}
}

func wsHandler(ctx context.Context, c *WSConnection) {
	var errorMsg *WSMessage
	defer func() {
		if r := recover(); r != nil {
			log.Printf("ws: stack error, %v", r)
			log.Printf("ws: context done, disconnecting %s", c.ID)
			err := c.WSConnection().Close()
			if err != nil {
				log.Printf("ws: failed to close connection %s: %v", c.ID, err)
			}
		}
	}()
	go func() {
		select {
		case <-ctx.Done():
		case <-c.Context().Done():
		}
		log.Printf("ws: context done, disconnecting %s", c.ID)
		err := c.WSConnection().Close()
		if err != nil {
			log.Printf("ws: failed to close connection %s: %v", c.ID, err)
		}
		log.Printf("ws: event socket, disconnecting %s", c.ID)
		c.Broker().Publish(rids.Route().EventSocketDisconnected(c.ID), nil, c.GetSessionToken())
	}()
	for {
		if errorMsg != nil {
			err := c.WSConnection().WriteJSON(errorMsg)
			if err != nil {
				log.Printf("ws: failed to send error message on connection %s with data %v: %v", c.ID, errorMsg, err)
				c.CancelContext()
				return
			}
			errorMsg = nil
		}

		var wsMsg WSMessage
		err := c.WSConnection().ReadJSON(&wsMsg)
		if err != nil {
			if _, ok := err.(*websocket.CloseError); ok {
				log.Printf("ws: closed connection %s", c.ID)
				c.CancelContext()
				return
			}
			wsMsg.Type = WSMessageTypeError
			wsMsg.Data = request.InternalError(err)
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
}
