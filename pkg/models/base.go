package models

import (
	"encoding/json"
	"fmt"
	"github.com/spike-events/spike-broker/pkg/providers"
	"github.com/spike-events/spike-broker/pkg/utils"
	"os"
	"time"

	"github.com/go-playground/validator"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// ProxyOptions options
type ProxyOptions struct {
	Developer   bool
	Services    []string
	StopTimeout time.Duration
	NatsConfig  *NatsConfig
	KafkaConfig *KafkaConfig
	SpikeConfig *SpikeConfig
}

type KafkaConfig struct {
	KafkaURL          string
	Debug             bool
	NumPartitions     int
	ReplicationFactor int
}

type SpikeConfig struct {
	SpikeURL string
	Debug    bool
}

type NatsConfig struct {
	LocalNats      bool
	LocalNatsDebug bool
	LocalNatsTrace bool
	NatsURL        string
}

// IsValid validate options
func (p *ProxyOptions) IsValid() error {
	if p.NatsConfig == nil && providers.ProviderType(os.Getenv("PROVIDER")) == providers.NatsProvider {
		return fmt.Errorf("NatsConfig is required")
	}
	if p.NatsConfig != nil && !p.NatsConfig.LocalNats && len(p.NatsConfig.NatsURL) == 0 &&
		providers.ProviderType(os.Getenv("PROVIDER")) == providers.NatsProvider {
		return fmt.Errorf("NatsURL is required")
	}
	if len(p.Services) == 0 && !p.Developer {
		return fmt.Errorf("Services are required")
	}
	if providers.ProviderType(os.Getenv("PROVIDER")) == providers.KafkaProvider && p.KafkaConfig == nil {
		return fmt.Errorf("kafka config is required")
	}
	if providers.ProviderType(os.Getenv("PROVIDER")) == providers.SpikeProvider && p.SpikeConfig == nil {
		return fmt.Errorf("spike config is required")
	}
	//if providers.ProviderType(os.Getenv("PROVIDER")) == providers.CustomProvider && p.CustomProvider == nil {
	//	return fmt.Errorf("invalid provider, provider wasen't instanciated")
	//}
	return nil
}

// Base contains common columns for all tables.
type Base struct {
	ID        uuid.UUID      `json:"id,omitempty" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (base *Base) BeforeCreate(tx *gorm.DB) error {
	if base.ID == uuid.Nil {
		nonce, err := uuid.NewV4()
		if err != nil {
			return err
		}
		base.ID = nonce
	}
	tx.Create(&TransactionSagaRow{
		CreatedAt: time.Now(),
		GoID:      utils.GoID(),
		Table:     tx.Statement.Table,
		RowID:     base.ID,
		Type:      "INSERT",
	})
	return nil
}

func (base *Base) BeforeSave(tx *gorm.DB) error {
	typeSave := "UPDATE"
	if base.ID == uuid.Nil {
		nonce, err := uuid.NewV4()
		if err != nil {
			return err
		}
		base.ID = nonce
		typeSave = "INSERT"
	}
	if typeSave == "UPDATE" {
		tx.Exec(fmt.Sprintf(`INSERT INTO TransactionSagaRow (created_at, go_id, type, metadata, table, row_id) VALUES (?, (
					select row_to_json (row)
					from (select * from %v where id = ?) row
					))`, tx.Statement.Table), time.Now(), utils.GoID(), typeSave, base.ID, tx.Statement.Table, base.ID)
	} else {
		tx.Create(&TransactionSagaRow{
			CreatedAt: time.Now(),
			GoID:      utils.GoID(),
			Table:     tx.Statement.Table,
			RowID:     base.ID,
			Type:      typeSave,
		})
	}
	return nil
}

// IsValid validates struct contents based on annotations
func (base *Base) IsValid() error {
	var validate *validator.Validate
	validate = validator.New()
	return validate.Struct(base)
}

//IsValid varificar se o Atividade é valido
func IsValid(p interface{}) error {
	var validate *validator.Validate
	validate = validator.New()
	return validate.Struct(p)
}

// ToJSON convert json models
func ToJSON(p interface{}) []byte {
	rs, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return rs
}
