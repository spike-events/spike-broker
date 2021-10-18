package route

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi"
	"github.com/spike-events/spike-broker/pkg/models"
	"github.com/spike-events/spike-broker/pkg/rids"
	"github.com/spike-events/spike-broker/pkg/service/request"
	"github.com/vincent-petithory/dataurl"
)

func (s *routeService) handler(endpoint rids.EndpointRest, w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()
	r.ParseMultipartForm(0)

	debugLevel := os.Getenv("API_LOG_LEVEL")

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		if debugLevel == "DEBUG" || debugLevel == "ERR" {
			log.Printf("api: failed to read all body: %v", err)
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	params := make(map[string]string)
	for _, p := range endpoint.Params {
		value := r.FormValue(p)
		if value != "" {
			params[p] = value
		}
	}
	for _, p := range endpoint.Params {
		value := chi.URLParam(r, p)
		if value != "" {
			params[p] = value
		}
	}
	for key, value := range r.URL.Query() {
		if len(value) > 0 {
			params[key] = value[0]
		}
	}
	token := r.Header.Get("token")
	call := request.CallRequest{
		Data:     data,
		Params:   params,
		Form:     r.Form,
		PostForm: r.PostForm,
		Endpoint: endpoint.Endpoint,
	}

	if len(token) > 0 {
		call.Token = token
	}

	if r.Method == http.MethodGet {
		query := r.Form.Encode()
		if len(query) > 0 {
			call.Query = query
		}
	}

	values := strings.Split(endpoint.Endpoint, ".")

	permission := models.HavePermissionRequest{
		Service:  values[0],
		Endpoint: strings.ReplaceAll(strings.Join(values, "."), "."+values[len(values)-1], ""),
		Method:   values[len(values)-1],
	}

	if endpoint.Authenticated && len(s.auths) > 0 {

		callAuth := request.NewRequest(permission)
		callAuth.Form = r.Form
		if token != "" {
			callAuth.Token = token
		}

		if !s.auths[0].Auth.UserHavePermission(callAuth) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	if debugLevel == "DEBUG" {
		log.Printf("api: requesting endpoint %s", endpoint.Endpoint)
	}
	result, rErr := s.Broker().RequestRaw(endpoint.Endpoint, call.ToJSON())
	if debugLevel == "DEBUG" {
		if rErr != nil {
			log.Printf("api: finished requesting endpoint with error %s", endpoint.Endpoint)
		} else {
			log.Printf("api: finished requesting endpoint %s without erros", endpoint.Endpoint)
		}
	}

	if rErr != nil {
		if debugLevel == "DEBUG" || debugLevel == "ERR" {
			log.Printf("api: failed to call endpoint: %v", rErr)
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("%v", rErr)))
		return
	}

	var parsedResponse request.ErrorRequest
	if parsedResponse.Parse(result); parsedResponse.Message != "" {
		if debugLevel == "DEBUG" || debugLevel == "ERR" {
			log.Printf("api: received error response: %s", parsedResponse.ToJSON())
		}
		w.WriteHeader(parsedResponse.Code)
		w.Write(parsedResponse.ToJSON())
		return
	}

	var dataURL dataurl.DataURL
	json.Unmarshal(parsedResponse.Data, &dataURL)

	if dataURL.ContentType() != "" && len(dataURL.Data) > 0 {
		w.Header().Set("Content-Type", dataURL.ContentType())
		w.Header().Set("ETag", fmt.Sprintf("%x", sha256.Sum256(dataURL.Data)))
		if len(dataURL.Params) > 0 {
			if filename, ok := dataURL.Params["filename"]; ok {
				w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write(dataURL.Data)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(parsedResponse.Data)
}
