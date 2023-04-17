package models

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spike-events/spike-broker/pkg/providers"

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
	//if providers.ProviderType(os.Getenv("PROVIDER")) == providers.KafkaProvider && p.KafkaConfig == nil {
	//	return fmt.Errorf("kafka config is required")
	//}
	//if providers.ProviderType(os.Getenv("PROVIDER")) == providers.SpikeProvider && p.SpikeConfig == nil {
	//	return fmt.Errorf("spike config is required")
	//}
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

// BeforeCreate will set a UUID rather than numeric ID.
func (base *Base) BeforeCreate(_ *gorm.DB) error {
	if base.ID == uuid.Nil {
		nonce, err := uuid.NewV4()
		if err != nil {
			return err
		}
		base.ID = nonce
	}
	return nil
}

// IsValid validates struct contents based on annotations
func (base *Base) IsValid() error {
	var validate *validator.Validate
	validate = validator.New()
	return validate.Struct(base)
}

// IsValid varificar se o Atividade é valido
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
