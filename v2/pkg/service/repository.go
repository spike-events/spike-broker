package service

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var LockNamePrepend = "spike-v2"

type Repository interface {
	TryLock(lockName ...string) bool
	Lock(lockName ...string)
	Unlock(lockName ...string)
}

func NewGormRepository(id uuid.UUID, service string, db *gorm.DB) Repository {
	return &GormRepository{
		id:      id,
		db:      db,
		service: service,
	}
}

type GormRepository struct {
	id      uuid.UUID
	db      *gorm.DB
	service string
}

func (s *GormRepository) TryLock(lockName ...string) bool {
	var lockNameString string
	if len(lockName) > 0 {
		lockNameString = fmt.Sprintf("%s-%s", LockNamePrepend, lockName[0])
	} else {
		lockNameString = fmt.Sprintf("%s-%s", LockNamePrepend, s.service)
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var lock APILock
		tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", lockNameString).First(&lock)

		// FIXME: Create better handling on orphaned locks
		if lock.UnlockedOn == nil && lock.LockedOn.After(time.Now().Add(-5*time.Minute)) {
			return fmt.Errorf("already locked")
		}

		lock.LockedOn = time.Now()
		lock.Name = lockNameString
		lock.LockedBy = s.id
		lock.UnlockedOn = nil
		err := tx.Save(&lock).Error
		if err != nil {
			panic(err)
		}
		return nil
	})

	if err == nil {
		return true
	}
	return false
}

func (s *GormRepository) Lock(lockName ...string) {
	for {
		if s.TryLock(lockName...) {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (s *GormRepository) Unlock(lockName ...string) {
	var lockNameString string
	if len(lockName) > 0 {
		lockNameString = fmt.Sprintf("%s-%s", LockNamePrepend, lockName[0])
	} else {
		lockNameString = fmt.Sprintf("%s-%s", LockNamePrepend, s.service)
	}

	_ = s.db.Transaction(func(tx *gorm.DB) error {
		var lock APILock
		tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", lockNameString).First(&lock)

		if lock.LockedBy != s.id || lock.UnlockedOn != nil {
			return fmt.Errorf("invalid lock state")
		}

		now := time.Now()
		lock.UnlockedOn = &now
		err := tx.Save(&lock).Error
		if err != nil {
			return err
		}
		return nil
	})
}
