package rids

import (
	"fmt"
	"reflect"
)

// Resource rids base
type Resource interface {
	Name() string
	WSPrefix() string
	Patterns(Resource) []Pattern
}

// Base rid
type Base struct {
	name       string
	label      string
	httpPrefix string
	wsPrefix   string
}

func NewRid(name, label, httpPrefix, wsPrefix string) Base {
	return Base{
		name:       name,
		label:      label,
		httpPrefix: httpPrefix,
		wsPrefix:   wsPrefix,
	}
}

func (b *Base) NewMethod(label, endpoint string, params ...fmt.Stringer) Method {
	return newMethod(b.name, b.label, label, b.httpPrefix, endpoint, params...)
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
func (b *Base) Patterns(rid Resource) []Pattern {
	t := reflect.TypeOf(rid)
	patterns := make([]Pattern, 0, t.NumMethod())
	for i := 0; i < t.NumMethod(); i++ {
		methodName := reflect.ValueOf(rid).Method(i)
		if methodName.Type().NumIn() == 0 || (methodName.Type().NumIn() == 1 && methodName.Type().IsVariadic()) {
			ps := methodName.Call(nil)
			if len(ps) > 0 {
				if v, ok := ps[0].Interface().(*pattern); ok {
					patterns = append(patterns, v)
				}
			}
		}
	}
	return patterns
}
