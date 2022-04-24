package broker

import (
	"time"
)

// SpikeOptions options
type SpikeOptions struct {
	Developer   bool
	Provider    Provider
	StopTimeout time.Duration
}
