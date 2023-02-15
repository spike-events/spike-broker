package nats

import "C"
import (
	"container/ring"
	"encoding/json"
	"fmt"
	"log"
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

func NewNatsProvider(config Config) broker.Provider {
	natsConn := &Provider{
		config: config,
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

	return broker.NewSpecific(natsConn)
}

type Provider struct {
	config    Config
	debug     bool
	localNats *server.Server
}

func (s *Provider) SubscribeRaw(sub broker.Subscription, group string, handler broker.ServiceHandler) (func(), broker.Error) {
	m.Lock()
	defer m.Unlock()
	subj, msgs := s.subscribe(sub, handler)
	reg, _ := regexp.Compile("\\$[^.]+")
	subj = reg.ReplaceAllString(subj, "*")

	unsubs := make([]func(), 0)
	for i := 0; i < MaxConns; i++ {
		bus := s.requestConn()
		unsub, err := bus.ChanQueueSubscribe(subj, group, msgs)
		if err != nil {
			log.Printf("nats: failed to subscribe to %s: %v", subj, err)
		}
		unsubs = append(unsubs, func() { unsub.Unsubscribe() })
		s.releaseConn(bus)
	}

	return func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("nats: panic on unsubscribe endpoint %s: %v", subj, r)
			}
		}()
		for _, unsub := range unsubs {
			unsub()
		}
		close(msgs)
	}, nil
}

func (s *Provider) Close() {
	s.drain()
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

func (s *Provider) PublishRaw(subject string, data []byte) broker.Error {
	s.printDebug("nats: publishing endpoint %s", subject)
	bus := s.requestConn()
	defer s.releaseConn(bus)
	err := bus.Publish(subject, data)
	if err != nil {
		return broker.InternalError(err)
	}
	return nil
}

func (s *Provider) RequestRaw(subject string, data []byte, overrideTimeout ...time.Duration) ([]byte, broker.Error) {
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

func (s *Provider) NewCall(p rids.Pattern, payload interface{}) broker.Call {
	return broker.NewCall(p, payload)
}

func (s *Provider) connError(_ *nats.Conn, err error) {
	if err != nil {
		s.printDebug("nats: disconnected with error: %s", err)
	}
}

func (s *Provider) asyncError(_ *nats.Conn, sub *nats.Subscription, err error) {
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
	opts.Timeout = globalTimeout
	opts.DisconnectedErrCB = s.connError
	opts.AsyncErrorCB = s.asyncError
	opts.FlusherTimeout = 3 * time.Second
	bus, err := opts.Connect()
	if err != nil {
		return nil, err
	}
	return bus, nil
}

func (s *Provider) drain() {
	s.printDebug("nats: closing bus")
	defer s.printDebug("nats: closing bus done")
	globalConnections.Do(func(busI interface{}) {
		if bus, ok := busI.(*nats.Conn); ok {
			bus.Drain()
		}
	})
}

func (s *Provider) requestConn() *nats.Conn {
	next := globalConnections.Next()
	globalConnections = next
	return next.Value.(*nats.Conn)
}

func (s *Provider) releaseConn(_ *nats.Conn) {
}

func (s *Provider) printDebug(str string, params ...interface{}) {
	if s.debug {
		log.Printf(str, params...)
	}
}

func (s *Provider) subscribe(sub broker.Subscription, handler broker.ServiceHandler) (string, chan *nats.Msg) {
	msgs := make(chan *nats.Msg, MaxChans)

	go func() {
		h := handler
		p := sub.Resource
		for msgCopy := range msgs {
			msg := msgCopy
			go func() {
				if msg == nil {
					panic("nats: invalid message")
				}
				h(sub, msg.Data, msg.Reply)
			}()
		}
		s.printDebug("nats: channel closed on endpoint %s", p.EndpointName())
	}()

	s.printDebug("nats: subscribed on %s\n", sub.Resource.EndpointName())
	return sub.Resource.EndpointName(), msgs
}

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
				//if msg.Header.Get("Status") == "503" {
				//	return nil, broker.ErrorServiceUnavailable
				//}
				return nil, broker.ErrorServiceUnavailable
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

// getTimeout is the function that returns the timeout for a Broker request
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
