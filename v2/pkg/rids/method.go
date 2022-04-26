package rids

import (
	"fmt"
	"strings"
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
	LabelValue       string
	ServiceName      string
	ServiceLabel     string
	HttpPrefix       string
	HttpMethod       string
	GenericEndpoint  string
	SpecificEndpoint string
	Params           map[string]fmt.Stringer
	IsPublic         bool
}

func newMethod(serviceName, serviceLabel, label, httpPrefix, endpoint string, params ...fmt.Stringer) *method {
	genericEndpoint := endpoint
	epParts := strings.Split(endpoint, ".")
	var paramsMap = make(map[string]fmt.Stringer)
	for _, param := range params {
		for _, epPart := range epParts {
			if strings.Contains(epPart, "$") {
				paramsMap[epPart[1:]] = param
				endpoint = strings.Replace(endpoint, epPart, param.String(), 1)
				epParts = strings.Split(endpoint, ".")
				break
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
	p.IsPublic = true
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
