package util

import (
	"reflect"
	"strings"
	"sync"
)

type fieldInfo struct {
	index    int
	jsonName string
	omit     bool
}

var fieldsCache sync.Map

func cachedFields(t reflect.Type) []fieldInfo {
	if cached, ok := fieldsCache.Load(t); ok {
		return cached.([]fieldInfo)
	}
	n := t.NumField()
	infos := make([]fieldInfo, 0, n)
	for i := 0; i < n; i++ {
		f := t.Field(i)
		if !f.IsExported() || f.Anonymous {
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name := f.Name
		var opts string
		if tag != "" {
			before, after, _ := strings.Cut(tag, ",")
			name = before
			opts = after
		}
		infos = append(infos, fieldInfo{
			index:    i,
			jsonName: name,
			omit:     opts == "omitempty",
		})
	}
	fieldsCache.Store(t, infos)
	return infos
}

func isEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	default:
		return v.IsZero()
	}
}

func deref(v reflect.Value) any {
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.String {
		return v.String()
	}
	return v.Interface()
}

func StructToMap[T any](v T) map[string]any {
	t := reflect.TypeOf(v)
	rv := reflect.ValueOf(v)
	fields := cachedFields(t)
	m := make(map[string]any, len(fields))
	for _, fi := range fields {
		fv := rv.Field(fi.index)
		if fi.omit && isEmpty(fv) {
			continue
		}
		m[fi.jsonName] = deref(fv)
	}
	return m
}
