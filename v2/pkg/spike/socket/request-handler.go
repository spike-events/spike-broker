package socket

import (
	"fmt"

	"github.com/spike-events/spike-broker/v2/pkg/broker"
)

func handleWSEvent(ws WSConnection, sub broker.Subscription, payload []byte, replyEndpoint string) {
	msg, err := broker.NewCallFromJSON(payload, sub.Resource, replyEndpoint)
	if err == nil {
		msg.SetProvider(ws.Broker())
		access := broker.NewAccess(msg)
		p := sub.Resource

		defer func() {
			if r := recover(); r != nil {
				var ok bool
				var rErr broker.Error
				err, ok = r.(error)
				if !ok {
					rErr = broker.InternalError(fmt.Errorf("request: %s: %v", p.EndpointName(), r))
				} else {
					rErr = broker.InternalError(err)
				}
				ws.Logger().Printf("http: panic on handler: %v", r)
				msg.Error(rErr)
			}
		}()

		if len(sub.Validators) > 0 {
			for _, validator := range sub.Validators {
				validator(access)
				if err = access.GetError(); err != nil {
					msg.Error(err)
					return
				}
			}
		}

		sub.Handler(msg)
	}
}
