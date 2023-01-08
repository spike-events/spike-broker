package socket

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/gorilla/websocket"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

const (
	WebSocketMsgBufferSize = 30
)

// NewConnectionWS socket
func NewConnectionWS(options Options) func(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		cancel()
	}()
	return func(w http.ResponseWriter, r *http.Request) {
		var upgrader = websocket.Upgrader{}
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade: %v", err)
			return
		}
		conn := newConnection(c, options)
		go wsHandler(ctx, conn)
	}
}

func wsHandler(ctx context.Context, c WSConnection) {
	var errorMsg *WSMessage
	defer func() {
		if r := recover(); r != nil {
			log.Printf("ws: stack error, %v", r)
			log.Printf(string(debug.Stack()))
			log.Printf("ws: context done, disconnecting %s", c.GetID())
			err := c.Close()
			if err != nil {
				log.Printf("ws: failed to close connection %s: %v", c.GetID(), err)
			}
		}
	}()
	go func() {
		<-ctx.Done()
		log.Printf("ws: context done, disconnecting %s", c.GetID())
		err := c.Close()
		if err != nil {
			log.Printf("ws: failed to close connection %s: %v", c.GetID(), err)
		}
	}()
	for {
		if errorMsg != nil {
			err := c.WriteJSON(errorMsg)
			if err != nil {
				log.Printf("ws: failed to send error message on connection %s with data %v: %v", c.GetID(), errorMsg, err)
			}
			errorMsg = nil
		}

		var wsMsg WSMessage
		err := c.ReadJSON(&wsMsg)
		if err != nil {
			if _, ok := err.(*websocket.CloseError); ok {
				log.Printf("ws: closed connection %s", c.GetID())
				c.CancelContext()
				c.Broker().Publish(rids.Spike().EventSocketDisconnected(c.GetID()), nil, c.GetSessionToken())
				return
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
}
