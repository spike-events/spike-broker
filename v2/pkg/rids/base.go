package rids

import (
	"fmt"
	"net/http"
	"reflect"
)

// Resource rids base
type Resource interface {
	Name() string
	WSPrefix() string
}

// Base rid
type Base struct {
	name       string
	label      string
	httpPrefix string
	wsPrefix   string
	version    int
}

func NewRid(name, label, httpPrefix string, version ...int) Base {
	var ver int
	if len(version) > 0 {
		switch version[0] {
		case 1, 2:
			ver = version[0]
		default:
			panic("invalid API version")
		}
	} else {
		ver = 2
	}

	return Base{
		name:       name,
		label:      label,
		httpPrefix: httpPrefix,
		version:    ver,
	}
}

func (b *Base) NewMethod(label, endpoint string, params ...fmt.Stringer) Method {
	return newMethod(b.name, b.label, label, b.httpPrefix, endpoint, b.version, params...)
}

func (b *Base) ByID(id ...fmt.Stringer) Pattern {
	return b.NewMethod("", "byId.$Id", id...).Get()
}

// Name retorna nome do serviço
func (b *Base) Name() string {
	return b.name
}

func (b *Base) WSPrefix() string {
	return b.wsPrefix
}

// Patterns retorna endpoints registrados
func Patterns(rid Resource) []Pattern {
	t := reflect.TypeOf(rid)
	patterns := make([]Pattern, 0, t.NumMethod())
	for i := 0; i < t.NumMethod(); i++ {
		methodName := reflect.ValueOf(rid).Method(i)
		methodType := methodName.Type()
		patterKind := reflect.TypeOf((*Pattern)(nil)).Elem()
		if methodType.NumIn() == 0 || (methodType.NumIn() == 1) && methodType.NumOut() == 1 && methodType.IsVariadic() && methodType.Out(0).Implements(patterKind) {
			ps := methodName.Call(nil)
			if len(ps) > 0 {
				if v, ok := ps[0].Interface().(*pattern); ok {
					switch v.Method() {
					case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost:
						patterns = append(patterns, v)
					}
				}
			}
		}
	}
	return patterns
}
