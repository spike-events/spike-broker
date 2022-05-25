package v2

import (
	"github.com/gofrs/uuid"
	"github.com/spike-events/spike-broker/pkg/rids"
	"github.com/spike-events/spike-broker/pkg/service"
	"github.com/spike-events/spike-broker/pkg/service/request"
	"gorm.io/gorm"
)

type QueryFilter struct {
	Filter      string `query:"filter"`
	OtherFilter string `query:"otherFilter"`
	EmptyValue  int    `query:"emptyValue"`
}

// V1Service Service
type v1ServiceRid struct {
	rids.Base
}

var v1ServiceImpl v1ServiceRid

func V1Service() *v1ServiceRid {
	if len(v1ServiceImpl.Base.Name()) == 0 {
		v1ServiceImpl.Base = rids.NewRid("v1Service", "")
	}
	return &v1ServiceImpl
}

func (c *v1ServiceRid) CallV2Service(param ...string) *rids.Pattern {
	return c.NewMethod("callV2Service", "callV2Service.$param", param...).NoAuth().Get()
}

// V2 Service RID declared with V1 format
type v2ServiceRid struct {
	rids.Base
}

var v2ServiceImpl v2ServiceRid

func V2Service() *v2ServiceRid {
	if len(v2ServiceImpl.Base.Name()) == 0 {
		v2ServiceImpl.Base = rids.NewRid("serviceTest", "")
	}
	return &v2ServiceImpl
}

func (v2 *v2ServiceRid) TestReply(id ...string) *rids.Pattern {
	return v2.NewMethod("", "reply.$ID", id...).Get()
}

type v1Service struct {
	*service.Base
}

func NewV1Service(db *gorm.DB, key uuid.UUID) service.Service {
	base := service.NewBaseService(db, key, V1Service())
	srv := &v1Service{Base: base}
	return srv
}

func (s *v1Service) Dependencies() []string {
	return append(s.Base.Dependencies(), "configService")
}

func (s *v1Service) Start() {
	s.Init(nil, func() {
		s.Broker().Subscribe(V1Service().CallV2Service(), s.callV2Service)
	})
}

func (s *v1Service) callV2Service(r *request.CallRequest) {
	param, err := uuid.FromString(r.PathParam("param"))
	if err != nil {
		r.Error(err)
		return
	}

	var retID uuid.UUID
	rErr := s.Broker().Request(V2Service().TestReply(param.String()), request.NewRequest(&param), &retID, "token-string")
	if rErr != nil {
		r.ErrorRequest(rErr)
		return
	}

	r.OK(&retID)
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
