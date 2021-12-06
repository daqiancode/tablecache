package tablecache

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const NullStr = "null"

var EmptyStruct = struct{}{}

// MakeKey make cache key. eg. /k1/v1/k2/v2
func makeKeyWithMap(indexValues map[string]interface{}) string {
	keys := make([]string, len(indexValues))
	values := make([]interface{}, len(indexValues))
	i := 0
	for k := range indexValues {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	i = 0
	for _, k := range keys {
		values[i] = indexValues[k]
		i++
	}
	tpl := ""
	for _, v := range keys {
		tpl += v + "/%v"
	}
	tpl = strings.ToLower(tpl)
	return fmt.Sprintf(tpl, values...)
}

//MakeKey MakeKey make cache key. eg. /k1/v1/k2/v2. input like: k1,v1,k2,v2
func makeKey(kvs ...interface{}) string {
	n := len(kvs) / 2
	m := make(map[string]interface{}, n)
	for i := 0; i < n; i++ {
		m[kvs[2*i].(string)] = kvs[2*i+1]
	}
	return makeKeyWithMap(m)
}

// subtract a-b
func subtract(a, b []uint64) []uint64 {
	m := make(map[uint64]bool, len(b))
	for _, v := range b {
		m[v] = true
	}
	var r []uint64
	for _, v := range a {
		if _, ok := m[v]; !ok {
			r = append(r, v)
		}
	}
	return r
}

func hasNil(vs map[string]string) bool {
	for _, v := range vs {
		if v == "" {
			return true
		}
	}
	return false
}
func argsToMap(args ...interface{}) map[string]interface{} {
	n := len(args)/2 + len(args)%2
	r := make(map[string]interface{}, n)
	for i := 0; i < n; i++ {
		r[args[2*i].(string)] = args[2*i+1]
	}
	if len(args)%2 == 1 {
		r[args[len(args)-1].(string)] = nil
	}
	return r
}

// func pick(structRef interface{}, fields ...string) map[string]interface{} {
// 	sr := NewStructRef(structRef)
// 	r := make(map[string]interface{}, len(fields))
// 	for _, f := range fields {
// 		r[f] = sr.GetIgnoreCase(f)
// 	}
// 	return r
// }
func pickFromMap(m map[string]interface{}, fields ...string) map[string]interface{} {
	r := make(map[string]interface{}, len(fields))
	for k, v := range m {
		for _, f := range fields {
			if strings.EqualFold(k, f) {
				r[f] = v
			}
		}
	}
	return r
}

type CacheUtil struct{}

func (s *CacheUtil) MakeKeyWithMap(kvs map[string]interface{}) string {
	return makeKeyWithMap(kvs)
}
func (s *CacheUtil) MakeKey(kvs ...interface{}) string {
	return makeKey(kvs...)
}

//Stringify transform value to string format
func (s *CacheUtil) Stringify(value interface{}) string {
	switch value := value.(type) {
	case string:
		return value
	case uint64:
		return strconv.FormatUint(value, 10)
	}
	return fmt.Sprintf("%v", value)
}

// Get field from
func (s *CacheUtil) GetFieldValue(structValue interface{}, field string) interface{} {
	return reflect.Indirect(reflect.ValueOf(structValue)).FieldByName(field).Interface()
}

func GetStructFields(root reflect.Type) []string {
	for root.Kind() == reflect.Ptr {
		root = root.Elem()
	}
	n := root.NumField()
	var r []string
	for i := 0; i < n; i++ {
		f := root.Field(i)
		if !f.Anonymous {
			r = append(r, f.Name)
		} else {
			r = append(r, GetStructFields(f.Type)...)
		}
	}
	return r
}

// a-b , in a not inb
func SubStrs(a, b []string) []string {
	m := make(map[string]bool, len(b))
	for _, v := range b {
		m[v] = true
	}
	var r []string
	for _, v := range a {
		if !m[v] {
			r = append(r, v)
		}
	}
	return r
}

func ToStringSlice(a interface{}) []string {
	if r, ok := a.([]string); ok {
		return r
	}
	aV := reflect.Indirect(reflect.ValueOf(a))
	n := aV.Len()
	r := make([]string, n)
	for i := 0; i < n; i++ {
		r[i] = fmt.Sprintf("%v", reflect.Indirect(aV.Index(i)).Interface())
	}
	return r
}

func isSlice(value interface{}) bool {
	return reflect.Indirect(reflect.ValueOf(value)).Kind() == reflect.Slice
}
