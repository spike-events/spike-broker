package rids

import (
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
}

type pattern struct {
	MethodValue      *method
	QueryParamsValue interface{}
}

func newPattern(m *method) Pattern {
	return &pattern{
		MethodValue:      m,
		QueryParamsValue: nil,
	}
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
	endpoint := p.MethodValue.SpecificEndpoint
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
	if p.MethodValue.Prefix != "" {
		returnValue = fmt.Sprintf("%s/", p.MethodValue.Prefix)
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

//func (p *pattern) register(r *chi.Mux, hc func(SpecificEndpoint EndpointRest, w http.ResponseWriter, r *http.Request)) {
//	if r == nil {
//		panic("chi.Mux is null")
//	}
//
//	var SpecificEndpoint string
//	if p.Endpoint == "" {
//		SpecificEndpoint = string(p.Service)
//	} else {
//		SpecificEndpoint = string(p.Service) + "." + p.EndpointNoParams
//	}
//
//	parts := strings.Split(SpecificEndpoint, ".")
//
//	var Params []fmt.Stringer
//	for i, part := range parts {
//		if strings.Contains(part, "$") {
//			part = strings.ReplaceAll(part, "$", "")
//			parts[i] = fmt.Sprintf("{%v}", part)
//			Params = append(Params, strings.NewReader(part))
//		}
//	}
//
//	methodName := fmt.Sprintf("/api/%v", strings.Join(parts, "/"))
//	handler := func(w http.ResponseWriter, r *http.Request) {
//		e := EndpointRest{
//			Endpoint:      p.EndpointName(),
//			Params:        Params,
//			HttpMethod:        p.Method,
//			Authenticated: p.Public(),
//		}
//		hc(e, w, r)
//	}
//
//	log.Printf("%s -> %s", p.Method, methodName)
//	switch p.Method {
//	case "GET":
//		r.Get(methodName, handler)
//	case "POST":
//		r.Post(methodName, handler)
//	case "PUT":
//		r.Put(methodName, handler)
//	case "PATCH":
//		r.Patch(methodName, handler)
//	case "DELETE":
//		r.Delete(methodName, handler)
//	}
//}
