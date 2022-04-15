package route

import (
	"context"
	"log"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/route/handlers"
	"github.com/spike-events/spike-broker/v2/pkg/route/socket"
)

func (s *routeService) serve() chan bool {
	started := make(chan bool)
	go func() {
		var r *chi.Mux
		if s.routes == nil {
			r = chi.NewRouter()

			// A good base middleware stack
			r.Use(middleware.RequestID)
			r.Use(middleware.RealIP)
			r.Use(middleware.Logger)
			r.Use(middleware.Recoverer)
			r.Use(handlers.ContentTypeJSONMiddleware)

			corsOpts := cors.New(cors.Options{
				// AllowedOrigins: []string{"https://foo.com"}, // Use this to allow specific origin hosts
				AllowedOrigins: []string{"*"},
				// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
				AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
				AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Cache-Control"},
				ExposedHeaders:   []string{"Link"},
				AllowCredentials: true,
				MaxAge:           300, // Maximum value not ignored by any of major browsers
			})
			r.Use(corsOpts.Handler)

			r.Use(middleware.Timeout(60 * time.Second))
			r.Use(handlers.AuthMiddleware(s.auths...))
			r.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusOK)
			})

			r.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusOK)
			})

			// Register pprof handlers
			r.HandleFunc("/debug/pprof/", pprof.Index)
			r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			r.HandleFunc("/debug/pprof/profile", pprof.Profile)
			r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

			r.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
			r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
			r.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
			r.Handle("/debug/pprof/block", pprof.Handler("block"))
			r.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
			r.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
			r.Handle("/metrics", promhttp.Handler())

		} else {
			r = s.routes
		}

		/* Register routes */
		for _, service := range s.auths {
			if len(service.Patterns) > 0 {
				rids.Routes(service.Patterns, r, s.handler)
			}
		}

		/* WebSocket handler with Patterns */
		r.HandleFunc("/ws", socket.NewConnectionWS(s.Base, s.auths...))

		if s.Cancel != nil {
			s.Cancel()
			<-s.terminated
		}

		go func() {
			srv := http.Server{Addr: ":3333", Handler: r}
			ctx, cancel := context.WithCancel(context.Background())
			s.Cancel = cancel

			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Fatal(err)
				}
			}()
			started <- true
			<-ctx.Done()
			s.terminated <- true
			srv.Shutdown(ctx)
		}()
	}()
	return started
}
