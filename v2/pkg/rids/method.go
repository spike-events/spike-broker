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

func (p *method) UnmarshalJSON(data []byte) error {
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
	p.LabelValue = methodInner.LabelValue
	p.ServiceName = methodInner.ServiceName
	p.ServiceLabel = methodInner.ServiceLabel
	p.HttpPrefix = methodInner.HttpPrefix
	p.HttpMethod = methodInner.HttpMethod
	p.GenericEndpoint = methodInner.GenericEndpoint
	p.SpecificEndpoint = methodInner.SpecificEndpoint
	p.IsPublic = methodInner.IsPublic
	p.Params = make(map[string]fmt.Stringer)
	for name, value := range methodInner.Params {
		p.Params[name] = spikeutils.Stringer(value)
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

func (p *method) updateParams(params map[string]fmt.Stringer) {
	p.Params = params
}

func (p *method) Public() Method {
	if p.HttpMethod != "INTERNAL" {
		p.IsPublic = true
	}
	return p
}

func (p *method) Get() Pattern {
	p.HttpMethod = "GET"
	return newPattern(p)
}

func (p *method) Post() Pattern {
	p.HttpMethod = "POST"
	return newPattern(p)
}

func (p *method) Put() Pattern {
	p.HttpMethod = "PUT"
	return newPattern(p)
}

func (p *method) Patch() Pattern {
	p.HttpMethod = "PATCH"
	return newPattern(p)
}

func (p *method) Delete() Pattern {
	p.HttpMethod = "DELETE"
	return newPattern(p)
}

func (p *method) Internal() Pattern {
	p.IsPublic = false
	p.HttpMethod = "INTERNAL"
	return newPattern(p)
}
