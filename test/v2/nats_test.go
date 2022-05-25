package v2

import (
	"context"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/broker/providers/nats"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/spike"
	"github.com/stretchr/testify/suite"
)

type NatsTest struct {
	suite.Suite
	id     uuid.UUID
	ctx    context.Context
	spike  spike.APIService
	http   spike.HttpServer
	broker broker.Provider
}

func (s *NatsTest) TearDownSuite() {
	s.spike.Stop()
	s.http.Shutdown()
	s.broker.Close()
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

	// Test Authenticator (validates token)
	authenticator := NewAuthenticator()

	// Test Authorizer (validates route access)
	authorizer := NewAuthorizer()

	// Initialize NATS
	s.broker = nats.NewNatsProvider(nats.Config{
		LocalNats:      true,
		LocalNatsDebug: false,
		LocalNatsTrace: false,
		DebugLevel:     0,
		Logger:         logger,
	})

	// Initialize Spike providing Service
	s.spike = spike.NewAPIService()
	err = s.spike.RegisterService(spike.Options{
		Service:       NewServiceTest(s.broker, logger),
		Authenticator: authenticator,
		Authorizer:    authorizer,
		Timeout:       2 * time.Minute,
	})
	if err != nil {
		s.FailNow("failed to initialize the API Service:", err)
		return
	}

	if err = s.spike.StartService(); err != nil {
		s.FailNow("failed to start the service")
		return
	}

	// Initialize HTTP Server
	s.http = spike.NewHttpServer(s.ctx, spike.HttpOptions{
		Broker:        s.broker,
		Resources:     []rids.Resource{ServiceTestRid()},
		Authenticator: authenticator,
		Authorizer:    authorizer,
		WSPrefix:      "ws",
		Logger:        logger,
		Address:       ":3333",
	})

	if err = s.http.ListenAndServe(); err != nil {
		s.FailNow("failed to start http server:", err)
		return
	}
}

func (s *NatsTest) TestReplyFailNoToken() {
	var id uuid.UUID
	err := Request(ServiceTestRid().TestReply(s.id), s, &id)
	s.Require().NotNilf(err, "should return error")
	s.Require().Equal(http.StatusUnauthorized, err.Code(), "incorrect error")
}

func (s *NatsTest) TestReplyFailInvalidToken() {
	var id uuid.UUID
	err := Request(ServiceTestRid().TestReply(s.id), s, &id, "invalid-token")
	s.Require().NotNilf(err, "should return error")
	s.Require().Equal(http.StatusUnauthorized, err.Code(), "incorrect error")
}

func (s *NatsTest) TestReplyWithToken() {
	var id uuid.UUID
	err := Request(ServiceTestRid().TestReply(s.id), s, &id, "token-string")
	s.Require().ErrorIs(err, nil, "error response")
	s.Require().Equal(s.id, id, "invalid response")
}

func (s *NatsTest) TestInternalServiceCall() {
	var id uuid.UUID
	err := Request(ServiceTestRid().FromMock(), nil, &id, "token-string")
	s.Require().ErrorIs(err, nil, "error response")
	s.Require().NotEqual(id, uuid.Nil, "invalid id received")
}

func TestNats(t *testing.T) {
	suite.Run(t, new(NatsTest))
}
