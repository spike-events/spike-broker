package test

import (
	"encoding/json"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/nats-io/nats.go"
	apiproxynats "github.com/spike-events/spike-broker"
	"github.com/spike-events/spike-broker/pkg/models"
	"github.com/spike-events/spike-broker/pkg/providers"
	"github.com/spike-events/spike-broker/pkg/rids"
	"github.com/spike-events/spike-broker/pkg/service"
	"github.com/spike-events/spike-broker/pkg/service/request"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type QueryFilter struct {
	Filter      string `query:"filter"`
	OtherFilter string `query:"otherFilter"`
	EmptyValue  int    `query:"emptyValue"`
}

// Config Service
type config struct {
	rids.Base
}

var configImpl config

// Config returns the RID mapper for Config Service
func Config() *config {
	if len(configImpl.Base.Name()) == 0 {
		configImpl.Base = rids.NewRid("configService", "")
	}
	return &configImpl
}

func (c *config) GetServiceConfig(serviceName ...string) *rids.Pattern {
	return c.NewMethod("GetServiceConfig", "GetServiceConfig.$Service", serviceName...).Internal()
}

type configService struct {
	*service.Base
}

func NewConfigService(db *gorm.DB, key uuid.UUID) service.Service {
	base := service.NewBaseService(db, key, Config())
	srv := &configService{Base: base}
	return srv
}

func (s *configService) Start() {
	s.Init(nil, func() {
		s.Broker().Subscribe(Config().GetServiceConfig(), s.getServiceConfig)
	})
}

func (s *configService) AfterStart() {
}

func (s *configService) getServiceConfig(r *request.CallRequest) {
}

// OtherConfig Service
type otherConfig struct {
	rids.Base
}

var otherConfigImpl otherConfig

func OtherConfig() *otherConfig {
	if len(otherConfigImpl.Base.Name()) == 0 {
		otherConfigImpl.Base = rids.NewRid("otherConfigService", "")
	}
	return &otherConfigImpl
}

func (c *otherConfig) GetServiceConfig(serviceName ...string) *rids.Pattern {
	return c.NewMethod("getOtherServiceConfig", "getOtherServiceConfig.$Service", serviceName...).NoAuth().Get()
}

func (c *otherConfig) GetAfterRestart(serviceName ...string) *rids.Pattern {
	return c.NewMethod("getOtherServiceConfig", "getOtherServiceConfig.$Service.afterRestart", serviceName...).NoAuth().Get()
}

type otherConfigService struct {
	*service.Base
}

func NewOtherConfigService(db *gorm.DB, key uuid.UUID) service.Service {
	base := service.NewBaseService(db, key, OtherConfig())
	srv := &otherConfigService{Base: base}
	return srv
}

func (s *otherConfigService) Dependencies() []string {
	return append(s.Base.Dependencies(), "configService")
}

func (s *otherConfigService) Start() {
	s.Init(nil, func() {
		s.Broker().Subscribe(OtherConfig().GetServiceConfig(), s.getOtherServiceConfig)
	})
}

func (s *otherConfigService) StartAfterDependency(dep string) {
	fmt.Println("other service received dependency", dep)
	c := make(chan bool, 3)
	go func() {
		s.Lock()
		time.Sleep(300 * time.Millisecond)
		s.Unlock()
		c <- true
	}()

	go func() {
		s.Lock()
		time.Sleep(400 * time.Millisecond)
		s.Unlock()
		c <- true
	}()

	go func() {
		s.Lock()
		time.Sleep(200 * time.Millisecond)
		s.Unlock()
		c <- true
	}()
	for range make([]int, 3) {
		<-c
	}
	log.Printf("%s: finished\n", s.Rid().Name())
}

func (s *otherConfigService) IncludeAfterRestart() {
	s.Broker().Subscribe(OtherConfig().GetAfterRestart(), s.getAfterRestart)
}

func (s *otherConfigService) getOtherServiceConfig(r *request.CallRequest) {
	var str string
	r.ParseData(&str)
	if len(str) > 0 {
		r.ErrorRequest(&request.ErrorStatusForbidden)
		return
	}
	var filter QueryFilter
	r.ParseQuery(&filter)
	s.IncludeAfterRestart()
	r.OK()
}

func (s *otherConfigService) getAfterRestart(r *request.CallRequest) {
	r.OK()
}

// Authenticator
type auth struct {
	*service.Base
}

func (s *auth) ValidateToken(token string) (string, bool) {
	return token, true
}

func (s *auth) UserHavePermission(r *request.CallRequest) bool {
	return true
}

func (s *auth) NewPermission(r *request.CallRequest) {

}

func TestService(t *testing.T) {
	t.Log("Testing service")

	os.Setenv("PROVIDER", string(providers.NatsProvider))

	options := models.ProxyOptions{
		Developer: true,
		NatsConfig: &models.NatsConfig{
			LocalNats: true,
			NatsURL:   nats.DefaultURL,
		},
	}

	services := []apiproxynats.HandlerService{
		NewOtherConfigService,
		NewConfigService,
	}

	os.Remove("gorm.db")
	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	_, connected, err := apiproxynats.NewProxyServer(db, services,
		func(db *gorm.DB, key uuid.UUID) service.Auth {
			return &auth{}
		}, options)
	if err != nil {
		panic(err)
	}

	<-connected

	t.Log("Started, testing endpoint")
	rErr := Request(OtherConfig().GetServiceConfig(Config().Name()), request.EmptyRequest(), nil)
	if rErr != nil {
		t.FailNow()
	}

	time.Sleep(3 * time.Second)

	t.Log("Started, testing endpoint after restart")
	rErr = Request(OtherConfig().GetAfterRestart(Config().Name()), request.EmptyRequest(), nil)
	if rErr != nil {
		t.FailNow()
	}

	time.Sleep(3 * time.Second)

	t.Log("Started, testing endpoint")
	query := &QueryFilter{
		Filter:      "filter1",
		OtherFilter: "filter2",
		EmptyValue:  0,
	}
	rErr = Request(OtherConfig().GetServiceConfig(Config().Name()).Query(query), request.EmptyRequest(), nil)
	if rErr != nil {
		t.FailNow()
	}

	time.Sleep(3 * time.Second)
	t.Log("Started, testing endpoint failure")
	rErr = Request(OtherConfig().GetServiceConfig(Config().Name()), "error", nil)
	if rErr == nil {
		t.FailNow()
	}
}

var client = &http.Client{}

func Request(p *rids.Pattern, param interface{}, rs interface{}) *request.ErrorRequest {
	endpoint := "http://localhost:3333" + p.EndpointHTTP()

	data, _ := json.Marshal(param)
	payload := strings.NewReader(string(data))
	req, err := http.NewRequest(p.Method, endpoint, payload)
	if err != nil {
		return &request.ErrorRequest{
			Message: err.Error(),
			Error:   err,
			Code:    http.StatusInternalServerError,
		}
	}
	req.Header.Add("Content-Type", "application/json")
	//if len(token) > 0 {
	//	req.Header.Add("Authorization", "Bearer "+token[0].SessionEncoded)
	//}

	res, err := client.Do(req)
	if err != nil {
		return &request.ErrorRequest{
			Message: err.Error(),
			Error:   err,
			Code:    http.StatusInternalServerError,
		}
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)

	if res.StatusCode != 200 {
		return &request.ErrorRequest{
			Message: string(body),
			Error:   fmt.Errorf(string(body)),
			Code:    res.StatusCode,
		}
	}

	if err != nil {
		return &request.ErrorRequest{
			Message: string(body),
			Error:   fmt.Errorf(string(body)),
			Code:    http.StatusInternalServerError,
		}
	}

	var rError *request.ErrorRequest
	rError.Parse(body)

	if rError != nil && rError.Code > 200 {
		return rError
	}

	if rs != nil {
		err = json.Unmarshal(body, rs)
		if err != nil {
			return &request.ErrorRequest{
				Message: err.Error(),
				Error:   err,
				Code:    http.StatusInternalServerError,
			}
		}
	}

	return nil
}
