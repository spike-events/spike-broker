package test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/broker/providers/testProvider"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/spike"
	"github.com/stretchr/testify/suite"
)

type UnitTest struct {
	suite.Suite
	id  uuid.UUID
	ctx context.Context
	svc spike.APITestService
}

func (u *UnitTest) SetupSuite() {
	// Create instance ID
	id, err := uuid.NewV4()
	if err != nil {
		u.FailNow("failed to create test instance ID:", err)
		return
	}
	u.id = id

	u.ctx = context.Background()

	// Use default logger to stderr
	logger := log.New(os.Stderr, "test", log.LstdFlags)

	// Create an empty repository
	var repo interface{}

	// Test Authenticator (validates token)
	authenticator := spike.NewTestAuthenticator(func(s string) (string, bool) {
		return s, true
	})

	// Test Authorizer (validates route access)
	authorizer := spike.NewTestAuthorizer(func(c broker.Call) bool {
		return true
	})

	// Initialize Spike providing Service
	spkService := spike.NewAPITestService(
		nil,
		testProvider.Mocks{
			Requests:     map[string]testProvider.RequestMock{},
			RequestsRaw:  map[string]testProvider.RequestRawMock{},
			Publishes:    map[string]testProvider.RequestMock{},
			PublishesRaw: map[string]testProvider.RequestRawMock{},
		})
	err = spkService.Initialize(spike.Options{
		Service:       NewServiceTest(),
		Repository:    repo,
		Logger:        logger,
		Authenticator: authenticator,
		Authorizer:    authorizer,
		Timeout:       2 * time.Minute,
	})
	if err != nil {
		u.FailNow("failed to initialize the API Service:", err)
		return
	}

	if err = spkService.StartService(); err != nil {
		u.FailNow("failed to start the service")
		return
	}

	u.svc = spkService
}

func (u *UnitTest) TestFailReplyNoToken() {
	t := spike.APITestRequestOrPublish{
		Pattern:    ServiceTestRid().TestReply(u.id),
		Repository: nil,
		Payload:    nil,
		Token:      "",
		RequestOk:  func(value ...interface{}) {},
		AccessOk:   func(value ...interface{}) {},
		RequestErr: func(value interface{}) {},
		AccessErr:  func(value interface{}) {},
		Mocks: testProvider.Mocks{
			Requests:     map[string]testProvider.RequestMock{},
			RequestsRaw:  map[string]testProvider.RequestRawMock{},
			Publishes:    map[string]testProvider.RequestMock{},
			PublishesRaw: map[string]testProvider.RequestRawMock{},
		},
	}
	err := u.svc.TestRequestOrPublish(t)
	u.Require().NotNil(err)
}

func (u *UnitTest) TestFromMock() {
	testReplyMock := func(p rids.Pattern, payload interface{}, res interface{}, token ...string) broker.Error {
		id, converted := payload.(uuid.UUID)
		if !converted {
			return broker.ErrorInvalidParams
		}
		*res.(*uuid.UUID) = id
		return nil
	}

	t := spike.APITestRequestOrPublish{
		Pattern:    ServiceTestRid().FromMock(),
		Repository: nil,
		Payload:    nil,
		Token:      "",
		RequestOk:  func(value ...interface{}) {},
		AccessOk:   func(value ...interface{}) {},
		RequestErr: func(value interface{}) {},
		AccessErr:  func(value interface{}) {},
		Mocks: testProvider.Mocks{
			Requests: map[string]testProvider.RequestMock{
				ServiceTestRid().TestReply().EndpointName(): testReplyMock,
			},
			RequestsRaw:  map[string]testProvider.RequestRawMock{},
			Publishes:    map[string]testProvider.RequestMock{},
			PublishesRaw: map[string]testProvider.RequestRawMock{},
		},
	}
	err := u.svc.TestRequestOrPublish(t)
	u.Require().Nil(err)
}

func TestUnit(t *testing.T) {
	suite.Run(t, new(UnitTest))
}
