package route

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/go-chi/chi"
	"github.com/gofrs/uuid"
	"github.com/spike-events/spike-broker/pkg/service/request"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/models"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/route/migration"
	"github.com/spike-events/spike-broker/v2/pkg/service"
)

type routeService struct {
	*service.Base

	auths      []*service.AuthRid
	Cancel     context.CancelFunc
	terminated chan bool

	m              *sync.Mutex
	restart        chan bool
	quitRestartJob chan bool

	routes *chi.Mux
}

var routeServiceImpl *routeService

// NewRouteService instance route service
func NewRouteService() service.Service {
	if routeServiceImpl == nil {
		routeServiceImpl = &routeService{
			terminated:     make(chan bool),
			m:              &sync.Mutex{},
			restart:        make(chan bool),
			quitRestartJob: make(chan bool),
		}
	}
	return routeServiceImpl
}

func (s *routeService) Int(instanceKey uuid.UUID, repo interface{}, broker request.Provider,
	options broker.SpikeOptions) service.Service {
	s.Base = service.NewBaseService(instanceKey, repo, rids.Route(), broker, options)
	return s
}

func (s *routeService) AddAuthRid(auth *service.AuthRid) {
	s.auths = append(s.auths, auth)
}

// Dependencies return no dependencies for route service
func (s *routeService) Dependencies() []string {
	return []string{}
}

// Start service route
func (s *routeService) Start() {
	s.Init(migration.NewRoute(), func() {
		/* Validate token and UserHavePermission are very busy, so run at least 5 parallel handlers */
		s.Broker().Subscribe(rids.Route().ValidateToken(), s.validateToken)
		s.Broker().Subscribe(rids.Route().UserHavePermission(), s.userHavePermission)
		s.Broker().Subscribe(rids.Route().Ready(), s.ready)
	})
}

func (s *routeService) Started() {
	fmt.Println("Restarting Http server")
	started := s.serve()
	<-started
	close(started)
}

func (s *routeService) Stop() {
	if s.Cancel != nil {
		s.Cancel()
	}
	s.quitRestartJob <- true
	close(s.quitRestartJob)
	close(s.restart)
	s.Base.Stop()
}

func (s *routeService) validateToken(r *request.Call) {
	if os.Getenv("API_LOG_LEVEL") == "DEBUG" {
		log.Printf("route: validating token")
	}
	if len(s.auths) > 0 {
		if processed, ok := s.auths[0].Auth.ValidateToken(r.Token); ok {
			if os.Getenv("API_LOG_LEVEL") == "DEBUG" {
				log.Printf("route: token validated")
			}
			type rs struct {
				Token string
			}
			r.OK(rs{processed})
			return
		}
	}
	if os.Getenv("API_LOG_LEVEL") == "DEBUG" {
		log.Printf("route: token validation failed")
	}
	r.ErrorRequest(&request.ErrorStatusUnauthorized)
}

func (s *routeService) userHavePermission(r *request.Call) {
	var req models.HavePermissionRequest
	_ = r.ParseData(&req)
	if os.Getenv("API_LOG_LEVEL") == "DEBUG" {
		log.Printf("route: validating permission for %s", req.Endpoint)
	}
	if len(s.auths) > 0 {
		if !s.auths[0].Auth.UserHavePermission(r) {
			if os.Getenv("API_LOG_LEVEL") == "DEBUG" {
				log.Printf("route: do not have permission for %s", req.Endpoint)
			}
			r.ErrorRequest(&request.ErrorStatusForbidden)
			return
		}
	}
	if os.Getenv("API_LOG_LEVEL") == "DEBUG" {
		log.Printf("route: permission for %s validated", req.Endpoint)
	}
	r.OK()
}

func (s *routeService) ready(r *request.Call) {
	/* Test Database */
	var schemaVersion models.SchemaVersion
	err := s.DB().Find(&schemaVersion).Error
	if err != nil {
		r.Error(err)
		return
	}
	r.OK()
}
