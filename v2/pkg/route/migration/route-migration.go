package migration

import (
	"github.com/spike-events/spike-broker/v2/pkg/models/migration"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/service/request"
	"gorm.io/gorm"
)

type srv1Migration struct {
	migration.Base
}

type v1 struct{ migration.BaseVersion }

func (m *v1) Migrate(db *gorm.DB) error {
	return nil
}

// NewRoute migration
func NewRoute() migration.Migration {
	return &srv1Migration{}
}

func (a *srv1Migration) Versions() []migration.Versions {
	return []migration.Versions{&v1{}}
}

func (a *srv1Migration) Migration(db *gorm.DB, provider request.Provider) error {
	return a.Migrate(db, provider, rids.Route().Name(), a.Versions())
}
