package nats

import "C"
import (
	"container/ring"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

const (
	MaxChans = 100
	MaxConns = 20
)

var globalTimeout time.Duration
var globalConnections *ring.Ring
var m sync.Mutex

func init() {
	t := os.Getenv("TIMEOUT")
	tParse, err := strconv.Atoi(t)
	if err == nil {
		globalTimeout = time.Duration(tParse) * time.Millisecond
	} else {
		globalTimeout = 30 * time.Second
	}
}

type Provider struct {
	config    Config
	handler   map[string]broker.ServiceHandler
	debug     bool
	localNats *server.Server
}

func (s *Provider) connError(con *nats.Conn, err error) {
	if err != nil {
		s.printDebug("nats: disconnected with error: %s", err)
	}
}

func (s *Provider) asyncError(con *nats.Conn, sub *nats.Subscription, err error) {
	if err != nil {
		log.Printf("nats: async error: %v", err)
	}
}

func (s *Provider) newNatsBus() (*nats.Conn, error) {
	opts := nats.GetDefaultOptions()
	opts.Url = s.config.NatsURL
	opts.AllowReconnect = true
	opts.MaxReconnect = -1
	opts.PingInterval = 5 * time.Second
	opts.MaxPingsOut = 3
	opts.Timeout = s.Timeout()
	opts.DisconnectedErrCB = s.connError
	opts.AsyncErrorCB = s.asyncError
	opts.FlusherTimeout = 3 * time.Second
	bus, err := opts.Connect()
	if err != nil {
		return nil, err
	}
	return bus, nil
}

func NewNatsProvider(config Config) broker.Provider {
	natsConn := &Provider{
		config:  config,
		handler: make(map[string]broker.ServiceHandler),
	}

	m.Lock()
	defer m.Unlock()

	if config.LocalNats {
		opts := &defaultNatsOptions
		opts.Debug = config.LocalNatsDebug
		opts.Trace = config.LocalNatsTrace
		natsConn.localNats = runServer(opts)
	}

	if globalConnections == nil {
		for range make([]int, MaxConns) {
			bus, err := natsConn.newNatsBus()
			if err != nil {
				panic(err)
			}
			if globalConnections == nil {
				globalConnections = &ring.Ring{Value: bus}
			} else {
				globalConnections = globalConnections.Link(&ring.Ring{Value: bus}).Next()
			}

		}
	}

	return natsConn
}

func (s *Provider) SetHandler(service string, handler broker.ServiceHandler) {
	s.handler[service] = handler
}

func (s *Provider) Drain() {
	s.printDebug("nats: closing bus")
	defer s.printDebug("nats: closing bus done")
	globalConnections.Do(func(busI interface{}) {
		if bus, ok := busI.(*nats.Conn); ok {
			bus.Drain()
		}
	})
}

func (s *Provider) Close() {
	s.Drain()
	s.printDebug("nats: closing bus")
	defer s.printDebug("nats: closing bus done")
	globalConnections.Do(func(busI interface{}) {
		if bus, ok := busI.(*nats.Conn); ok {
			bus.Close()
		}
	})
	globalConnections.Unlink(globalConnections.Len() - 1)
	globalConnections = nil
	if s.localNats != nil {
		s.localNats.Shutdown()
	}
}

func (s *Provider) requestConn() *nats.Conn {
	next := globalConnections.Next()
	globalConnections = next
	return next.Value.(*nats.Conn)
}

func (s *Provider) releaseConn(_ *nats.Conn) {
}

func (s *Provider) Timeout() time.Duration {
	return globalTimeout
}

func (s *Provider) printDebug(str string, params ...interface{}) {
	if s.debug {
		log.Printf(str, params...)
	}
}

func (s *Provider) subscribe(pattern rids.Pattern) (string, string, chan *nats.Msg) {
	msgs := make(chan *nats.Msg, MaxChans)

	go func() {
		p := pattern
		for msg := range msgs {
			go s.handler[pattern.Service()](p, msg.Data, msg.Reply)
		}
		s.printDebug("nats: channel closed on endpoint %s", p.EndpointName())
	}()

	s.printDebug("nats: subscribed on %s\n", pattern.EndpointName())
	return pattern.EndpointName(), pattern.EndpointName(), msgs
}

// Subscribe endpoint nats in balanced mode
func (s *Provider) Subscribe(sub broker.Subscription) (interface{}, error) {
	m.Lock()
	defer m.Unlock()
	subj, grp, msgs := s.subscribe(sub.Resource)
	reg, _ := regexp.Compile("\\$[^.]+")
	subj = reg.ReplaceAllString(subj, "*")

	for i := 0; i < MaxConns; i++ {
		bus := s.requestConn()
		_, err := bus.ChanQueueSubscribe(subj, grp, msgs)
		if err != nil {
			log.Printf("nats: failed to subscribe to %s: %v", subj, err)
		}
		s.releaseConn(bus)
	}
	return s, nil
}

// SubscribeAll endpoint nats in non balanced mode (receives all messages)
func (s *Provider) SubscribeAll(sub broker.Subscription) (interface{}, error) {
	m.Lock()
	defer m.Unlock()
	bus := s.requestConn()
	defer s.releaseConn(bus)
	subj, _, ch := s.subscribe(sub.Resource)
	reg, _ := regexp.Compile("\\$[^.]+")
	subj = reg.ReplaceAllString(subj, "*")
	return bus.ChanSubscribe(subj, ch)
}

// Monitor listens on endpoint nats in balanced mode do not respond. Use it to listen to events without answering them
func (s *Provider) Monitor(monitoringGroup string, sub broker.Subscription) (func(), error) {
	m.Lock()
	m.Unlock()
	specific := sub.Resource.EndpointNameSpecific()
	reg, _ := regexp.Compile("\\$[^.]+")
	specific = reg.ReplaceAllString(specific, "*")
	bus := s.requestConn()
	defer s.releaseConn(bus)
	_, _, msgs := s.subscribe(sub.Resource)
	natsSub, err := bus.ChanQueueSubscribe(specific, monitoringGroup, msgs)
	if err != nil {
		return nil, err
	}
	return func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("nats: panic on unsubscribe endpoint %s: %v", specific, r)
			}
		}()
		s.printDebug("nats: unsubscribing monitor endpoint %s", specific)
		err := natsSub.Unsubscribe()
		if err != nil {
			log.Printf("nats: failed to unsubscribe monitor %s: %v", specific, err)
		}
		close(msgs)
		s.printDebug("nats: successfully unsubscribed monitor endpoint %s", specific)
	}, nil
}

