package service

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

// Service Interface allows implementing Service code that will be initialized by Spike
type Service interface {
	// Start will be called after all migration, config and handlers setup has been done, expecting it to return without
	// blocking
	Start(key uuid.UUID, ctx context.Context) error

	// Stop will be called when the service needs to be shutdown and will wait on returned channel up to 30 seconds then
	// it will cease control to operating system
	Stop() chan bool

	// Handlers must return all Subscription resources that will be handled by the Service
	Handlers() []broker.Subscription

	// Key returns the Service unique instance key
	Key() uuid.UUID

	// Rid return the rids.Resource that defines the Service Subscription methods
	Rid() rids.Resource

	// Broker returns the Broker Provider used to communicate with other Service
	Broker() broker.Provider

	// Logger returns the logger to be used by the Service
	Logger() Logger
}

type WithMonitors interface {
	// Monitors must return all Subscription that will handler events on the specified key group
	Monitors() map[string][]broker.Subscription
}

type WithEvents interface {
	// Events returns all events this service may trigger
	Events() []broker.Event
}

type WithRepository interface {
	// SetRepository allows Spike to inject another Repository. It must be idempotent.
	SetRepository(repository interface{}) error
}

type WithExternalAPI interface {
	// SetExternalAPI allows to define external API implementations to be mocked on tests
	SetExternalAPI(api string, implementation interface{}) error
}
