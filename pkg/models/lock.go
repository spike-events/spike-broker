package models

import (
	"github.com/gofrs/uuid"
	"time"
)

type APILock struct {
	Name       string `gorm:"primaryKey"`
	LockedOn   time.Time
	UnlockedOn *time.Time
	LockedBy   uuid.UUID
}
