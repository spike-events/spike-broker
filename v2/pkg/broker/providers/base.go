package providers

import (
	"fmt"

	"github.com/spike-events/spike-broker/v2/pkg/broker"
)

func NewBase() Base {
	return Base{
		handler: make(map[string]broker.ServiceHandler),
	}
}

type Base struct {
	handler map[string]broker.ServiceHandler
}

func (b *Base) GetHandler(service string) broker.ServiceHandler {
	if b.handler == nil {
		return nil
	}

	if handler, ok := b.handler[service]; ok {
		return handler
	}
	return nil
}

func (b *Base) SetHandler(service string, handler broker.ServiceHandler) {
	if _, exists := b.handler[service]; exists {
		panic(fmt.Sprintf("cannot change broker handler for the service %s", service))
	}
	b.handler[service] = handler
}
