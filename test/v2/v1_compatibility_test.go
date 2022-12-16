package v2

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/nats-io/nats.go"
	spikebroker "github.com/spike-events/spike-broker"
	"github.com/spike-events/spike-broker/pkg/models"
	"github.com/spike-events/spike-broker/pkg/providers"
	"github.com/spike-events/spike-broker/pkg/service"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	nats2 "github.com/spike-events/spike-broker/v2/pkg/broker/providers/nats"
	"github.com/spike-events/spike-broker/v2/pkg/spike"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type V1Test struct {
	suite.Suite
	id     uuid.UUID
	ctx    context.Context
	broker broker.Provider
}

func (v1 *V1Test) SetupSuite() {
	os.Setenv("TIMEOUT", strconv.Itoa(int((10 * time.Minute).Milliseconds())))

	// Setup v2 router
	// Create instance ID
	id, err := uuid.NewV4()
	if err != nil {
		v1.FailNow("failed to create test instance ID:", err)
		return
	}
	v1.id = id

	v1.ctx = context.Background()

	// Use default logger to stderr
	logger := log.New(os.Stderr, "test", log.LstdFlags)

	// Test Authenticator (validates token)
	authenticator := NewAuthenticator()

	// Test Authorizer (validates route access)
	authorizer := NewAuthorizer()

	// Initialize NATS
	v1.broker = nats2.NewNatsProvider(nats2.Config{
		LocalNats:      true,
		LocalNatsDebug: false,
		LocalNatsTrace: false,
		DebugLevel:     0,
		Logger:         logger,
	})

	// Initialize Spike providing Service
	spkService := spike.NewAPIService()
	err = spkService.RegisterService(spike.Options{
		Service:       NewServiceTest(v1.broker, logger),
		Authenticator: authenticator,
		Authorizer:    authorizer,
		Timeout:       10 * time.Minute,
	})
	if err != nil {
		v1.FailNow("failed to initialize the API Service:", err)
		return
	}

	if err = spkService.StartService(); err != nil {
		v1.FailNow("failed to start the service")
		return
	}

	// Setup v1 router
	os.Setenv("PROVIDER", string(providers.NatsProvider))

	options := models.ProxyOptions{
		NatsConfig: &models.NatsConfig{
			NatsURL: nats.DefaultURL,
		},
		Services: []string{"v1Service", "route"},
	}

	services := []spikebroker.HandlerService{
		NewV1Service,
	}

	if err = os.RemoveAll("gorm.db"); err != nil {
		log.Printf("could not remove gorm.db old database: %s", err)
	}
	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	_, connected, err := spikebroker.NewProxyServer(db, services,
		func(db *gorm.DB, key uuid.UUID) service.Auth {
			return &auth{}
		}, options)
	if err != nil {
		panic(err)
	}

	<-connected
}

func (v1 *V1Test) TestV1ToV2Request() {
	id, _ := uuid.NewV4()
	var retID uuid.UUID
	err := RequestV1(V1Service().CallV2Service(id.String()), nil, &retID)
	v1.Nil(err, "failed")
	v1.Equal(id.String(), retID.String(), "invalid response")
}

func (v1 *V1Test) TestV2ToV1Request() {
	err := v1.broker.Request(ServiceTestRid().CallV1(), nil, nil, []byte("token-string"))
	v1.Nil(err, "failed")
}

func (v1 *V1Test) TestV1SendErrorToV2() {
	err := v1.broker.Request(V2RidForV1Service().AnswerV2Forbidden(), nil, nil, []byte("token-string"))
	v1.Equal(err.Code(), http.StatusForbidden, "invalid response")
}

func (v1 *V1Test) TearDownSuite() {
	if err := os.RemoveAll("gorm.db"); err != nil {
		log.Printf("could not remove gorm.db old database: %s", err)
	}
	v1.broker.Close()
}

func TestV1Compatibility(t *testing.T) {
	suite.Run(t, new(V1Test))
}
