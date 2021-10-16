package main

import (
	apiproxynats "github.com/spike-events/spike-broker"
	"github.com/spike-events/spike-broker/pkg/models"
)

func main() {
	apiproxynats.NewProxyServer(nil, nil, nil, models.ProxyOptions{})
}