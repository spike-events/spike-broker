package providers

type ProviderType string

const (
	NatsProvider   ProviderType = "nats"
	KafkaProvider  ProviderType = "kafka"
	SpikeProvider  ProviderType = "spike"
)
