package v2

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/spike-events/spike-broker/pkg/rids"
	"github.com/spike-events/spike-broker/pkg/service/request"
)

func RequestV1(p *rids.Pattern, param interface{}, rs interface{}) *request.ErrorRequest {
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
