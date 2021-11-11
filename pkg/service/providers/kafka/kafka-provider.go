package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gofrs/uuid"
	"github.com/hetiansu5/urlquery"
	"github.com/spike-events/spike-broker/pkg/models"
	"github.com/spike-events/spike-broker/pkg/rids"
	"github.com/spike-events/spike-broker/pkg/service/request"
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

type KafkaConn struct {
	config *models.KafkaConfig
	ctx    context.Context
	admin  *kafka.AdminClient
}

func NewKafkaConn(config *models.KafkaConfig, ctx context.Context) *KafkaConn {
	a, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": config.KafkaURL})
	if err != nil {
		fmt.Printf("Failed to create Admin client: %s\n", err)
		os.Exit(1)
	}
	return &KafkaConn{config, ctx, a}
}

func (s *KafkaConn) Close() {
	s.admin.Close()
}

func (s *KafkaConn) subscribe(monitor bool, group string, p *rids.Pattern, hc func(msg *request.CallRequest), access ...func(msg *request.AccessRequest)) (interface{}, error) {

	topics, err := s.admin.CreateTopics(s.ctx, []kafka.TopicSpecification{{
		Topic:             p.EndpointName(),
		NumPartitions:     s.config.NumPartitions,
		ReplicationFactor: s.config.ReplicationFactor,
	}})
	if err != nil {
		return nil, err
	}
	for _, item := range topics {
		if item.Error.Code() != kafka.ErrNoError && item.Error.Code() != kafka.ErrTopicAlreadyExists {
			return nil, err
		}
	}

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": s.config.KafkaURL,
		"group.id":          group,
		"auto.offset.reset": "earliest",
		//"go.events.channel.enable": true,
	})
	if err != nil {
		return nil, err
	}
	err = c.SubscribeTopics([]string{p.EndpointName()}, nil)
	if err != nil {
		return nil, err
	}
	var wg sync.WaitGroup
	var messages []*kafka.Message
	cMessages := make(chan *kafka.Message)
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
					go func(m *kafka.Message) {
						defer wait.Done()
						s.newMessage(monitor, m, p, hc, access...)
					}(m)
				}
				wait.Wait()
			}
			messages = []*kafka.Message{}
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
			case ev := <-c.Events():
				switch e := ev.(type) {
				case kafka.AssignedPartitions:
					fmt.Fprintf(os.Stderr, "%% %v\n", e)
					c.Assign(e.Partitions)
				case kafka.RevokedPartitions:
					fmt.Fprintf(os.Stderr, "%% %v\n", e)
					c.Unassign()
				case *kafka.Message:
					fmt.Printf("%% Message on %s:\n%s\n", e.TopicPartition, string(e.Value))
					cMessages <- ev.(*kafka.Message)
				case kafka.PartitionEOF:
					fmt.Printf("%% Reached %v\n", e)
				case kafka.Error:
					// Errors should generally be considered as informational, the client will try to automatically recover
					fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
				}
			}
		}
	}()
	return c, nil
}

func (s *KafkaConn) printDebug(str string, params ...interface{}) {
	if s.config.Debug {
		log.Printf(str, params...)
	}
}

