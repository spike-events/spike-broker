package spike

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-kit/kit/endpoint"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/spike-events/spike-broker/pkg/models"
	"github.com/spike-events/spike-broker/pkg/service/request"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/spike/socket"
	"github.com/vincent-petithory/dataurl"
)

type serviceImpl struct {
	opts Options
}

func (s *serviceImpl) Start(options Options) error {
	s.opts = options
	options.Provider.SetHandler(func(p rids.Pattern, payload []byte) {
		call := broker.NewCall(p, payload)
		access := broker.NewAccess(call)
		handleRequest(p, call, access, options)
	})
	return nil
}

func (s *serviceImpl) Stop(timeout time.Duration) error {
	s.opts.Provider.Close()
	return nil
}

func (s *serviceImpl) listenHttp() chan bool {
	started := make(chan bool)
	go func() {
		var r *chi.Mux
		r = chi.NewRouter()

		// A good base middleware stack
		r.Use(middleware.RequestID)
		r.Use(middleware.RealIP)
		r.Use(middleware.Logger)
		r.Use(middleware.Recoverer)
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Content-Type", "application/json")
				next.ServeHTTP(w, r)
			})
		})

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

		/* Register routes */
		for _, sub := range s.opts.Service.Handlers() {
			method := sub.Resource.EndpointREST()
			httpHandler := func(w http.ResponseWriter, r *http.Request) {
				call, access := s.httpHandler(sub.Resource, w, r)
				handleRequest(sub.Resource, call, access, s.opts)
			}

			log.Printf("%s -> %s", sub.Resource.Method(), method)
			switch sub.Resource.Method() {
			case "GET":
				r.Get(method, httpHandler)
			case "POST":
				r.Post(method, httpHandler)
			case "PUT":
				r.Put(method, httpHandler)
			case "PATCH":
				r.Patch(method, httpHandler)
			case "DELETE":
				r.Delete(method, httpHandler)
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

func (s *serviceImpl) httpHandler(p rids.Pattern, w http.ResponseWriter, r *http.Request) (broker.Call, broker.Access) {

	defer r.Body.Close()
	r.ParseMultipartForm(0)

	debugLevel := os.Getenv("API_LOG_LEVEL")

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		if debugLevel == "DEBUG" || debugLevel == "ERR" {
			log.Printf("api: failed to read all body: %v", err)
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	params := make(map[string]string)
	for _, p := range endpoint.Params {
		value := r.FormValue(p)
		if value != "" {
			params[p] = value
		}
	}
	for _, p := range endpoint.Params {
		value := chi.URLParam(r, p)
		if value != "" {
			params[p] = value
		}
	}
	for key, value := range r.URL.Query() {
		if len(value) > 0 {
			params[key] = value[0]
		}
	}
	token := r.Header.Get("token")
	call := request.CallRequest{
		Data:     data,
		Params:   params,
		Form:     r.Form,
		PostForm: r.PostForm,
		Endpoint: endpoint.Endpoint,
		Header:   r.Header,
	}

	if len(token) > 0 {
		call.Token = token
	}

	if r.Method == http.MethodGet {
		query := r.Form.Encode()
		if len(query) > 0 {
			call.Query = query
		}
	}

	values := strings.Split(endpoint.Endpoint, ".")

	permission := models.HavePermissionRequest{
		Service:  values[0],
		Endpoint: strings.ReplaceAll(strings.Join(values, "."), "."+values[len(values)-1], ""),
		Method:   values[len(values)-1],
	}

	if endpoint.Authenticated && len(s.auths) > 0 {

		callAuth := request.NewRequest(permission)
		callAuth.Form = r.Form
		if token != "" {
			callAuth.Token = token
		}

		if !s.auths[0].Auth.UserHavePermission(callAuth) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	if debugLevel == "DEBUG" {
		log.Printf("api: requesting endpoint %s", endpoint.Endpoint)
	}
	result, rErr := s.Broker().RequestRaw(endpoint.Endpoint, call.ToJSON())
	if debugLevel == "DEBUG" {
		if rErr != nil {
			log.Printf("api: finished requesting endpoint with error %s", endpoint.Endpoint)
		} else {
			log.Printf("api: finished requesting endpoint %s without erros", endpoint.Endpoint)
		}
	}

	if rErr != nil {
		if debugLevel == "DEBUG" || debugLevel == "ERR" {
			log.Printf("api: failed to call endpoint: %v", rErr)
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("%v", rErr)))
		return
	}

	var parsedResponse request.ErrorRequest
	if parsedResponse.Parse(result); parsedResponse.Message != "" {
		if debugLevel == "DEBUG" || debugLevel == "ERR" {
			log.Printf("api: received error response: %s", parsedResponse.ToJSON())
		}
		w.WriteHeader(parsedResponse.Code)
		w.Write(parsedResponse.ToJSON())
		return
	}

	var redirect request.RedirectRequest
	json.Unmarshal(result, &redirect)

	if redirect.URL != "" {
		http.Redirect(w, r, redirect.URL, http.StatusMovedPermanently)
		return
	}

	var dataURL dataurl.DataURL
	json.Unmarshal(parsedResponse.Data, &dataURL)

	if dataURL.ContentType() != "" && len(dataURL.Data) > 0 {
		w.Header().Set("Content-Type", dataURL.ContentType())
		w.Header().Set("ETag", fmt.Sprintf("%x", sha256.Sum256(dataURL.Data)))
		if len(dataURL.Params) > 0 {
			if filename, ok := dataURL.Params["filename"]; ok {
				w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write(dataURL.Data)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(parsedResponse.Data)
}
