package broker

import "encoding/json"

type Success struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}
