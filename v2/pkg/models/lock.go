package models

import (
	"time"

	"github.com/gofrs/uuid"
)

type APILock struct {
	Name       string `gorm:"primaryKey"`
	LockedOn   time.Time
	UnlockedOn *time.Time
	LockedBy   uuid.UUID
}
