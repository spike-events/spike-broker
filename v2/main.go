package spikebroker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofrs/uuid"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/models"
	"github.com/spike-events/spike-broker/v2/pkg/route"
	"github.com/spike-events/spike-broker/v2/pkg/service"
	"gorm.io/gorm"
)

// NewProxyServer new service proxy
func NewProxyServer(db *gorm.DB, options broker.SpikeOptions) (ctx context.Context, connected chan bool, err error) {
	if db == nil {
		err = fmt.Errorf("db instance is required")
		return
	}

	if options.Logger == nil {
		options.Logger = log.Default()
	}

	logger := options.Logger
	logger.Println(">> AutoMigrate infrastructure")
	err = db.AutoMigrate(&models.SchemaVersion{}, &models.APILock{})
	if err != nil {
		logger.Println(err)
	}
	defer logger.Println("<< AutoMigrate infrastructure")

	err = options.IsValid()
	if err != nil {
		return
	}

	servicesType := make(map[string]service.Service)
	for _, srv := range options.Services {
		servicesType[srv.Rid().Name()] = srv
	}

	if options.Developer {
		if config, ok := options.ProviderConfig.(broker.NatsOptions); ok {
			runDefaultServer(config.LocalNatsDebug, config.LocalNatsTrace)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	connected = make(chan bool)

	go func() {
		stopSignal := make(chan os.Signal, 1)
		signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-stopSignal
			if options.StopTimeout == 0 {
				options.StopTimeout = 60 * time.Second
			}
			logger.Printf("main: terminating with sig %s, giving %f seconds to WebSocket handling",
				sig.String(), options.StopTimeout.Seconds())
			time.Sleep(options.StopTimeout)
			cancel()
		}()

		key, _ := uuid.NewV4()
		v5 := uuid.NewV5(key, "github.com/spike-events/spike-broker")

		routerService := route.NewService()
		routerService := route.NewRouteService(db, v5)

		if servicesType[routerService.Rid().Name()] == routerService.Rid().Name() || options.Developer {
			patterns := routerService.Rid().Patterns(routerService.Rid())
			routerService.AddAuthRid(&service.AuthRid{
				Service:  routerService.Rid().Name(),
				Auth:     authService,
				Patterns: patterns,
			})
			routerService.SetOptions(options)
			routerService.SetContext(ctx)
			routerService.Start()
			delete(servicesType, routerService.Rid().Name())
		}

		startedList := make([]service.Service, 0, len(services))
		startDeps := make([]func(), 0)
		startAllDeps := make([]func(), 0)
		for _, srv := range services {
			s := srv(db, v5)
			if servicesType[s.Rid().Name()] == s.Rid().Name() || options.Developer {
				startedList = append(startedList, s)
				patterns := s.Rid().Patterns(s.Rid())
				routerService.AddAuthRid(&service.AuthRid{Service: s.Rid().Name(), Auth: authService, Patterns: patterns})
				s.SetOptions(options)
				s.SetContext(ctx)
				logger.Println(">> Starting", s.Rid().Name())
				s.Start()
				logger.Println("<< Started", s.Rid().Name())
				go func() {
					<-ctx.Done()
					logger.Println(">> Stopping", s.Rid().Name())
					s.Stop()
					logger.Println("<< Stopped", s.Rid().Name())
				}()
				startAllDeps = append(startAllDeps, func() {
					logger.Println(">> Started", s.Rid().Name())
					s.Started()
					logger.Println("<< allServicesStarted", s.Rid().Name())
				})
			}
		}

		for _, depsStart := range startDeps {
			depsStart()
		}

		for _, allSrvStart := range startAllDeps {
			allSrvStart()
		}

		// Inform Route Service
		routerService.Started()
		connected <- true

		// Wait for context completion
		<-ctx.Done()
		logger.Printf("main: waiting for services to shutdown...")
		routerService.Stop()
		for _, s := range startedList {
			s.Stop()
		}
		logger.Printf("main: all services stopped")
	}()

	return
}

// defaultNatsOptions are default options for the unit tests.
var defaultNatsOptions = server.Options{
	Host:       "127.0.0.1",
	Port:       4222,
	MaxPayload: 100 * 1024 * 1024,
	MaxPending: 100 * 1024 * 1024,
}

// RunDefaultServer starts a new Go routine based server using the default options
func runDefaultServer(debug, trace bool) *server.Server {
	opts := &defaultNatsOptions
	opts.Debug = debug
	opts.Trace = trace
	return runServer(opts)
}

// RunServer starts a new Go routine based server
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
