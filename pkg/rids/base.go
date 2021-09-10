package rids

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/gofrs/uuid"
	"github.com/hetiansu5/urlquery"
	"log"
	"net/http"
	"reflect"
	"strings"
)

// BaseRid rids base
type BaseRid interface {
	Name() string
	Patterns(BaseRid) []*Pattern
}

// Base rid
type Base struct {
	name  string
	label string
}

func NewRid(name, label string) Base {
	return Base{name, label}
}

// EndpointRest endpoint rest params
type EndpointRest struct {
	method        string
	Endpoint      string
	Params        []string
	Authenticated bool
}

// Pattern estrutura de endpoints
type Pattern struct {
	Label            string
	Service          string
	ServiceLabel     string
	Endpoint         string
	Method           string
	Authenticated    bool
	EndpointNoParams string
	Params           map[string]string
	QueryParams      interface{}

	p *method
}

type method struct {
	label            string
	service          string
	serviceLabel     string
	endpoint         string
	method           string
	auth             bool
	params           map[string]string
	endpointNoParams string
	persistent       bool
}

func (p *Pattern) Auth() bool {
	return p.Authenticated
}

func (p *Pattern) Query(q interface{}) *Pattern {
	p.QueryParams = q
	return p
}

func (p *Pattern) EndpointNoMethod() string {
	if p.Endpoint == "" {
		return p.Service
	}
	return fmt.Sprintf("%v.%v", p.Service, p.Endpoint)
}

func (p *Pattern) EndpointHTTP() string {
	endpoint := strings.ReplaceAll(p.EndpointNoMethod(), ".", "/")
	for key, vl := range p.Params {
		endpoint = strings.ReplaceAll(endpoint, "$"+key, vl)
	}
	var urlQuery string
	if p.QueryParams != nil {
		urlEncoder := urlquery.NewEncoder(urlquery.WithNeedEmptyValue(true))
		urlQueryBytes, _ := urlEncoder.Marshal(p.QueryParams)
		if len(urlQueryBytes) > 0 {
			urlQuery = fmt.Sprintf("?%s", string(urlQueryBytes))
		}
	}
	return fmt.Sprintf("/api/%s%s", endpoint, urlQuery)
}

// EndpointName retorna endpoint do pattern
func (p *Pattern) EndpointName() string {
	if p.Endpoint == "" {
		return fmt.Sprintf("%v.%v", p.Service, p.Method)
	}
	return fmt.Sprintf("%v.%v.%v", p.Service, p.EndpointNoParams, p.Method)
}

func (p *Pattern) Persistent() bool {
	return p.p.persistent
}

func (p Pattern) Concat(c string) *method {
	endpointNoParams := p.EndpointNoParams + "." + c
	endpoint := p.Endpoint + "." + c
	return &method{
		p.p.label,
		p.p.service,
		p.p.serviceLabel,
		endpoint,
		"",
		true,
		p.p.params,
		endpointNoParams,
		false,
	}
}

func (b *Base) ByID(id ...string) *Pattern {
	return b.NewMethod("", "byId.$Id", id...).Get()
}

func (b *Base) NewMethod(label, endpoint string, params ...string) *method {
	endpointNoParams := endpoint
	sp := strings.Split(endpoint, ".")
	var paramsEndpoint = make(map[string]string)
	for _, param := range params {
		for _, rep := range sp {
			if strings.Contains(rep, "$") {
				paramsEndpoint[rep[1:]] = param
				endpoint = strings.Replace(endpoint, rep, param, 1)
				sp = strings.Split(endpoint, ".")
				break
			}
		}
	}
	return &method{
		label,
		b.name,
		b.label,
		endpoint,
		"",
		true,
		paramsEndpoint,
		endpointNoParams,
		false,
	}
}

// Patterns retorna endpoints registrados
func (b *Base) Patterns(rid BaseRid) []*Pattern {
	t := reflect.TypeOf(rid)
	patterns := make([]*Pattern, 0, t.NumMethod())
	for i := 0; i < t.NumMethod(); i++ {
		method := reflect.ValueOf(rid).Method(i)
		if method.Type().NumIn() == 0 || (method.Type().NumIn() == 1 && method.Type().IsVariadic()) {
			ps := method.Call(nil)
			if len(ps) > 0 {
				if v, ok := ps[0].Interface().(*Pattern); ok {
					patterns = append(patterns, v)
				}
			}
		}
	}
	return patterns
}

func (p *method) register(method string) *Pattern {
	pa := &Pattern{
		p.label,
		p.service,
		p.serviceLabel,
		p.endpoint,
		method,
		p.auth,
		p.endpointNoParams,
		p.params,
		nil,
		p,
	}
	return pa
}

func (p *method) NoAuth() *method {
	p.auth = false
	return p
}

func (p *method) Persistent() *method {
	p.persistent = true
	return p
}

func (p *method) Get() *Pattern {
	return p.register("GET")
}

func (p *method) Post() *Pattern {
	return p.register("POST")
}

func (p *method) Put() *Pattern {
	return p.register("PUT")
}

func (p *method) Patch() *Pattern {
	return p.register("PATCH")
}

func (p *method) Delete() *Pattern {
	return p.register("DELETE")
}

func (p *method) Internal() *Pattern {
	return &Pattern{
		p.label,
		p.service,
		p.serviceLabel,
		p.endpoint,
		"INTERNAL",
		false,
		p.endpointNoParams,
		p.params,
		nil,
		p,
	}
}

func (p *Pattern) register(r *chi.Mux, hc func(endpoint EndpointRest, w http.ResponseWriter, r *http.Request)) {
	if r == nil {
		panic("chi.Mux is null")
	}

	var endpoint string
	if p.Endpoint == "" {
		endpoint = string(p.Service)
	} else {
		endpoint = string(p.Service) + "." + p.EndpointNoParams
	}

	parts := strings.Split(endpoint, ".")

	var params []string
	for i, p := range parts {
		if strings.Contains(p, "$") {
			p = strings.ReplaceAll(p, "$", "")
			parts[i] = fmt.Sprintf("{%v}", p)
			params = append(params, p)
		}
	}

	method := fmt.Sprintf("/api/%v", strings.Join(parts, "/"))
	handler := func(w http.ResponseWriter, r *http.Request) {
		e := EndpointRest{
			Endpoint:      p.EndpointName(),
			Params:        params,
			method:        p.Method,
			Authenticated: p.Auth(),
		}
		hc(e, w, r)
	}

	log.Printf("%s -> %s", p.Method, method)
	switch p.Method {
	case "GET":
		r.Get(method, handler)
	case "POST":
		r.Post(method, handler)
	case "PUT":
		r.Put(method, handler)
	case "PATCH":
		r.Patch(method, handler)
	case "DELETE":
		r.Delete(method, handler)
	}
}

// Routes controe todas as rotas
func Routes(patterns []*Pattern, r *chi.Mux, hc func(endpoint EndpointRest, w http.ResponseWriter, r *http.Request)) {
	for _, p := range patterns {
		if p.Method != "INTERNAL" {
			p.register(r, hc)
		}
	}
}

// Routes rids
func (b *Base) Routes(service string, key uuid.UUID) *Pattern {
	endpoint := fmt.Sprintf("route.%v", key.String())
	m := &method{
		"",
		service,
		"",
		endpoint,
		"",
		false,
		nil,
		endpoint,
		false,
	}
	return m.Get()
}

// Name retorna nome do serviço
func (b *Base) Name() string {
	return b.name
}
