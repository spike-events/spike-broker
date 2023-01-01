package spike

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	"github.com/spike-events/spike-broker/v2/pkg/service"
	spikeutils "github.com/spike-events/spike-broker/v2/pkg/spike-utils"
	"github.com/spike-events/spike-broker/v2/pkg/spike/socket"
	"github.com/vincent-petithory/dataurl"
)

type HttpOptions struct {
	// Broker implements the broker.Provider interface to allow HTTP Server to reach Spike network
	Broker broker.Provider

	// Resources handled by the HTTP Server
	Resources []rids.Resource

	// Authenticator implements the service.Authenticator interface to validate and process token
	Authenticator service.Authenticator

	// Authorizer implements the service.Authorizer interface to check if user has priviledge to access the endpoint
	Authorizer service.Authorizer

	// WSPrefix is the prefix path for http upgrade to web socket connection
	WSPrefix string

	// Address optionally specifies the TCP address for the server to listen on, in the form "host:port".
	// If empty, ":http" (port 80) is used. The service names are defined in RFC 6335 and assigned by IANA.
	// See net.Dial for details of the address format.
	Address string

	// Logger implements the logger interface for registering logs
	Logger service.Logger
}

// HttpServer implements the server that handles REST and WebSocket requests
type HttpServer interface {
	// ListenAndServe starts the HTTP server listens until Shutdown is called or some error occurs
	ListenAndServe() error

	// Shutdown requests the HTTP server to stop listening and return
	Shutdown() error
}

// NewHttpServer returns an HTTP Server instance using chi.Mux router
func NewHttpServer(ctx context.Context, opts HttpOptions) HttpServer {
	if opts.Broker == nil {
		panic("invalid empty options Broker")
	}

	if len(opts.Resources) == 0 {
		panic("invalid empty options Resources")
	}

	router := chi.NewRouter()
	return &httpServer{
		ctx:    ctx,
		server: &http.Server{Addr: opts.Address, Handler: router},
		router: router,
		opts:   opts,
	}
}

type httpServer struct {
	ctx    context.Context
	server *http.Server
	router *chi.Mux
	opts   HttpOptions
}

func (h *httpServer) ListenAndServe() error {
	c := make(chan bool)
	go func() {
		// A good base middleware stack
		h.router.Use(middleware.RequestID)
		h.router.Use(middleware.RealIP)
		h.router.Use(middleware.Logger)
		h.router.Use(middleware.Recoverer)
		h.router.Use(func(next http.Handler) http.Handler {
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
		h.router.Use(corsOpts.Handler)

		h.router.Use(middleware.Timeout(60 * time.Second)) // FIXME: HTTP timeout should be passed as parameter
		h.router.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(http.StatusOK)
		})

		h.router.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(http.StatusOK)
		})

		// Register pprof handlers
		h.router.HandleFunc("/debug/pprof/", pprof.Index)
		h.router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		h.router.HandleFunc("/debug/pprof/profile", pprof.Profile)
		h.router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

		h.router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		h.router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		h.router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		h.router.Handle("/debug/pprof/block", pprof.Handler("block"))
		h.router.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
		h.router.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
		h.router.Handle("/metrics", promhttp.Handler())

		h.httpSetup(h.opts.WSPrefix)

		go h.server.ListenAndServe()
		time.Sleep(2 * time.Second)
		c <- true
	}()
	<-c
	return nil
}

func (h *httpServer) Shutdown() error {
	if h.server != nil {
		return h.server.Shutdown(context.Background())
	}
	return fmt.Errorf("no server is listening")
}

func (h *httpServer) httpSetup(wsPrefix string) {
	// Register routes
	handlers := make([]rids.Pattern, 0)
	for _, resource := range h.opts.Resources {
		resource.Name()
		h.opts.Broker.SetHandler(resource.Name(), func(sub broker.Subscription, payload []byte, replyEndpoint string) {
			call, err := broker.NewCallFromJSON(payload, sub.Resource, replyEndpoint)
			if err == nil {
				call.SetProvider(h.opts.Broker)
				access := broker.NewAccess(call)
				handleWSRequest(sub, call, access, h.opts)
			}
		})
		rHandlers := rids.Patterns(resource)
		handlers = append(handlers, rHandlers...)
		for _, ptr := range rHandlers {
			p := ptr
			endpoint := p.EndpointREST()
			httpHandler := func(w http.ResponseWriter, r *http.Request) {
				h.httpHandler(p, w, r)
			}

			for param := range p.Params() {
				endpoint = strings.ReplaceAll(endpoint, fmt.Sprintf("$%s", param), fmt.Sprintf("{%s}", param))
			}
			h.opts.Logger.Printf("%s -> %s", p.Method(), endpoint)
			switch p.Method() {
			case http.MethodGet:
				h.router.Get(endpoint, httpHandler)
			case http.MethodPost:
				h.router.Post(endpoint, httpHandler)
			case http.MethodPut:
				h.router.Put(endpoint, httpHandler)
			case http.MethodPatch:
				h.router.Patch(endpoint, httpHandler)
			case http.MethodDelete:
				h.router.Delete(endpoint, httpHandler)
			}

		}
	}
	// TODO: Implement WebSocket handler
	wsOpts := socket.Options{
		Handlers:      handlers,
		Broker:        h.opts.Broker,
		Authenticator: h.opts.Authenticator,
		Authorizer:    h.opts.Authorizer,
		Logger:        h.opts.Logger,
	}
	wsPrefix = strings.Replace(wsPrefix, "/", "", 1)
	h.router.HandleFunc(fmt.Sprintf("/%s", wsPrefix), socket.NewConnectionWS(wsOpts))
}

func (h *httpServer) httpHandler(p rids.Pattern, w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	data, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	var token json.RawMessage
	tokenStr, _ := spikeutils.GetBearer(r)
	if len(tokenStr) > 0 {
		token = json.RawMessage(tokenStr)
	}
	params := make(map[string]fmt.Stringer)
	for param := range p.Params() {
		value := chi.URLParam(r, param)
		if value != "" {
			params[param] = spikeutils.Stringer(value)
		}
	}

	p = p.Clone()
	call := broker.NewHTTPCall(p, token, data, params, r.Form.Encode())
	result, rErr := h.opts.Broker.RequestRaw(p.EndpointName(), call.ToJSON())
	if rErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("%v", rErr)))
		return
	}

	parsedResponse := broker.NewMessageFromJSON(result)
	if parsedResponse != nil && parsedResponse.Error() != "" {
		w.WriteHeader(parsedResponse.Code())
		w.Write(parsedResponse.ToJSON())
		return
	}

	data = parsedResponse.Data()
	var dataURL dataurl.DataURL
	err = json.Unmarshal(data, &dataURL)
	if err == nil && dataURL.ContentType() != "" && len(dataURL.Data) > 0 {
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
	w.Write(data)
}
