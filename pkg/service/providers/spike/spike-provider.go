package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/hetiansu5/urlquery"
	"github.com/spike-events/spike-broker/pkg/models"
	"github.com/spike-events/spike-broker/pkg/rids"
	"github.com/spike-events/spike-broker/pkg/service/request"
	spike_io "github.com/spike-events/spike-events"
	"github.com/spike-events/spike-events/pkg/client"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var globalTimeout time.Duration

func init() {
	t := os.Getenv("TIMEOUT")
	tParse, err := strconv.Atoi(t)
	if err == nil {
		globalTimeout = time.Duration(tParse) * time.Millisecond
	} else {
		globalTimeout = 30 * time.Second
	}
}

type SpikeConn struct {
	config *models.SpikeConfig
	ctx    context.Context
	admin  client.Conn
}

func NewSpikeConn(config *models.SpikeConfig, ctx context.Context) *SpikeConn {
	a, err := spike_io.NewClient(config.SpikeURL)
	if err != nil {
		fmt.Printf("Failed to create Admin client: %s\n", err)
		os.Exit(1)
	}
	return &SpikeConn{config, ctx, a}
}

func (s *SpikeConn) Close() {
	//s.admin.Close()
}

func (s *SpikeConn) subscribe(monitor bool, group string, p *rids.Pattern, hc func(msg *request.CallRequest), access ...func(msg *request.AccessRequest)) (interface{}, error) {
	c, err := s.admin.Subscribe(s.ctx, client.Topic{
		Topic:      p.EndpointName(),
		GroupId:    group,
		Persistent: p.Persistent(),
	})
	if err != nil {
		return nil, err
	}
	var wg sync.WaitGroup
	var messages []*client.Message
	cMessages := make(chan *client.Message)
	go func() {
		defer close(cMessages)
		for msg := range cMessages {
			select {
			case <-s.ctx.Done():
				return
			default:
			}
			wg.Wait()
			wg.Add(1)
			messages = append(messages, msg)
			wg.Done()
		}
	}()
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(time.Millisecond * 50):
			}
			wg.Wait()
			wg.Add(1)
			if len(messages) == 0 {
				wg.Done()
				continue
			}
			for _, m := range chunkSlice(messages, 50) {
				var wait sync.WaitGroup
				for _, m := range m {
					wait.Add(1)
					go func(m *client.Message) {
						defer wait.Done()
						s.newMessage(monitor, m, p, hc, access...)
					}(m)
				}
				wait.Wait()
			}
			messages = []*client.Message{}
			wg.Done()
		}
	}()
	go func() {
		defer c.Close()
		for {
			select {
			case <-s.ctx.Done():
				cMessages <- nil
				return
			case msg := <-c.Event():
				fmt.Printf("%% Message on %s:\n%s\n", msg.Topic, string(msg.Value))
				cMessages <- msg
			}
		}
	}()
	return c, nil
}

func (s *SpikeConn) printDebug(str string, params ...interface{}) {
	if s.config.Debug {
		log.Printf(str, params...)
	}
}

