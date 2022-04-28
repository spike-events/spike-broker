package test

import (
	"context"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/spike-events/spike-broker/v2/pkg/broker/providers/nats"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/spike"
	"github.com/stretchr/testify/suite"
)

type NatsTest struct {
	suite.Suite
	id  uuid.UUID
	ctx context.Context
}

func (s *NatsTest) SetupSuite() {
	// Create instance ID
	id, err := uuid.NewV4()
	if err != nil {
		s.FailNow("failed to create test instance ID:", err)
		return
	}
	s.id = id

	s.ctx = context.Background()

	// Use default logger to stderr
	logger := log.New(os.Stderr, "test", log.LstdFlags)

	// Create an empty repository
	var repo interface{}

	// Test Authenticator (validates token)
	authenticator := NewAuthenticator()

	// Test Authorizer (validates route access)
	authorizer := NewAuthorizer()

	// Initialize NATS
	broker := nats.NewNatsProvider(nats.Config{
		LocalNats:      true,
		LocalNatsDebug: false,
		LocalNatsTrace: false,
		DebugLevel:     0,
		Logger:         logger,
	})

	// Initialize Spike providing Service
	spkService := spike.NewAPIService()
	err = spkService.Initialize(spike.Options{
		Service:       NewServiceTest(),
		Broker:        broker,
		Repository:    repo,
		Logger:        logger,
		Authenticator: authenticator,
		Authorizer:    authorizer,
		Timeout:       2 * time.Minute,
	})
	if err != nil {
		s.FailNow("failed to initialize the API Service:", err)
		return
	}

	if err = spkService.StartService(); err != nil {
		s.FailNow("failed to start the service")
		return
	}

	// Initialize HTTP Server
	httpServer := spike.NewHttpServer(s.ctx, spike.HttpOptions{
		Broker: broker,
		Handlers: []rids.Pattern{
			ServiceTestRid().TestReply(),
		},
		Authenticator: authenticator,
		Authorizer:    authorizer,
		WSPrefix:      "ws",
		Logger:        logger,
		Address:       ":3333",
	})

	if err = httpServer.ListenAndServe(); err != nil {
		s.FailNow("failed to start http server:", err)
		return
	}
}

func (s *NatsTest) TestReplyFailNoToken() {
	var id uuid.UUID
	err := Request(ServiceTestRid().TestReply(s.id), s, &id)
	s.Require().NotNilf(err, "should return error")
	s.Require().Equal(err.Code(), http.StatusUnauthorized, "incorrect error")
}

func (s *NatsTest) TestReplyFailInvalidToken() {
	var id uuid.UUID
	err := Request(ServiceTestRid().TestReply(s.id), s, &id, "invalid-token")
	s.Require().NotNilf(err, "should return error")
	s.Require().Equal(err.Code(), http.StatusUnauthorized, "incorrect error")
}

func (s *NatsTest) TestReplyWithToken() {
	var id uuid.UUID
	err := Request(ServiceTestRid().TestReply(s.id), s, &id, "token-string")
	s.Require().ErrorIs(err, nil, "error response")
	s.Require().Equal(id, s.id, "invalid response")
}

func TestNats(t *testing.T) {
	suite.Run(t, new(NatsTest))
}
