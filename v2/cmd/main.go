package main

import (
	spikebroker "github.com/spike-events/spike-broker/v2"
	"github.com/spike-events/spike-broker/v2/pkg/models"
)

func main() {
	spikebroker.NewProxyServer(nil, nil, nil, models.ProxyOptions{})
}
