package socket

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/spike-events/spike-broker/v2/pkg/service"
	"github.com/spike-events/spike-broker/v2/pkg/service/request"
)

func init() {
}

// WSConnection struct
type WSConnection struct {
	ID        string `json:"id"`
	ctx       context.Context
	cancelCtx context.CancelFunc
	ws        *websocket.Conn
	route     *service.Service
	provider  request.Provider
	auth      []*service.AuthRid
	token     string
}

func (ws *WSConnection) Context() context.Context {
	return ws.ctx
}

func (ws *WSConnection) CancelContext() {
	ws.cancelCtx()
}
func (ws *WSConnection) WSConnection() *websocket.Conn {
	return ws.ws
}

func (ws *WSConnection) Broker() request.Provider {
	return ws.provider
}

func (ws *WSConnection) GetSessionToken() string {
	return ws.token
}

func (ws *WSConnection) SetSessionToken(token string) {
	ws.token = token
}

func (ws *WSConnection) SetSessionID(id string) {
	ws.ID = id
}
func newConnection(conn *websocket.Conn, provider request.Provider, oauth ...*service.AuthRid) *WSConnection {
	id, _ := uuid.NewV4()
	inCtx, cancel := context.WithCancel(context.Background())
	return &WSConnection{
		ID:        id.String(),
		ctx:       inCtx,
		cancelCtx: cancel,
		ws:        conn,
		provider:  provider,
		auth:      oauth,
	}
}