func (s *KafkaConn) newMessage(monitor bool, m *kafka.Message, p *rids.Pattern, hc func(msg *request.CallRequest), access ...func(msg *request.AccessRequest)) {
	msg := new(request.CallRequest)
	var reply string
	for _, item := range m.Headers {
		if item.Key == "reply" && !monitor {
			reply = string(item.Value)
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
		for _, accessPart := range access {
			acr := request.AccessRequest{
				CallRequest: msg,
			}

			accessPart(&acr)
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
	}

	s.printDebug("kafka: calling handler for endpoint %s inbox %s", p.EndpointName(), reply)
	hc(msg)
	s.printDebug("kafka: after calling handler for endpoint %s inbox %s", p.EndpointName(), reply)
}

func (s *KafkaConn) Subscribe(p *rids.Pattern, hc func(msg *request.CallRequest), access ...func(msg *request.AccessRequest)) (interface{}, error) {
	return s.subscribe(false, p.Service, p, hc, access...)
}

func (s KafkaConn) SubscribeAll(p *rids.Pattern, hc func(msg *request.CallRequest), access ...func(msg *request.AccessRequest)) (interface{}, error) {
	return s.subscribe(false, p.Service, p, hc, access...)
}

func (s KafkaConn) Monitor(monitoringGroup string, p *rids.Pattern, hc func(msg *request.CallRequest), access ...func(msg *request.AccessRequest)) (func(), error) {
	_, err := s.subscribe(true, monitoringGroup, p, hc, access...)
	if err != nil {
		return nil, err
	}
	return func() {}, nil
}

func (s KafkaConn) RegisterMonitor(p *rids.Pattern) error {
	// panic("implement me")
	return nil
}

func (s KafkaConn) Publish(p *rids.Pattern, payload *request.CallRequest, token ...string) error {
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
	if len(token) > 0 && token[0] != "" && len(token[0]) > 0 {
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
	return s.PublishRaw(specific, payload.ToJSON())
}

func (s KafkaConn) publish(subject string, data json.RawMessage, headers ...kafka.Header) error {
	if subject == "" {
		return nil
	}

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": s.config.KafkaURL,
	})
	if err != nil {
		return err
	}

	deliveryChan := make(chan kafka.Event)
	defer close(deliveryChan)
	err = p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &subject, Partition: kafka.PartitionAny},
		Value:          data,
		Headers:        headers,
	}, deliveryChan)
	if err != nil {
		return err
	}

	e := <-deliveryChan
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		fmt.Printf("Delivery failed: %v\n", m.TopicPartition.Error)
	} else {
		fmt.Printf("Delivered message to topic %s [%d] at offset %v\n",
			*m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
	}

	return err
}

func (s KafkaConn) PublishRaw(subject string, data json.RawMessage) error {
	return s.publish(subject, data)
}

func (s KafkaConn) Get(p *rids.Pattern, rs interface{}, token ...string) *request.ErrorRequest {
	return s.Request(p, nil, rs, token...)
}

func (s KafkaConn) Request(p *rids.Pattern, payload *request.CallRequest, rs interface{}, token ...string) *request.ErrorRequest {
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
		payload.Token = token[0]
	}

	// Check dependencies
	result, rErr := s.RequestRaw(p.EndpointName(), payload.ToJSON())
	if rErr != nil {
		return rErr
	} else {
		s.printDebug("kafka: request to endpoint %s successful", p.EndpointName())
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

func (s *KafkaConn) RequestRaw(subject string, data json.RawMessage, overrideTimeout ...time.Duration) (json.RawMessage, *request.ErrorRequest) {
	s.printDebug("kafka: requesting endpoint %s", subject)

	id, _ := uuid.NewV4()
	inbox := fmt.Sprintf("%v.inbox.%v", strings.Split(subject, ".")[0], id.String())

	topics, err := s.admin.CreateTopics(s.ctx, []kafka.TopicSpecification{{
		Topic:             inbox,
		NumPartitions:     s.config.NumPartitions,
		ReplicationFactor: s.config.ReplicationFactor,
	}})
	if err != nil {
		return nil, request.InternalError(err)
	}
	for _, item := range topics {
		if item.Error.Code() != kafka.ErrNoError {
			return nil, request.InternalError(err)
		}
	}

	defer func() {
		topics, err := s.admin.DeleteTopics(s.ctx, []string{inbox})
		if err != nil {
			fmt.Println(err)
		}
		for _, item := range topics {
			fmt.Println(item)
		}
	}()

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": s.config.KafkaURL,
		"group.id":          strings.Split(subject, ".")[0],
		"auto.offset.reset": "earliest",
		//"go.events.channel.enable": true,
	})

	err = c.SubscribeTopics([]string{inbox}, nil)
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
			s.printDebug("kafka: failed response on endpoint %s inbox %s", subject, inbox)
		} else {
			s.printDebug("kafka: received response on endpoint %s inbox %s", subject, inbox)
		}
	}()
	err = s.publish(subject, data, kafka.Header{Key: "reply", Value: []byte(inbox)})
	if err != nil {
		s.printDebug("kafka: failed to publish on endpoint %s inbox %s: %s", subject, inbox, err)
		return nil, request.InternalError(err)
	}
	wg.Wait()

	return rs, rsErr
}

