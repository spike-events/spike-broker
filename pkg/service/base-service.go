package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi"
	"github.com/gofrs/uuid"
	"github.com/nats-io/nats.go"
	"github.com/spike-events/spike-broker/pkg/models"
	"github.com/spike-events/spike-broker/pkg/models/migration"
	"github.com/spike-events/spike-broker/pkg/providers"
	"github.com/spike-events/spike-broker/pkg/rids"
	providers2 "github.com/spike-events/spike-broker/pkg/service/providers/nats"
	"github.com/spike-events/spike-broker/pkg/service/request"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	LockNamePrepend = "nats-mutex"
)

type Endpoint interface {
	Register(b *Base)
}

// Service base
type Service interface {
	Start()
	StartAfterDependency(dep string)
	AllServicesStarted()
	HandlerFunc(r *chi.Mux)
	Stop()
	Rid() rids.BaseRid
	SetOptions(options models.ProxyOptions)
	SetContext(ctx context.Context)
	Dependencies() []string
	TryLock(lockName ...string) bool
	Lock(lockName ...string)
	Unlock(lockName ...string)
}

// Base service
type Base struct {
	ctx      context.Context
	key      uuid.UUID
	db       *gorm.DB
	provider request.Provider
	quit     chan bool
	debug    string

	rid rids.BaseRid

	options models.ProxyOptions
}

// AddEndpoint new endpoint
func (s *Base) AddEndpoint(endpoint Endpoint) {
	endpoint.Register(s)
}

// NewBaseService nova instancia base
func NewBaseService(db *gorm.DB, key uuid.UUID, rid rids.BaseRid) *Base {
	return &Base{
		db:    db,
		key:   key,
		rid:   rid,
		debug: os.Getenv("API_LOG_LEVEL"),
	}
}

// SetOptions set options config
func (s *Base) SetOptions(options models.ProxyOptions) {
	s.options = options
}

func (s *Base) HandlerFunc(r *chi.Mux) {

}

// SetContext set context on base
func (s *Base) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// DeferRecover metodo para defer de verificação de erro
func (s *Base) DeferRecover() {
	if p := recover(); p != nil {
		fmt.Fprintln(os.Stderr, p, string(debug.Stack()))
	}
}

func (s *Base) TryLock(lockName ...string) bool {
	if s.debug == "DEBUG" {
		log.Printf("api: entering lock on service %s", s.rid.Name())
	}

	var lockNameString string
	if len(lockName) > 0 {
		lockNameString = fmt.Sprintf("%s-%s", LockNamePrepend, lockName[0])
	} else {
		lockNameString = fmt.Sprintf("%s-%s", LockNamePrepend, s.rid.Name())
	}

	// Log places where lock has been stuck
	e := time.Now()
	if s.debug == "DEBUG" {
		defer func() {
			log.Printf("api: lock on service %s name %s aquired after %dms",
				s.Rid().Name(),
				lockNameString,
				time.Now().Sub(e).Milliseconds(),
			)
		}()
	}

	err := s.DB().Transaction(func(tx *gorm.DB) error {
		var lock models.APILock
		tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", lockNameString).First(&lock)

		// FIXME: Create better handling on orphaned locks
		if lock.UnlockedOn == nil && lock.LockedOn.After(time.Now().Add(-5*time.Minute)) {
			return fmt.Errorf("already locked")
		}

		lock.LockedOn = time.Now()
		lock.Name = lockNameString
		lock.LockedBy = s.ServiceKey()
		lock.UnlockedOn = nil
		err := tx.Save(&lock).Error
		if err != nil {
			return err
		}
		return nil
	})

	if err == nil {
		return true
	}
	return false
}

// Lock na instância geral do serviço
func (s *Base) Lock(lockName ...string) {
	for {
		if s.TryLock(lockName...) {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// Unlock desbloqueia uso do serviço
func (s *Base) Unlock(lockName ...string) {
	if s.debug == "DEBUG" {
		log.Printf("api: entering unlock on service %s", s.rid.Name())
	}

	var lockNameString string
	if len(lockName) > 0 {
		lockNameString = fmt.Sprintf("%s-%s", LockNamePrepend, lockName[0])
	} else {
		lockNameString = fmt.Sprintf("%s-%s", LockNamePrepend, s.rid.Name())
	}

	// Log places where lock has been stuck
	e := time.Now()
	if s.debug == "DEBUG" {
		defer func() {
			log.Printf("api: lock on service %s name %s released after %dms",
				s.Rid().Name(),
				lockNameString,
				time.Now().Sub(e).Milliseconds(),
			)
		}()
	}

	_ = s.DB().Transaction(func(tx *gorm.DB) error {
		var lock models.APILock
		tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", lockNameString).First(&lock)

		if lock.LockedBy != s.ServiceKey() || lock.UnlockedOn != nil {
			panic("invalid lock state")
		}

		now := time.Now()
		lock.UnlockedOn = &now
		err := tx.Save(&lock).Error
		if err != nil {
			panic(err)
		}
		return nil
	})
}

func (s *Base) Broker() request.Provider {
	return s.provider
}

func (s *Base) BrokerInstance() interface{} {
	return s.provider
}

func (s *Base) DB() *gorm.DB {
	return s.db
}

func (s *Base) ServiceKey() uuid.UUID {
	return s.key
}

func (s *Base) Dependencies() []string {
	return []string{"route"}
}

func (s *Base) Init(migration migration.Migration, subscribers ...func()) {
	if s.rid == nil {
		panic("rid not registered")
	}

	switch providers.ProviderType(os.Getenv("PROVIDER")) {
	case providers.NatsProvider:
		natsURL := nats.DefaultURL
		if len(s.options.NatsConfig.NatsURL) > 0 {
			natsURL = s.options.NatsConfig.NatsURL
		}
		var natsDebug providers2.Config
		if os.Getenv("API_LOG_LEVEL") == "DEBUG" {
			natsDebug = providers2.ConfigEnableDebug
		}
		s.provider = providers2.NewNatsConn(natsURL, natsDebug)
	// case providers.CustomProvider:
	//	if s.options.CustomProvider == nil {
	//		panic("invalid provider")
	//	}
	//	s.provider = s.options.CustomProvider
	// case providers.KafkaProvider:
	//	s.provider = providerKafka.NewKafkaConn(s.options.KafkaConfig, s.ctx)
	// case providers.SpikeProvider:
	//	s.provider = providerSpike.NewSpikeConn(s.options.SpikeConfig, s.ctx)
	default:
		panic(fmt.Errorf("invalid provider: %v", os.Getenv("PROVIDER")))
	}

	if len(subscribers) > 0 {
		subscribers[0]()
	}

	if migration != nil {
		s.Lock("api-migration")
		err := migration.Migration(s.DB(), s.provider)
		s.Unlock("api-migration")
		if err != nil {
			panic(err)
		}
	}
}

// StartAfterDependency defaults to do nothing
func (s *Base) StartAfterDependency(_ string) {}

// AllServicesStarted defaults to do nothing
func (s *Base) AllServicesStarted() {}

// Stop service
func (s *Base) Stop() {
	s.provider.Close()
}

// ParseMessage data
func (s *Base) ParseMessage(data []byte, p interface{}) {
	json.Unmarshal(data, p)
}

// ServiceType retorna nome do serviço
func (s *Base) ServiceType() string {
	return s.rid.Name()
}

func (s *Base) Rid() rids.BaseRid {
	return s.rid
}
