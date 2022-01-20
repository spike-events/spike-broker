package models

import (
	"github.com/gofrs/uuid"
	"gorm.io/datatypes"
	"time"
)

// SchemaVersion migration
type SchemaVersion struct {
	Base
	Service string `gorm:"unique_index:service_version"`
	Version int    `gorm:"unique_index:service_version"`
}

type TransactionSaga struct {
	Base
	TransactionSagaID uuid.UUID
	GoID              int
}

type TransactionSagaRow struct {
	CreatedAt time.Time
	GoID      int
	Type      string
	Metadata  datatypes.JSON
	Table     string
	RowID     uuid.UUID
}