func (s *KafkaConn) processResponse(subject, inbox string, c *kafka.Consumer, t time.Duration) (json.RawMessage, *request.ErrorRequest) {
	s.printDebug("kafka: waiting for response on endpoint %s inbox %s", subject, inbox)
	start := time.Now()
	defer func() {
		s.printDebug("kafka: finished processing response on endpoint %s inbox %s in %f ms",
			subject, inbox, time.Now().Sub(start).Milliseconds())
	}()
	for {
		timer := time.NewTimer(t)
		select {
		case <-s.ctx.Done():
			if s.config.Debug {
				log.Printf("kafka: timed out on endpoint %s inbox %s in %f",
					subject, inbox, time.Now().Sub(start).Seconds())
			}
			return nil, &request.ErrorInternalServerError
		case <-time.After(t):
			if s.config.Debug {
				log.Printf("kafka: timed out on endpoint %s inbox %s in %f",
					subject, inbox, time.Now().Sub(start).Seconds())
			}
			return nil, &request.ErrorTimeout
		case ev := <-c.Events():
			switch e := ev.(type) {
			case kafka.AssignedPartitions:
				fmt.Fprintf(os.Stderr, "%% %v\n", e)
				c.Assign(e.Partitions)
				//return nil, request.InternalError(fmt.Errorf("%% %v\n", e))
			case kafka.RevokedPartitions:
				fmt.Fprintf(os.Stderr, "%% %v\n", e)
				c.Unassign()
				//return nil, request.InternalError(fmt.Errorf("%% %v\n", e))
			case *kafka.Message:
				msg := ev.(*kafka.Message)
				s.printDebug("kafka message:", string(msg.Value))
				timer.Stop()
				respStr := string(msg.Value)
				timeout, resErr := s.getTimeout(respStr)
				if resErr != nil {
					s.printDebug("kafka: invalid timeout response on endpoint %s inbox %s: %s", subject, inbox, resErr.Message)
					return nil, resErr
				}
				if timeout != nil {
					t = *timeout
					s.printDebug("kafka: timeout extended in %f seconds on endpoint %s inbox %s", t.Seconds(), subject, inbox)
					break
				}
				return msg.Value, nil
			case kafka.PartitionEOF:
				fmt.Printf("%% Reached %v\n", e)
				//return nil, request.InternalError(fmt.Errorf("%% Reached %v\n", e))
			case kafka.Error:
				// Errors should generally be considered as informational, the client will try to automatically recover
				return nil, request.InternalError(fmt.Errorf("%% Error: %v\n", e))
			}
		}
	}
}

func (s *KafkaConn) getTimeout(respStr string) (*time.Duration, *request.ErrorRequest) {
	if strings.HasPrefix(respStr, "timeout:") {
		respParts := strings.Split(respStr, ":")
		if len(respParts) == 2 && respParts[0] == "timeout" {
			if len(respParts[1]) == 0 {
				err := &request.ErrorInvalidParams
				err.Message = fmt.Sprintf("Invalid timeout param %s", respParts[1])
				return nil, &request.ErrorInvalidParams
			}

			if len(respParts[1]) > 20 {
				log.Printf("kafka: timeout: too large timeout message: %s", respParts[1])
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
