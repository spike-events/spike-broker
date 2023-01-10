package v2

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net/http"

	"github.com/spike-events/spike-broker/v2/pkg/broker"
	"github.com/spike-events/spike-broker/v2/pkg/rids"
)

var client = &http.Client{}

func Request(p rids.Pattern, param interface{}, rs interface{}, token ...string) broker.Error {
	endpoint := "http://localhost:3333" + p.EndpointREST()

	data, _ := json.Marshal(param)
	payload := bytes.NewReader(data)
	req, err := http.NewRequest(p.Method(), endpoint, payload)
	if err != nil {
		return broker.InternalError(err)

	}
	req.Header.Add("Content-Type", "application/json")
	if len(token) > 0 {
		req.Header.Add("Authorization", "Bearer "+token[0])
	}

	res, err := client.Do(req)
	if err != nil {
		return broker.InternalError(err)
	}

	defer res.Body.Close()

	if math.Abs(float64(res.StatusCode-http.StatusOK)) >= 100 {
		return broker.NewError("request failed", res.StatusCode, nil)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return broker.InternalError(err)
	}

	if res.Header.Get("Content-Type") == "application/json" {
		rError := broker.NewMessageFromJSON(body)
		if rError != nil && rError.Code() > http.StatusOK {
			return rError
		}

		if rs != nil {
			err = json.Unmarshal(body, rs)
			if err != nil {
				return broker.InternalError(err)
			}
		}
	} else if rs != nil {
		if _, ok := rs.(*[]byte); ok {
			*rs.(*[]byte) = body
		}
	}

	return nil
}
