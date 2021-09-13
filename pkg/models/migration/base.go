package migration

import (
	"github.com/spike-events/spike-broker/pkg/service/request"
	"reflect"
	"regexp"
	"strconv"

	"github.com/spike-events/spike-broker/pkg/models"
	"gorm.io/gorm"
)

// Migration interface
type Migration interface {
	Migration(*gorm.DB, request.Provider) error
	Migrate(*gorm.DB, request.Provider, string, []Versions) error
	Versions() []Versions
}

// Versions version migrate
type Versions interface {
	Save(*gorm.DB, request.Provider, string, int) error
	Migrate(db *gorm.DB) error
}

// Base migration
type Base struct{}

// NewBase new instance migration
func NewBase() *Base {
	return &Base{}
}

// Migrate run migration
func (m *Base) Migrate(db *gorm.DB, provider request.Provider, name string, versions []Versions) error {
	var schema models.SchemaVersion
	db.Where(&models.SchemaVersion{Service: name}).Order("version DESC").First(&schema)

	for _, item := range versions {
		itemStructName := reflect.TypeOf(item).Elem().Name()
		versionIndexStr := regexp.MustCompile("[^\\d]").ReplaceAllString(itemStructName, "")
		index, err := strconv.Atoi(versionIndexStr)
		if err != nil {
			panic(err)
		}
		if schema.Version < index {
			err = item.Migrate(db)
			if err != nil {
				panic(err)
			}
			err = db.Transaction(func(tx *gorm.DB) error {
				err = item.Save(tx, provider, name, index)
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
	}

	// FIXME: Should not panic on error and instead return
	return nil
}

// BaseVersion base migration version
type BaseVersion struct{}

// Save base
func (b *BaseVersion) Save(tx *gorm.DB, _ request.Provider, name string, version int) error {
	var schema models.SchemaVersion
	schema.Service = name
	schema.Version = version
	return tx.Create(&schema).Error
}
