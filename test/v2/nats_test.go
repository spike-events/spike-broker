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
	"github.com/vincent-petithory/dataurl"
)

type NatsTest struct {
	suite.Suite
	id            uuid.UUID
	ctx           context.Context
	spike         spike.APIService
	http          spike.HttpServer
	serviceBroker broker.Provider
	httpBroker    broker.Provider
}

func (s *NatsTest) TearDownSuite() {
	s.spike.Stop()
	s.http.Shutdown()
	s.serviceBroker.Close()
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
	s.serviceBroker = nats.NewNatsProvider(nats.Config{
		LocalNats:      true,
		LocalNatsDebug: false,
		LocalNatsTrace: false,
		DebugLevel:     0,
		Logger:         logger,
	})

	s.httpBroker = nats.NewNatsProvider(nats.Config{
		LocalNats: false,
		Logger:    logger,
	})

	// Initialize Spike providing Service
	s.spike = spike.NewAPIService()
	err = s.spike.RegisterService(spike.Options{
		Service:       NewServiceTest(s.serviceBroker, logger),
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
		Broker:        s.httpBroker,
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

func (s *NatsTest) TestRootRID() {
	var ret struct {
		Field string
		Value int
	}
	err := Request(ServiceTestRid().RootEP(), nil, &ret, "token-string")
	s.Require().Nil(err, "should have succeeded")
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
	err := Request(ServiceTestRid().FromMock(), s, &id, "token-string")
	s.Require().ErrorIs(err, nil, "error response")
	s.Require().NotEqual(id, uuid.Nil, "invalid id received")
}

func (s *NatsTest) TestCallInvalidRoute() {
	err := Request(ServiceTestRid().NoHandler(), nil, nil, "invalid-token")
	s.Require().NotNil(err, "should have failed")
}

func (s *NatsTest) TestReplyWithTokenAndPayload() {
	payload := map[string]interface{}{
		"attr1": 10,
		"attr2": "Ok",
	}
	var respPayload map[string]interface{}
	err := Request(ServiceTestRid().CallWithObjPayload(), payload, &respPayload, "token-string")
	s.Require().ErrorIs(err, nil, "error response")
	s.Require().Equal(float64(10), respPayload["attr1"], "invalid response")
	s.Require().Equal("Ok", respPayload["attr2"], "invalid response")
}

func (s *NatsTest) TestExpectingFile() {
	var fData []byte
	err := Request(ServiceTestRid().CallExpectingFile(), nil, &fData, "token-string")
	s.Require().ErrorIs(err, nil, "error response")
	f, cErr := dataurl.DecodeString(dataurl.EncodeBytes(fData))
	s.Require().Nil(cErr, "should have been able to convert to dataURL")
	s.Require().NotNil(f.Data, "invalid returned data")
	s.Require().Equal("image/webp", f.ContentType(), "invalid returned type")
}
func TestNats(t *testing.T) {
	suite.Run(t, new(NatsTest))
}