// Publish endpoint nats
func (s *Provider) Publish(p rids.Pattern, payload interface{}, token ...string) error {
	call := broker.NewCall(p, payload)
	if len(token) > 0 && len(token[0]) > 0 {
		call.SetToken(token[0])
	}
	bus := s.requestConn()
	defer s.releaseConn(bus)
	return bus.Publish(p.EndpointNameSpecific(), call.ToJSON())
}

// Get endpoint nats
func (s *Provider) Get(p rids.Pattern, rs interface{}, token ...string) broker.Error {
	return s.Request(p, nil, rs, token...)
}

// Request endpoint nats FIXME: This is a general handler
func (s *Provider) Request(p rids.Pattern, payload interface{}, rs interface{}, token ...string) broker.Error {
	call := broker.NewCall(p, payload)
	if len(token) > 0 && token[0] != "" && len(token[0]) > 0 {
		call.SetToken(token[0])
	}

	// Check dependencies
	result, rErr := s.RequestRaw(p.EndpointName(), call.ToJSON())
	if rErr != nil {
		return rErr
	} else {
		s.printDebug("nats: request to endpoint %s successful", p.EndpointName())
	}

	bMsg := broker.NewMessageFromJSON(result)
	if bMsg == nil {
		return broker.NewError("invalid payload", http.StatusInternalServerError, result)
	}

	if math.Abs(float64(bMsg.Code()-http.StatusOK)) >= 100 {
		return bMsg
	}

	if rs != nil {
		data, err := json.Marshal(bMsg.Data())
		if err != nil {
			return broker.InternalError(err)
		}
		if err = json.Unmarshal(data, rs); err != nil {
			return broker.InternalError(err)
		}
	}
	return nil
}

func (s *Provider) PublishRaw(subject string, data json.RawMessage) error {
	s.printDebug("nats: publishing endpoint %s", subject)
	bus := s.requestConn()
	defer s.releaseConn(bus)
	return bus.Publish(subject, data)
}

