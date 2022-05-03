package rids

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hetiansu5/urlquery"
)

// Pattern interface for building up endpoints
type Pattern interface {
	Public() bool
	Query(q interface{}) Pattern
	Service() string
	Method() string
	Params() map[string]fmt.Stringer
	EndpointREST() string
	EndpointName() string
	EndpointNameSpecific() string
	QueryParams() interface{}
	SetParams(params map[string]fmt.Stringer)
	Clone() Pattern
	Version() int
}

func newPattern(m *method) Pattern {
	return &pattern{
		MethodValue:      m,
		QueryParamsValue: nil,
	}
}

func UnmarshalPattern(data json.RawMessage) (Pattern, error) {
	var p pattern
	err := json.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

type pattern struct {
	MethodValue      *method     `json:"methodValue"`
	QueryParamsValue interface{} `json:"queryParamsValue"`
}

func (p *pattern) Version() int {
	return p.MethodValue.Version
}

func (p *pattern) Clone() Pattern {
	mClone := *p.MethodValue
	clone := newPattern(&mClone).(*pattern)
	clone.QueryParamsValue = p.QueryParamsValue
	return clone
}

func (p *pattern) QueryParams() interface{} {
	return p.QueryParamsValue
}

func (p *pattern) Params() map[string]fmt.Stringer {
	return p.MethodValue.Params
}

func (p *pattern) Public() bool {
	return p.MethodValue.IsPublic
}

func (p *pattern) Query(q interface{}) Pattern {
	p.QueryParamsValue = q
	return p
}

func (p *pattern) Service() string {
	return p.MethodValue.ServiceName
}

func (p *pattern) Method() string {
	return p.MethodValue.HttpMethod
}

func (p *pattern) EndpointREST() string {
	endpoint := p.EndpointNameSpecific()
	endpointParts := strings.Split(endpoint, ".")
	endpoint = strings.Join(endpointParts[:len(endpointParts)-1], "/")
	var urlQuery string
	switch p.QueryParamsValue.(type) {
	case nil:
	case []byte:
		if len(p.QueryParamsValue.([]byte)) > 0 {
			urlQuery = fmt.Sprintf("?%s", string(p.QueryParamsValue.([]byte)))
		}
	case string:
		if len(p.QueryParamsValue.(string)) > 0 {
			urlQuery = fmt.Sprintf("?%s", p.QueryParamsValue.(string))
		}
	default:
		urlEncoder := urlquery.NewEncoder(urlquery.WithNeedEmptyValue(true))
		urlQueryBytes, _ := urlEncoder.Marshal(p.QueryParamsValue)
		if len(urlQueryBytes) > 0 {
			urlQuery = fmt.Sprintf("?%s", string(urlQueryBytes))
		}
	}
	var returnValue string
	if p.MethodValue.HttpPrefix != "" {
		returnValue = fmt.Sprintf("%s/", p.MethodValue.HttpPrefix)
	}
	return fmt.Sprintf("/%s%s%s", returnValue, endpoint, urlQuery)
}

// EndpointName returns generic SpecificEndpoint version (with patterns)
func (p *pattern) EndpointName() string {
	return fmt.Sprintf("%v.%v.%v", p.Service(), p.MethodValue.GenericEndpoint, p.MethodValue.HttpMethod)
}

func (p *pattern) EndpointNameSpecific() string {
	return fmt.Sprintf("%v.%v.%v", p.Service(), p.MethodValue.SpecificEndpoint, p.MethodValue.HttpMethod)
}

func (p *pattern) SetParams(params map[string]fmt.Stringer) {
	m := *p.MethodValue
	m.updateParams(params)
	p.MethodValue = &m
}
