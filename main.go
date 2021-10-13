package apiproxynats

import (
	"context"
	"fmt"
	"github.com/spike-events/spike-broker/pkg/providers"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofrs/uuid"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/spike-events/spike-broker/pkg/models"
	"github.com/spike-events/spike-broker/pkg/route"
	"github.com/spike-events/spike-broker/pkg/service"
	"gorm.io/gorm"
)

var localNatsServer *server.Server

// HandlerService handler
type HandlerService func(db *gorm.DB, key uuid.UUID) service.Service

// HandlerAuthService required instance auth service
type HandlerAuthService func(db *gorm.DB, key uuid.UUID) service.Auth

// NewProxyServer new service proxy
func NewProxyServer(db *gorm.DB, services []HandlerService, handlerAuth HandlerAuthService,
	options models.ProxyOptions) (ctx context.Context, connected chan bool, err error) {
	if db == nil {
		err = fmt.Errorf("db instance is required")
		return
	}

	log.Println(">> AutoMigrate infrastructure")
	err = db.AutoMigrate(&models.SchemaVersion{}, &models.APILock{})
	if err != nil {
		log.Println(err)
	}
	defer log.Println("<< AutoMigrate infrastructure")

	err = options.IsValid()
	if err != nil {
		return
	}

	servicesType := make(map[string]string)
	for _, srv := range options.Services {
		servicesType[srv] = srv
	}

	if options.Developer {
		if providers.ProviderType(os.Getenv("PROVIDER")) == providers.NatsProvider {
			localNatsServer = runDefaultServer(options.NatsConfig.LocalNatsDebug, options.NatsConfig.LocalNatsTrace)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	connected = make(chan bool)

	go func() {
		receivedSignal := make(chan os.Signal, 1)
		signal.Notify(receivedSignal, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-receivedSignal
			if options.StopTimeout == 0 {
				options.StopTimeout = 60 * time.Second
			}
			log.Printf("main: terminating with sig %s, giving %f seconds to WebSocket handling",
				sig.String(), options.StopTimeout.Seconds())
			time.Sleep(options.StopTimeout)
			cancel()
		}()

		key, _ := uuid.NewV4()
		v5 := uuid.NewV5(key, "github.com/spike-events/spike-broker")

		authService := handlerAuth(db, v5)
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
				log.Println(">> Starting", s.Rid().Name())
				s.Start()
				log.Println("<< Started", s.Rid().Name())
				startDeps = append(startDeps, func() {
					log.Println(">> Processing Deps", s.Rid().Name())
					processDependencies(s)
					log.Println("<< Deps Processed", s.Rid().Name())

					go func() {
						<-ctx.Done()
						log.Println(">> Stopping", s.Rid().Name())
						s.Stop()
						log.Println("<< Stopped", s.Rid().Name())
					}()
				})
				startAllDeps = append(startAllDeps, func() {
					log.Println(">> AllServicesStarted", s.Rid().Name())
					s.AllServicesStarted()
					log.Println("<< allServicesStarted", s.Rid().Name())
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
		routerService.AllServicesStarted()
		connected <- true

		// Wait for context completion
		<-ctx.Done()

		for _, s := range startedList {
			s.Stop()
		}
	}()

	return
}

// defaultNatsOptions are default options for the unit tests.
var defaultNatsOptions = server.Options{
	Host:       "127.0.0.1",
	Port:       4222,
	MaxPayload: 100 * 1024 * 1024,
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

func processDependencies(s service.Service) {
	deps := s.Dependencies()
	for _, dep := range deps {
		log.Printf(">> %s processing dep %s\n", s.Rid().Name(), dep)
		s.StartAfterDependency(dep)
		log.Printf("<< %s processing dep %s\n", s.Rid().Name(), dep)
	}
}