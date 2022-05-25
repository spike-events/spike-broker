package v2

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

func (u *UnitTest) TearDownSuite() {
	u.svc.Stop()
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

	// Test Authenticator (validates token)
	authenticator := spike.NewTestAuthenticator(func(s string) (string, bool) {
		return s, true
	})

	// Test Authorizer (validates route access)
	authorizer := spike.NewTestAuthorizer(func(c broker.Call) bool {
		return true
	})

	// Test Broker
	tp := testProvider.NewTestProvider(u.ctx)

	// Initialize Spike providing Service
	spkService := spike.NewAPITestService(testProvider.Mocks{})
	err = spkService.RegisterService(spike.Options{
		Service:       NewServiceTest(tp, logger),
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
		Pattern: ServiceTestRid().TestReply(u.id),
		AccessErr: func(value interface{}) {
			err, valid := value.(broker.Error)
			u.Require().True(valid, "invalid error")
			u.Require().ErrorIs(broker.ErrorAccessDenied, err)
		},
	}
	err := u.svc.TestRequestOrPublish(t)
	u.Require().NotNil(err)
}

func (u *UnitTest) TestFromMock() {
	testReplyMock := func(p rids.Pattern, payload interface{}, res interface{}, token ...string) broker.Error {
		id, converted := payload.(uuid.UUID)
		u.Require().True(converted)
		*res.(*uuid.UUID) = id
		return nil
	}

	t := spike.APITestRequestOrPublish{
		Pattern: ServiceTestRid().FromMock(),
		RequestOk: func(value ...interface{}) {
			u.Require().NotEmpty(value, "no value provider")
			_, valid := value[0].(*uuid.UUID)
			u.Require().True(valid, "value is not uuid")
		},
		AccessOk: func(value ...interface{}) {
			u.Require().Empty(value)
		},
		RequestErr: func(value interface{}) {
			u.Require().Nil(value)
		},
		AccessErr: func(value interface{}) {
			u.Require().Nil(value)
		},
		Mocks: testProvider.Mocks{
			Requests: map[string]testProvider.RequestMock{
				ServiceTestRid().TestReply().EndpointName(): testReplyMock,
			},
		},
	}
	err := u.svc.TestRequestOrPublish(t)
	u.Require().Nil(err)
}

func TestUnit(t *testing.T) {
	suite.Run(t, new(UnitTest))
}
