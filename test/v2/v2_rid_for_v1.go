package v2

import (
	"fmt"

	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

type v2RidForV1 struct {
	rids.Base
}

var v2RidForV1Impl *v2RidForV1

func V2RidForV1Service() *v2RidForV1 {
	if v2RidForV1Impl == nil {
		v2RidForV1Impl = &v2RidForV1{
			Base: rids.NewRid("v1Service", "V2 interface for Service V1", "api", 1),
		}
	}
	return v2RidForV1Impl
}

func (v *v2RidForV1) AnswerV2Service(id ...fmt.Stringer) rids.Pattern {
	return v.NewMethod("", "answerV2Service.$ID", id...).Post()
}

func (v *v2RidForV1) AnswerV2Forbidden() rids.Pattern {
	return v.NewMethod("", "answerV2Forbidden").Get()
}
