package nats

import "github.com/spike-events/spike-broker/v2/pkg/service"

type Config struct {
	LocalNats      bool
	LocalNatsDebug bool
	LocalNatsTrace bool
	NatsURL        string
	DebugLevel     int
	Logger         service.Logger
}