func (s *Provider) RequestRaw(subject string, data json.RawMessage, overrideTimeout ...time.Duration) (json.RawMessage, broker.Error) {
	s.printDebug("nats: requesting endpoint %s", subject)
	bus := s.requestConn()
	defer s.releaseConn(bus)

	c := make(chan *nats.Msg, 50)
	defer close(c)

	inbox := nats.NewInbox()
	sr, err := bus.ChanSubscribe(inbox, c)
	if err != nil {
		s.printDebug("nats: request to endpoint %s inbox %s failed with internal error %s", subject, inbox, err)
		return nil, broker.InternalError(err)
	}

	defer func() {
		subErr := sr.Unsubscribe()
		if subErr != nil {
			log.Printf("nats: failed to unsubscribe endpoint %s inbo %s: %s", subject, inbox, subErr)
		}
	}()

	err = bus.PublishMsg(&nats.Msg{
		Data:    data,
		Reply:   inbox,
		Subject: subject,
	})
	if err != nil {
		s.printDebug("nats: failed to publish on endpoint %s inbox %s: %s", subject, inbox, err)
		return nil, broker.InternalError(err)
	}

	var t time.Duration
	if len(overrideTimeout) > 0 {
		t = overrideTimeout[0]
	} else {
		t = globalTimeout
	}
	rs, rsErr := s.processResponse(subject, inbox, c, t)
	if rsErr != nil {
		s.printDebug("nats: failed response on endpoint %s inbox %s", subject, inbox)
		return nil, rsErr
	} else {
		s.printDebug("nats: received response on endpoint %s inbox %s", subject, inbox)
	}

	return rs, nil
}

/*
request is the function that sends a message during a Resgate request
*/
func (s *Provider) processResponse(subject, inbox string, c chan *nats.Msg, t time.Duration) (json.RawMessage, broker.Error) {
	s.printDebug("nats: waiting for response on endpoint %s inbox %s", subject, inbox)
	start := time.Now()
	defer func() {
		s.printDebug("nats: finished processing response on endpoint %s inbox %s in %f ms",
			subject, inbox, time.Now().Sub(start).Milliseconds())
	}()
	for {
		timer := time.NewTimer(t)
		select {
		case msg := <-c:
			timer.Stop()
			respStr := string(msg.Data)
			timeout, resErr := s.getTimeout(respStr)
			if resErr != nil {
				s.printDebug("nats: invalid timeout response on endpoint %s inbox %s: %s", subject, inbox, resErr.Error())
				return nil, resErr
			}

			if timeout != nil {
				t = *timeout
				s.printDebug("nats: timeout extended in %f seconds on endpoint %s inbox %s", t.Seconds(), subject, inbox)
				break
			}

			if respStr == "" {
				// TODO: Map all possible problematic headers
				if msg.Header.Get("Status") == "503" {
					return nil, broker.ErrorServiceUnavailable
				}
				return nil, broker.ErrorNotFound
			}
			return msg.Data, nil

		case <-timer.C:
			if s.debug {
				log.Printf("nats: timed out on endpoint %s inbox %s in %f",
					subject, inbox, time.Now().Sub(start).Seconds())
			}
			return nil, broker.ErrorTimeout
		}
	}
}

/*
getTimeout is the function that returns the timeout for a Broker request
*/
func (s *Provider) getTimeout(respStr string) (*time.Duration, broker.Error) {
	if strings.HasPrefix(respStr, "timeout:") {
		respParts := strings.Split(respStr, ":")
		if len(respParts) == 2 && respParts[0] == "timeout" {
			if len(respParts[1]) == 0 {
				err := broker.NewInvalidParamsError(fmt.Sprintf("Invalid timeout param %s", respParts[1]))
				return nil, err
			}

			if len(respParts[1]) > 20 {
				log.Printf("nats: timeout: too large timeout message: %s", respParts[1])
			}

			timeout, err := strconv.ParseInt(respParts[1], 10, 64)
			if err != nil {
				return nil, broker.InternalError(err)
			}
			t := time.Duration(timeout)
			return &t, nil
		}
	}

	return nil, nil
}

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
