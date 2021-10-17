package test

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/spike-events/spike-broker"
	"github.com/spike-events/spike-broker/pkg/models"
	"github.com/spike-events/spike-broker/pkg/providers"
	"github.com/spike-events/spike-broker/pkg/rids"
	"github.com/spike-events/spike-broker/pkg/service"
	"github.com/spike-events/spike-broker/pkg/service/request"
	spike_io "github.com/spike-events/spike-events"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"testing"
)

// OtherConfig Service
type spikeRid struct {
	rids.Base
}

var spikeImpl spikeRid

func SpikeRid() *spikeRid {
	if len(spikeImpl.Base.Name()) == 0 {
		spikeImpl.Base = rids.NewRid("spike", "")
	}
	return &spikeImpl
}

func (k *spikeRid) NewRequest() *rids.Pattern {
	return k.NewMethod("", "request").NoAuth().Post()
}

func (k *spikeRid) Event() *rids.Pattern {
	return k.NewMethod("", "event").Internal()
}

func (k *spikeRid) EventQueue() *rids.Pattern {
	return k.NewMethod("", "queue").Internal()
}

type spikeService struct {
	*service.Base
}

func NewKafkaService(db *gorm.DB, key uuid.UUID) service.Service {
	base := service.NewBaseService(db, key, SpikeRid())
	srv := &spikeService{Base: base}
	return srv
}

func (s *spikeService) Start() {
	s.Init(nil, func() {

		s.Broker().Subscribe(SpikeRid().NewRequest(), func(msg *request.CallRequest) {
			fmt.Println(">> kafka new message: subscribe 111111111111111")
			msg.OK(map[string]string{"OK": "true"})
		})

		s.Broker().Monitor("monitor1", SpikeRid().NewRequest(), func(msg *request.CallRequest) {
			fmt.Println(">> kafka new message: subscribe 22222222222222")
			msg.OK(map[string]string{"OK": "true"})
		})

		s.Broker().Monitor("monitor2", SpikeRid().NewRequest(), func(msg *request.CallRequest) {
			fmt.Println(">> kafka new message: subscribe 3333333333333333")
			msg.OK(map[string]string{"OK": "true"})
		})

		//s.Broker().Subscribe(SpikeRid().NewRequest(), func(msg *request.CallRequest) {
		//	fmt.Println(">> kafka new message: subscribe 2222222222222222")
		//
		//	msg.OK(map[string]string{"OK": "true"})
		//})

		s.Broker().Subscribe(SpikeRid().EventQueue(), func(msg *request.CallRequest) {

			fmt.Println(string(msg.Data))
			msg.OK(map[string]string{"OK": "true"})

		})

		s.Broker().Subscribe(SpikeRid().Event(), func(msg *request.CallRequest) {

			fmt.Println(string(msg.Data))
			msg.OK(map[string]string{"OK": "true"})

		})
	})
}

func TestKafka(t *testing.T) {
	c, err := spike_io.NewServer(":5672")
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	<-c

	t.Log("Testing service")

	os.Setenv("PROVIDER", string(providers.SpikeProvider))

	options := models.ProxyOptions{
		Developer: true,
		SpikeConfig: &models.SpikeConfig{
			SpikeURL: ":5672",
			Debug:    true,
		},
	}

	services := []spikebroker.HandlerService{
		NewKafkaService,
	}

	os.Remove("gorm.db")
	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	_, connected, err := spikebroker.NewProxyServer(db, services,
		func(db *gorm.DB, key uuid.UUID) service.Auth {
			return &auth{}
		}, options)
	if err != nil {
		panic(err)
	}

	<-connected

	t.Log("Started, testing endpoint")
	rErr := Request(SpikeRid().NewRequest(), request.EmptyRequest(), nil)
	if rErr != nil {
		t.FailNow()
	}

	t.Log("Started, testing endpoint")
	rErr = Request(SpikeRid().NewRequest(), request.EmptyRequest(), nil)
	if rErr != nil {
		t.FailNow()
	}

	t.Log("Started, testing endpoint")
	rErr = Request(SpikeRid().NewRequest(), request.EmptyRequest(), nil)
	if rErr != nil {
		t.FailNow()
	}

	t.Log("Started, testing endpoint")
	rErr = Request(SpikeRid().NewRequest(), request.EmptyRequest(), nil)
	if rErr != nil {
		t.FailNow()
	}
}
