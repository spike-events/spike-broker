package nats

import (
	"container/ring"
	"encoding/json"
	"fmt"
	"github.com/hetiansu5/urlquery"
	"github.com/spike-events/spike-broker/pkg/service/request"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/spike-events/spike-broker/pkg/models"
	"github.com/spike-events/spike-broker/pkg/rids"
)

const (
	NATSMaxChans = 100
	NATSMaxConns = 20
)

const (
	ConfigEnableDebug = Config(1)
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

type Config int
type NatsConn struct {
	natsURL string
	debug   bool
}

func (s *NatsConn) connError(con *nats.Conn, err error) {
	if err != nil {
		s.printDebug("nats: disconnected with error: %s", err)
	}
}

func (s *NatsConn) asyncError(con *nats.Conn, sub *nats.Subscription, err error) {
	if err != nil {
		log.Printf("nats: async error: %v", err)
	}
}

func (s *NatsConn) newNatsBus() (*nats.Conn, error) {
	opts := nats.GetDefaultOptions()
	opts.Url = s.natsURL
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

func NewNatsConn(natsURL string, configs ...Config) *NatsConn {
	natsConn := &NatsConn{
		natsURL: natsURL,
	}

	for _, config := range configs {
		switch config {
		case ConfigEnableDebug:
			natsConn.debug = true
		}
	}

	m.Lock()
	defer m.Unlock()

	if globalConnections == nil {
		for range make([]int, NATSMaxConns) {
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

func (s *NatsConn) Drain() {
	s.printDebug("nats: closing bus")
	defer s.printDebug("nats: closing bus done")
	globalConnections.Do(func(busI interface{}) {
		if bus, ok := busI.(*nats.Conn); ok {
			bus.Drain()
		}
	})
}

func (s *NatsConn) Close() {
	s.Drain()
	s.printDebug("nats: closing bus")
	defer s.printDebug("nats: closing bus done")
	globalConnections.Do(func(busI interface{}) {
		if bus, ok := busI.(*nats.Conn); ok {
			bus.Close()
		}
	})
}

func (s *NatsConn) requestConn() *nats.Conn {
	next := globalConnections.Next()
	globalConnections = next
	return next.Value.(*nats.Conn)
}

func (s *NatsConn) releaseConn(_ *nats.Conn) {
}

func (s *NatsConn) Timeout() time.Duration {
	return globalTimeout
}

func (s *NatsConn) printDebug(str string, params ...interface{}) {
	if s.debug {
		log.Printf(str, params...)
	}
}

func (s *NatsConn) subscribe(p *rids.Pattern,
	hc func(msg *request.CallRequest),
	access ...func(msg *request.AccessRequest)) (string, string, chan *nats.Msg) {

	handler := func(m *nats.Msg) {
		bus := s.requestConn()
		defer s.releaseConn(bus)
		defer func() {
			if r := recover(); r != nil {
				req := new(request.CallRequest)
				req.SetProvider(s)
				req.SetReply(m.Reply)
				resErr := &request.ErrorInternalServerError
				err, ok := r.(error)
				if !ok {
					resErr.Error = fmt.Errorf("request: %s: %v", p.EndpointName(), r)
				} else {
					resErr.Message = err.Error()
				}
				log.Printf("nats: panic on handler: %v", r)
				req.ErrorRequest(&request.ErrorInternalServerError)
			}
		}()

		// Parse message
		msg := new(request.CallRequest)
		err := msg.Unmarshal(m.Data, m.Reply, s)
		if err != nil {
			panic(err)
		}

		// validate token and permission
		if p.Authenticated {
			if len(msg.RawToken()) == 0 {
				msg.ErrorRequest(&request.ErrorStatusUnauthorized)
				return
			}

			rErr := s.Request(rids.Route().ValidateToken(), request.NewRequest(msg.RawToken()), nil)
			if rErr != nil {
				msg.ErrorRequest(&request.ErrorStatusUnauthorized)
				return
			}

			req := request.NewRequest(models.HavePermissionRequest{
				Service:  p.Service,
				Endpoint: p.EndpointNoMethod(),
				Method:   p.Method,
			})

			req.Token = string(msg.RawToken())
			req.Form = msg.Form
			req.Params = msg.Params

			if p.Method == http.MethodGet {
				req.Query = msg.Query
			}

			rErr = s.Request(rids.Route().UserHavePermission(), req, nil, msg.RawToken())
			if rErr != nil {
				msg.ErrorRequest(rErr)
				return
			}
		}

		// local access
		if len(access) > 0 {
			acr := request.AccessRequest{
				CallRequest: msg,
			}

			access[0](&acr)
			if acr.IsError() {
				msg.ErrorRequest(&request.ErrorStatusForbidden)
				return
			}

			if acr.RequestIsGet() != nil && *acr.RequestIsGet() && p.Method != "GET" {
				msg.ErrorRequest(&request.ErrorStatusForbidden)
				return
			}
			if acr.RequestIsGet() != nil && !*acr.RequestIsGet() {
				match := false
				for _, m := range acr.Methods() {
					match = match || strings.Contains(p.EndpointName(), m)
				}
				if !match {
					msg.ErrorRequest(&request.ErrorStatusForbidden)
					return
				}
			}
		}

		s.printDebug("nats: calling handler for endpoint %s inbox %s", p.EndpointName(), m.Reply)
		hc(msg)
		s.printDebug("nats: after calling handler for endpoint %s inbox %s", p.EndpointName(), m.Reply)
	}

	msgs := make(chan *nats.Msg, NATSMaxChans)
	//running := make(chan bool, 0)

	go func() {
		//running <- true
		for msg := range msgs {
			go handler(msg)
		}
		s.printDebug("nats: channel closed on endpoint %s", p.EndpointName())
	}()

	//<-running
	//close(running)

	s.printDebug("nats: subscribed on %s\n", p.EndpointName())
	return p.EndpointName(), p.EndpointName(), msgs
}

// Subscribe endpoint nats in balanced mode
func (s *NatsConn) Subscribe(p *rids.Pattern,
	hc func(msg *request.CallRequest),
	access ...func(msg *request.AccessRequest)) (interface{}, error) {
	subj, grp, msgs := s.subscribe(p, hc, access...)
	reg, _ := regexp.Compile("\\$[^.]+")
	subj = reg.ReplaceAllString(subj, "*")

	var sub *nats.Subscription
	for i := 0; i < NATSMaxConns; i++ {
		bus := s.requestConn()
		sub_, err := bus.ChanQueueSubscribe(subj, grp, msgs)
		if err != nil {
			log.Printf("nats: failed to subscribe to %s: %v", subj, err)
		}
		s.releaseConn(bus)
		sub = sub_
	}
	return sub, nil
}

// SubscribeAll endpoint nats in non balanced mode (receives all messages)
func (s *NatsConn) SubscribeAll(p *rids.Pattern,
	hc func(msg *request.CallRequest),
	access ...func(msg *request.AccessRequest)) (interface{}, error) {
	bus := s.requestConn()
	defer s.releaseConn(bus)
	subj, _, ch := s.subscribe(p, hc, access...)
	reg, _ := regexp.Compile("\\$[^.]+")
	subj = reg.ReplaceAllString(subj, "*")
	return bus.ChanSubscribe(subj, ch)
}

// Monitor listens on endpoint nats in balanced mode do not respond. Use it to listen to events without answering them
func (s *NatsConn) Monitor(monitoringGroup string,
	p *rids.Pattern,
	hc func(msg *request.CallRequest),
	access ...func(msg *request.AccessRequest)) (func(), error) {
	specific := p.EndpointName()
	for key, vl := range p.Params {
		specific = strings.ReplaceAll(specific, "$"+key, vl)
	}
	reg, _ := regexp.Compile("\\$[^.]+")
	specific = reg.ReplaceAllString(specific, "*")
	bus := s.requestConn()
	defer s.releaseConn(bus)
	_, _, msgs := s.subscribe(p, hc, access...)
	sub, err := bus.ChanQueueSubscribe(specific, monitoringGroup, msgs)
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
		err := sub.Unsubscribe()
		if err != nil {
			log.Printf("nats: failed to unsubscribe monitor %s: %v", specific, err)
		}
		close(msgs)
		s.printDebug("nats: successfully unsubscribed monitor endpoint %s", specific)
	}, nil
}

// RegisterMonitor must be called for all events that can be published
func (s *NatsConn) RegisterMonitor(p *rids.Pattern) error {
	_, err := s.Monitor(p.Service, p, func(r *request.CallRequest) { r.OK() })
	return err
}

// Publish endpoint nats
func (s *NatsConn) Publish(p *rids.Pattern, payload *request.CallRequest, token ...json.RawMessage) error {
	if payload == nil {
		payload = request.EmptyRequest()
	}
	if p.QueryParams != nil {
		urlEncoder := urlquery.NewEncoder(urlquery.WithNeedEmptyValue(true))
		query, err := urlEncoder.Marshal(p.QueryParams)
		if err != nil {
			return err
		}
		payload.Query = string(query)
	}
	if len(token) > 0 && token[0] != nil && len(token[0]) > 0 {
		payload.Token = string(token[0])
	}
	specific := p.EndpointName()
	for key, vl := range p.Params {
		specific = strings.ReplaceAll(specific, "$"+key, vl)
	}
	if payload.Params == nil {
		payload.Params = make(map[string]string)
	}
	for key := range p.Params {
		payload.Params[key] = p.Params[key]
	}
	bus := s.requestConn()
	defer s.releaseConn(bus)
	return bus.Publish(specific, payload.ToJSON())
}

// Get endpoint nats
func (s *NatsConn) Get(p *rids.Pattern, rs interface{}, token ...json.RawMessage) *request.ErrorRequest {
	return s.Request(p, nil, rs, token...)
}

// Request endpoint nats
func (s *NatsConn) Request(p *rids.Pattern, payload *request.CallRequest, rs interface{}, token ...json.RawMessage) *request.ErrorRequest {
	if payload == nil {
		payload = &request.CallRequest{}
	}
	if payload.Params == nil {
		payload.Params = make(map[string]string)
	}
	for key := range p.Params {
		payload.Params[key] = p.Params[key]
	}
	if p.QueryParams != nil {
		urlEncoder := urlquery.NewEncoder(urlquery.WithNeedEmptyValue(true))
		query, err := urlEncoder.Marshal(p.QueryParams)
		if err != nil {
			return &request.ErrorInvalidParams
		}
		payload.Query = string(query)
	}
	if len(token) > 0 && token[0] != nil && len(token[0]) > 0 {
		payload.Token = string(token[0])
	}

	// Check dependencies
	result, rErr := s.RequestRaw(p.EndpointName(), payload.ToJSON())
	if rErr != nil {
		return rErr
	} else {
		s.printDebug("nats: request to endpoint %s successful", p.EndpointName())
	}

	var eError request.ErrorRequest
	eError.Parse(result)
	if eError.Code != 200 && eError.Message != "" {
		return &eError
	}

	if rs != nil {
		err := json.Unmarshal(eError.Data, rs)
		if err != nil {
			return &request.ErrorRequest{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
				Error:   err,
			}
		}
	}
	return nil
}

func (s *NatsConn) PublishRaw(subject string, data json.RawMessage) error {
	s.printDebug("nats: publishing endpoint %s", subject)
	bus := s.requestConn()
	defer s.releaseConn(bus)
	return bus.Publish(subject, data)
}

func (s *NatsConn) RequestRaw(subject string, data json.RawMessage, overrideTimeout ...time.Duration) (json.RawMessage, *request.ErrorRequest) {
	s.printDebug("nats: requesting endpoint %s", subject)
	bus := s.requestConn()
	defer s.releaseConn(bus)

	c := make(chan *nats.Msg, 50)
	defer close(c)

	inbox := nats.NewInbox()
	sr, err := bus.ChanSubscribe(inbox, c)
	if err != nil {
		s.printDebug("nats: request to endpoint %s inbox %s failed with internal error %s", subject, inbox, err)
		return nil, request.InternalError(err)
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
		return nil, request.InternalError(err)
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
func (s *NatsConn) processResponse(subject, inbox string, c chan *nats.Msg, t time.Duration) (json.RawMessage, *request.ErrorRequest) {
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
				s.printDebug("nats: invalid timeout response on endpoint %s inbox %s: %s", subject, inbox, resErr.Message)
				return nil, resErr
			}

			if timeout != nil {
				t = *timeout
				s.printDebug("nats: timeout extended in %f seconds on endpoint %s inbox %s", t.Seconds(), subject, inbox)
				break
			}

			return msg.Data, nil

		case <-timer.C:
			if s.debug {
				log.Printf("nats: timed out on endpoint %s inbox %s in %f",
					subject, inbox, time.Now().Sub(start).Seconds())
			}
			return nil, &request.ErrorTimeout
		}
	}
}

/*
getTimeout is the function that returns the timeout for a Broker request
*/
func (s *NatsConn) getTimeout(respStr string) (*time.Duration, *request.ErrorRequest) {
	if strings.HasPrefix(respStr, "timeout:") {
		respParts := strings.Split(respStr, ":")
		if len(respParts) == 2 && respParts[0] == "timeout" {
			if len(respParts[1]) == 0 {
				err := &request.ErrorInvalidParams
				err.Message = fmt.Sprintf("Invalid timeout param %s", respParts[1])
				return nil, &request.ErrorInvalidParams
			}

			if len(respParts[1]) > 20 {
				log.Printf("nats: timeout: too large timeout message: %s", respParts[1])
			}

			timeout, err := strconv.ParseInt(respParts[1], 10, 64)
			if err != nil {
				return nil, request.InternalError(err)
			}
			t := time.Duration(timeout)
			return &t, nil
		}
	}

	return nil, nil
}
