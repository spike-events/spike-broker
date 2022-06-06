package socket

import (
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
	GetToken() string
	GetHandlers() []rids.Pattern
	WSConnection() *websocket.Conn
	Broker() broker.Provider
	Authenticator() service.Authenticator
	Authorizer() service.Authorizer
	GetSessionToken() string
	SetSessionToken(token string)
	SetSessionID(id string)
}

type wsConnection struct {
	ID            string `json:"id"`
	ws            *websocket.Conn
	provider      broker.Provider
	authenticator service.Authenticator
	authorizer    service.Authorizer
	handlers      []rids.Pattern
	token         string
}

func (ws *wsConnection) GetID() fmt.Stringer {
	return spike_utils.Stringer(ws.ID)
}

func (ws *wsConnection) GetToken() string {
	return ws.token
}

func (ws *wsConnection) GetHandlers() []rids.Pattern {
	return ws.handlers
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

func (ws *wsConnection) GetSessionToken() string {
	return ws.token
}

func (ws *wsConnection) SetSessionToken(token string) {
	ws.token = token
}

func (ws *wsConnection) SetSessionID(id string) {
	ws.ID = id
}
func newConnection(conn *websocket.Conn, options Options) WSConnection {
	id, _ := uuid.NewV4()
	return &wsConnection{
		ID:            id.String(),
		ws:            conn,
		provider:      options.Broker,
		authenticator: options.Authenticator,
		authorizer:    options.Authorizer,
		handlers:      options.Handlers,
	}
}
