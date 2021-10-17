package main

import (
	"github.com/spike-events/spike-broker"
	"github.com/spike-events/spike-broker/pkg/models"
)

func main() {
	spikebroker.NewProxyServer(nil, nil, nil, models.ProxyOptions{})
}
