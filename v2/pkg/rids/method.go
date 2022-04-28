package rids

import (
	"encoding/json"
	"fmt"
	"strings"

	spikeutils "github.com/spike-events/spike-broker/v2/pkg/spike-utils"
)

type Method interface {
	Public() Method
	Get() Pattern
	Post() Pattern
	Put() Pattern
	Patch() Pattern
	Delete() Pattern
	Internal() Pattern
}

type method struct {
	LabelValue       string                  `json:"labelValue"`
	ServiceName      string                  `json:"serviceName"`
	ServiceLabel     string                  `json:"serviceLabel"`
	HttpPrefix       string                  `json:"httpPrefix"`
	HttpMethod       string                  `json:"httpMethod"`
	GenericEndpoint  string                  `json:"genericEndpoint"`
	SpecificEndpoint string                  `json:"specificEndpoint"`
	Params           map[string]fmt.Stringer `json:"params"`
	IsPublic         bool                    `json:"isPublic"`
}

func (m *method) UnmarshalJSON(data []byte) error {
	type methodInnerType struct {
		LabelValue       string            `json:"labelValue"`
		ServiceName      string            `json:"serviceName"`
		ServiceLabel     string            `json:"serviceLabel"`
		HttpPrefix       string            `json:"httpPrefix"`
		HttpMethod       string            `json:"httpMethod"`
		GenericEndpoint  string            `json:"genericEndpoint"`
		SpecificEndpoint string            `json:"specificEndpoint"`
		Params           map[string]string `json:"params"`
		IsPublic         bool              `json:"isPublic"`
	}
	var methodInner methodInnerType
	if err := json.Unmarshal(data, &methodInner); err != nil {
		return err
	}
	m.LabelValue = methodInner.LabelValue
	m.ServiceName = methodInner.ServiceName
	m.ServiceLabel = methodInner.ServiceLabel
	m.HttpPrefix = methodInner.HttpPrefix
	m.HttpMethod = methodInner.HttpMethod
	m.GenericEndpoint = methodInner.GenericEndpoint
	m.SpecificEndpoint = methodInner.SpecificEndpoint
	m.IsPublic = methodInner.IsPublic
	m.Params = make(map[string]fmt.Stringer)
	for name, value := range methodInner.Params {
		m.Params[name] = spikeutils.Stringer(value)
	}
	return nil
}

func newMethod(serviceName, serviceLabel, label, httpPrefix, endpoint string, params ...fmt.Stringer) *method {
	genericEndpoint := endpoint
	epParts := strings.Split(endpoint, ".")
	var paramsMap = make(map[string]fmt.Stringer)
	for _, param := range params {
		for _, epPart := range epParts {
			if strings.Contains(epPart, "$") && len(epPart) > 1 {
				paramsMap[epPart[1:]] = param
				endpoint = strings.Replace(endpoint, epPart, param.String(), 1)
				epParts = strings.Split(endpoint, ".")
				break
			}
		}
	}

	for _, epPart := range epParts {
		if strings.Contains(epPart, "$") {
			epPartName := epPart[1:]
			if _, ok := paramsMap[epPartName]; !ok {
				paramsMap[epPartName] = spikeutils.Stringer("")
			}
		}
	}

	return &method{
		LabelValue:       label,
		ServiceName:      serviceName,
		ServiceLabel:     serviceLabel,
		HttpPrefix:       httpPrefix,
		HttpMethod:       "",
		GenericEndpoint:  genericEndpoint,
		SpecificEndpoint: endpoint,
		Params:           paramsMap,
	}
}

func (m *method) updateParams(params map[string]fmt.Stringer) {
	m.Params = params
}

func (m *method) Public() Method {
	if m.HttpMethod != "INTERNAL" {
		m.IsPublic = true
	}
	return m
}

func (m *method) Get() Pattern {
	m.HttpMethod = "GET"
	return newPattern(m)
}

func (m *method) Post() Pattern {
	m.HttpMethod = "POST"
	return newPattern(m)
}

func (m *method) Put() Pattern {
	m.HttpMethod = "PUT"
	return newPattern(m)
}

func (m *method) Patch() Pattern {
	m.HttpMethod = "PATCH"
	return newPattern(m)
}

func (m *method) Delete() Pattern {
	m.HttpMethod = "DELETE"
	return newPattern(m)
}

func (m *method) Internal() Pattern {
	m.IsPublic = false
	m.HttpMethod = "INTERNAL"
	return newPattern(m)
}
