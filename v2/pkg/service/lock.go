package service

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

type APILock struct {
	Name       string `gorm:"primaryKey"`
	LockedOn   time.Time
	UnlockedOn *time.Time
	LockedBy   uuid.UUID
}
