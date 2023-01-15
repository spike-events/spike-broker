package broker

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spike-events/spike-broker/pkg/service/request"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
	spikeUtils "github.com/spike-events/spike-broker/v2/pkg/spike-utils"
)

type callV1 struct {
	call
}

type v1Message struct {
	Message
	DataIface json.RawMessage `json:"data"`
}

func NewCallV1FromJSON(data []byte, p rids.Pattern, reply string) (Call, error) {
	type fromV1Data struct {
		Params map[string]string `json:"Params"`
		Data   RawData           `json:"Data"`
		Token  string            `json:"Token"`
		Query  string            `json:"Query"`
	}
	var v1Data fromV1Data
	err := json.Unmarshal(data, &v1Data)
	if err != nil {
		return nil, err
	}

	params := make(map[string]fmt.Stringer, 0)
	for i := range v1Data.Params {
		params[i] = spikeUtils.Stringer(v1Data.Params[i])
	}

	v1 := callV1{
		call: call{
			callBase: callBase{
				Data:            v1Data.Data,
				ReplyStr:        reply,
				EndpointPattern: p.Clone(),
				Token:           []byte(v1Data.Token),
			},
			APIVersion: 1,
		},
	}
	v1.EndpointPattern.SetParams(params)
	return &v1, nil
}

func NewCallFromV1(r *request.CallRequest) (Call, error) {
	p, err := rids.NewPatternFromV1(r.Endpoint, r.Params)
	if err != nil {
		return nil, err
	}
	v1 := callV1{
		call: call{
			callBase: callBase{
				Data:            RawData(r.Data),
				EndpointPattern: p,
				Token:           []byte(r.Token),
			},
			APIVersion: 2,
		},
	}
	return &v1, nil
}

func (c *callV1) ToJSON() json.RawMessage {
	data, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	return data
}

// OK result
func (c *callV1) OK(result ...interface{}) {
	if c.ReplyStr == "" {
		return
	}

	var success v1Message
	success.CodeInt = http.StatusOK

	if len(result) == 0 {
		data, err := json.Marshal(&success)
		if err != nil {
			panic(err)
		}
		c.provider.Reply(c.ReplyStr, data)
		return
	}

	switch result[0].(type) {
	case string:
		success.DataIface = []byte(result[0].(string))
		data, err := json.Marshal(&success)
		if err != nil {
			panic(err)
		}
		c.provider.Reply(c.ReplyStr, data)
		return
	case []byte:
		success.DataIface = result[0].([]byte)
		data, err := json.Marshal(&success)
		if err != nil {
			panic(err)
		}
		c.provider.Reply(c.ReplyStr, data)
		return
	}

	// Make sure we always marshal pointer structures
	result[0] = spikeUtils.PointerFromInterface(result[0])
	payload, err := json.Marshal(result[0])
	if err != nil {
		panic(err)
	}
	success.DataIface = payload
	data, err := json.Marshal(&success)
	if err != nil {
		panic(err)
	}
	c.provider.Reply(c.Reply(), data)
}
