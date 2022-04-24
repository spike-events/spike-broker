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
	Prefix           string
	HttpMethod       string
	GenericEndpoint  string
	SpecificEndpoint string
	Params           map[string]fmt.Stringer
	IsPublic         bool
}

func newMethod(serviceName, serviceLabel, label, prefix, endpoint string, params ...fmt.Stringer) *method {
	genericEndpoint := endpoint
	sp := strings.Split(endpoint, ".")
	var paramsMap = make(map[string]fmt.Stringer)
	for _, param := range params {
		for _, rep := range sp {
			if strings.Contains(rep, "$") {
				paramsMap[rep[1:]] = param
				endpoint = strings.Replace(endpoint, rep, param.String(), 1)
				sp = strings.Split(endpoint, ".")
				break
			}
		}
	}

	return &method{
		LabelValue:       label,
		ServiceName:      serviceName,
		ServiceLabel:     serviceLabel,
		Prefix:           prefix,
		HttpMethod:       "",
		GenericEndpoint:  genericEndpoint,
		SpecificEndpoint: endpoint,
		Params:           paramsMap,
	}
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
