package socket

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/service"
	spike_utils "github.com/spike-events/spike-broker/v2/pkg/spike-utils"
)

func init() {
}

type Options struct {
	Handlers      []rids.Pattern
	Broker        broker.Provider
	Authenticator service.Authenticator
	Authorizer    service.Authorizer
	Logger        service.Logger
}

type WSConnection interface {
	GetID() fmt.Stringer
	GetToken() json.RawMessage
	GetHandlers() []rids.Pattern
	Context() context.Context
	CancelContext()
	WSConnection() *websocket.Conn
	Broker() broker.Provider
	Authenticator() service.Authenticator
	Authorizer() service.Authorizer
	GetSessionToken() json.RawMessage
	SetSessionToken(token json.RawMessage)
	SetSessionID(id string)
}

type wsConnection struct {
	ID            string `json:"id"`
	ctx           context.Context
	cancelCtx     context.CancelFunc
	ws            *websocket.Conn
	provider      broker.Provider
	authenticator service.Authenticator
	authorizer    service.Authorizer
	handlers      []rids.Pattern
	token         json.RawMessage
}

func (ws *wsConnection) GetID() fmt.Stringer {
	return spike_utils.Stringer(ws.ID)
}

func (ws *wsConnection) GetToken() json.RawMessage {
	return ws.token
}

func (ws *wsConnection) GetHandlers() []rids.Pattern {
	return ws.handlers
}

func (ws *wsConnection) Context() context.Context {
	return ws.ctx
}

func (ws *wsConnection) CancelContext() {
	ws.cancelCtx()
}
func (ws *wsConnection) WSConnection() *websocket.Conn {
	return ws.ws
}

func (ws *wsConnection) Broker() broker.Provider {
	return ws.provider
}

func (ws *wsConnection) Authenticator() service.Authenticator {
	return ws.authenticator
}

func (ws *wsConnection) Authorizer() service.Authorizer {
	return ws.authorizer
}

func (ws *wsConnection) GetSessionToken() json.RawMessage {
	return ws.token
}

func (ws *wsConnection) SetSessionToken(token json.RawMessage) {
	ws.token = token
}

func (ws *wsConnection) SetSessionID(id string) {
	ws.ID = id
}
func newConnection(conn *websocket.Conn, options Options) WSConnection {
	id, _ := uuid.NewV4()
	inCtx, cancel := context.WithCancel(context.Background())
	return &wsConnection{
		ID:            id.String(),
		ctx:           inCtx,
		cancelCtx:     cancel,
		ws:            conn,
		provider:      options.Broker,
		authenticator: options.Authenticator,
		authorizer:    options.Authorizer,
		handlers:      options.Handlers,
	}
}
