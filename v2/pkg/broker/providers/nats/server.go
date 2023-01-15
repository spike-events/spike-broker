package nats

import (
	"time"

	"github.com/nats-io/nats-server/v2/server"
)

var defaultNatsOptions = server.Options{
	Host:       "127.0.0.1",
	Port:       4222,
	MaxPayload: 100 * 1024 * 1024,
	MaxPending: 100 * 1024 * 1024,
}

func runServer(opts *server.Options) *server.Server {
	if opts == nil {
		opts = &defaultNatsOptions
	}
	s, err := server.NewServer(opts)
	if err != nil {
		panic(err)
	}

	s.ConfigureLogger()

	// Run server in Go routine.
	go s.Start()

	// Wait for accept loop(s) to be started
	if !s.ReadyForConnections(10 * time.Second) {
		panic("Unable to start Broker Server in Go Routine")
	}
	return s
}