func (s *SpikeConn) newMessage(monitor bool, m *client.Message, p *rids.Pattern, hc func(msg *request.CallRequest), access ...func(msg *request.AccessRequest)) {
	msg := new(request.CallRequest)
	var reply string
	for k, item := range m.Header {
		if k == "reply" && !monitor {
			reply = string(item)
		}
	}
	err := msg.Unmarshal(m.Value, reply, s)
	if err != nil {
		panic(err)
	}

	if p.Authenticated {
		if len(msg.Token) == 0 {
			msg.ErrorRequest(&request.ErrorStatusUnauthorized)
			return
		}

		type rs struct {
			Token string
		}
		var result rs
		rErr := s.Request(rids.Route().ValidateToken(), nil, &result, msg.Token)
		if rErr != nil {
			msg.ErrorRequest(&request.ErrorStatusUnauthorized)
			return
		}
		msg.Token = result.Token

		req := request.NewRequest(models.HavePermissionRequest{
			Service:  p.Service,
			Endpoint: p.EndpointNoMethod(),
			Method:   p.Method,
		})

		req.Token = msg.Token
		req.Form = msg.Form
		req.Params = msg.Params

		if p.Method == http.MethodGet {
			req.Query = msg.Query
		}

		rErr = s.Request(rids.Route().UserHavePermission(), req, nil, msg.Token)
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

	s.printDebug("spike: calling handler for endpoint %s inbox %s", p.EndpointName(), reply)
	hc(msg)
	s.printDebug("spike: after calling handler for endpoint %s inbox %s", p.EndpointName(), reply)
}

func (s *SpikeConn) Subscribe(p *rids.Pattern, hc func(msg *request.CallRequest), access ...func(msg *request.AccessRequest)) (interface{}, error) {
	return s.subscribe(false, p.Service, p, hc, access...)
}

func (s SpikeConn) SubscribeAll(p *rids.Pattern, hc func(msg *request.CallRequest), access ...func(msg *request.AccessRequest)) (interface{}, error) {
	return s.subscribe(false, p.Service, p, hc, access...)
}

func (s SpikeConn) Monitor(monitoringGroup string, p *rids.Pattern, hc func(msg *request.CallRequest), access ...func(msg *request.AccessRequest)) (func(), error) {
	_, err := s.subscribe(true, monitoringGroup, p, hc, access...)
	if err != nil {
		return nil, err
	}
	return func() {}, nil
}

func (s SpikeConn) RegisterMonitor(p *rids.Pattern) error {
	// panic("implement me")
	return nil
}

func (s SpikeConn) Publish(p *rids.Pattern, payload *request.CallRequest, token ...string) error {
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
	if len(token) > 0 && len(token[0]) > 0 {
		payload.Token = token[0]
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
	return s.PublishRaw(specific, payload.ToJSON())
}

func (s SpikeConn) publish(subject string, data json.RawMessage, headers map[string][]byte) error {
	if subject == "" {
		return nil
	}

	msg := client.Message{
		Topic: subject,
		Value: data,
	}
	if headers != nil {
		msg.Header = headers
	}
	_, err := s.admin.Publish(msg)

	return err
}

func (s SpikeConn) PublishRaw(subject string, data json.RawMessage) error {
	return s.publish(subject, data, nil)
}

func (s SpikeConn) Get(p *rids.Pattern, rs interface{}, token ...string) *request.ErrorRequest {
	return s.Request(p, nil, rs, token...)
}

func (s SpikeConn) Request(p *rids.Pattern, payload *request.CallRequest, rs interface{}, token ...string) *request.ErrorRequest {
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
	if len(token) > 0 && token[0] != "" && len(token[0]) > 0 {
		payload.Token = string(token[0])
	}

	// Check dependencies
	result, rErr := s.RequestRaw(p.EndpointName(), payload.ToJSON())
	if rErr != nil {
		return rErr
	} else {
		s.printDebug("spike: request to endpoint %s successful", p.EndpointName())
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

func (s *SpikeConn) RequestRaw(subject string, data json.RawMessage, overrideTimeout ...time.Duration) (json.RawMessage, *request.ErrorRequest) {
	s.printDebug("spike: requesting endpoint %s", subject)

	id, _ := uuid.NewV4()
	inbox := fmt.Sprintf("%v.inbox.%v", strings.Split(subject, ".")[0], id.String())

	c, err := s.admin.Subscribe(s.ctx, client.Topic{
		Topic:   inbox,
		GroupId: strings.Split(subject, ".")[0],
	})
	if err != nil {
		return nil, request.InternalError(err)
	}
	defer c.Close()

	var rs json.RawMessage
	var rsErr *request.ErrorRequest
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		var t time.Duration
		if len(overrideTimeout) > 0 {
			t = overrideTimeout[0]
		} else {
			t = globalTimeout
		}
		rs, rsErr = s.processResponse(subject, inbox, c, t)
		if rsErr != nil {
			s.printDebug("spike: failed response on endpoint %s inbox %s", subject, inbox)
		} else {
			s.printDebug("spike: received response on endpoint %s inbox %s", subject, inbox)
		}
	}()
	err = s.publish(subject, data, map[string][]byte{"reply": []byte(inbox)})
	if err != nil {
		s.printDebug("spike: failed to publish on endpoint %s inbox %s: %s", subject, inbox, err)
		return nil, request.InternalError(err)
	}
	wg.Wait()

	return rs, rsErr
}

func (s *SpikeConn) processResponse(subject, inbox string, c *client.Subscriber, t time.Duration) (json.RawMessage, *request.ErrorRequest) {
	s.printDebug("spike: waiting for response on endpoint %s inbox %s", subject, inbox)
	start := time.Now()
	defer func() {
		s.printDebug("spike: finished processing response on endpoint %s inbox %s in %f ms",
			subject, inbox, time.Now().Sub(start).Milliseconds())
	}()
	for {
		timer := time.NewTimer(t)
		select {
		case <-s.ctx.Done():
			if s.config.Debug {
				log.Printf("spike: timed out on endpoint %s inbox %s in %f",
					subject, inbox, time.Now().Sub(start).Seconds())
			}
			return nil, &request.ErrorInternalServerError
		case <-time.After(t):
			if s.config.Debug {
				log.Printf("spike: timed out on endpoint %s inbox %s in %f",
					subject, inbox, time.Now().Sub(start).Seconds())
			}
			return nil, &request.ErrorTimeout
		case msg := <-c.Event():
			s.printDebug("spike message inbox:", string(msg.Value))
			timer.Stop()
			respStr := string(msg.Value)
			timeout, resErr := s.getTimeout(respStr)
			if resErr != nil {
				s.printDebug("spike: invalid timeout response on endpoint %s inbox %s: %s", subject, inbox, resErr.Message)
				return nil, resErr
			}
			if timeout != nil {
				t = *timeout
				s.printDebug("spike: timeout extended in %f seconds on endpoint %s inbox %s", t.Seconds(), subject, inbox)
				break
			}
			return msg.Value, nil
		}
	}
}

func (s *SpikeConn) getTimeout(respStr string) (*time.Duration, *request.ErrorRequest) {
	if strings.HasPrefix(respStr, "timeout:") {
		respParts := strings.Split(respStr, ":")
		if len(respParts) == 2 && respParts[0] == "timeout" {
			if len(respParts[1]) == 0 {
				err := &request.ErrorInvalidParams
				err.Message = fmt.Sprintf("Invalid timeout param %s", respParts[1])
				return nil, &request.ErrorInvalidParams
			}

			if len(respParts[1]) > 20 {
				log.Printf("spike: timeout: too large timeout message: %s", respParts[1])
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
