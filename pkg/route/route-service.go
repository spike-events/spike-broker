package route

import (
	"context"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/gofrs/uuid"
	"github.com/spike-events/spike-broker/pkg/models"
	"github.com/spike-events/spike-broker/pkg/rids"
	"github.com/spike-events/spike-broker/pkg/route/migration"
	"github.com/spike-events/spike-broker/pkg/service"
	"github.com/spike-events/spike-broker/pkg/service/request"
	"gorm.io/gorm"
	"log"
	"os"
	"sync"
)

type routeService struct {
	*service.Base

	auths      []*service.AuthRid
	Cancel     context.CancelFunc
	terminated chan bool

	conf           models.ProxyOptions
	m              *sync.Mutex
	restart        chan bool
	quitRestartJob chan bool

	routes *chi.Mux
}

var routeServiceImpl *routeService

// NewRouteService instance route service
func NewRouteService(db *gorm.DB, key uuid.UUID, auths ...*service.AuthRid) *routeService {
	if routeServiceImpl == nil {
		routeServiceImpl = &routeService{
			Base:           service.NewBaseService(db, key, rids.Route()),
			auths:          auths,
			Cancel:         nil,
			terminated:     make(chan bool),
			conf:           models.ProxyOptions{},
			m:              &sync.Mutex{},
			restart:        make(chan bool),
			quitRestartJob: make(chan bool),
			routes:         nil,
		}
	}
	return routeServiceImpl
}

func NewService(db *gorm.DB, key uuid.UUID) service.Service {
	return NewRouteService(db, key)
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

		s.Broker().RegisterMonitor(rids.Route().EventSocketConnected())
		s.Broker().RegisterMonitor(rids.Route().EventSocketDisconnected())
	})
}

func (s *routeService) AllServicesStarted() {
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

func (s *routeService) validateToken(r *request.CallRequest) {
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

func (s *routeService) userHavePermission(r *request.CallRequest) {
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

func (s *routeService) ready(r *request.CallRequest) {
	/* Test Database */
	var schemaVersion models.SchemaVersion
	err := s.DB().Find(&schemaVersion).Error
	if err != nil {
		r.Error(err)
		return
	}
	r.OK()
}
