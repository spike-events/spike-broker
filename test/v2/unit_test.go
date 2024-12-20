package v2

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/broker/providers/testProvider"
	"github.com/spike-events/spike-broker/v2/pkg/spike"
	"github.com/stretchr/testify/suite"
	"github.com/vincent-petithory/dataurl"
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
	authenticator := spike.NewTestAuthenticator(func(s []byte) ([]byte, bool) {
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
	t := spike.APITestAccess{
		Pattern: ServiceTestRid().TestReply(u.id),
		Err: func(index int, value interface{}) {
			u.Require().Equal(0, index, "invalid error index")
			err, valid := value.(broker.Error)
			u.Require().True(valid, "invalid error")
			u.Require().ErrorIs(broker.ErrorAccessDenied, err)
		},
	}
	err := u.svc.TestAccess(t)
	u.Require().NotNil(err)
}

func (u *UnitTest) TestFromMock() {
	testReplyMock := func(c broker.Call) {
		var id uuid.UUID
		err := json.Unmarshal(c.RawData(), &id)
		u.Require().Nil(err, "invalid payload providaded")
		c.OK(id)
	}

	t := spike.APITestRequestOrPublish{
		Pattern: ServiceTestRid().FromMock(),
		Ok: func(value ...interface{}) {
			u.Require().NotEmpty(value, "no value provider")
			_, valid := value[0].(*uuid.UUID)
			u.Require().True(valid, "value is not uuid")
		},
		Err: func(value interface{}) {
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

func (u *UnitTest) TestWithObjectPayload() {
	obj := map[string]interface{}{
		"attr1": 10,
		"attr2": "Ok",
	}
	t := spike.APITestRequestOrPublish{
		Pattern:    ServiceTestRid().CallWithObjPayload(),
		Repository: nil,
		Payload:    obj,
		Token:      nil,
		Ok: func(i ...interface{}) {
			u.Require().NotEmpty(i, "should have a return value")
			payload, valid := i[0].(*LocalPayload)
			u.Require().True(valid, "should have been a LocalPayload")
			u.Require().Equal(payload.Attr1, 10)
			u.Require().Equal(payload.Attr2, "Ok")
		},
		Err: func(i interface{}) {
			u.FailNow("Should have succeeded")
		},
		Mocks: testProvider.Mocks{},
	}
	err := u.svc.TestRequestOrPublish(t)
	u.Require().Nil(err, "Should have returned success")
}

func (u *UnitTest) TestExpectingFile() {
	t := spike.APITestRequestOrPublish{
		Pattern:    ServiceTestRid().CallExpectingFile(),
		Repository: "dumb-repository",
		Payload:    nil,
		Token:      nil,
		Ok: func(i ...interface{}) {
			u.Require().NotEmpty(i, "should have a return value")
			payload, valid := i[0].(*LocalPayload)
			u.Require().True(valid, "should have been a LocalPayload")
			u.Require().Equal(payload.Attr1, 10)
			u.Require().Equal(payload.Attr2, "Ok")
		},
		File: func(f *dataurl.DataURL) {
			u.Require().NotNil(f, "invalid nil file on response")
			u.Require().NotNil(f.Data, "invalid nil data on response")
			u.Require().Equal("image/webp", f.ContentType(), "invalid content type")
		},
		Err: func(i interface{}) {
			u.FailNow("Should have succeeded")
		},
		Mocks: testProvider.Mocks{},
	}
	err := u.svc.TestRequestOrPublish(t)
	u.Require().Nil(err, "Should have returned success")
}

func TestUnit(t *testing.T) {
	suite.Run(t, new(UnitTest))
}
