package hls

import (
	"fmt"
	"image"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/as/hls/m3u"
)

func marshalTag0(s interface{}) ([]m3u.Tag, error) {
	return marshalTag(reflect.ValueOf(s))
}

func unmarshalTag0(s interface{}, t ...m3u.Tag) error {
	return unmarshalTag(reflect.ValueOf(s), t...)
}

func marshalTag(s reflect.Value) ([]m3u.Tag, error) {
	sym := register(s, false)
	// access all the fields that have m3u tags
	// all of these will be tags themselves
	tags := []m3u.Tag{}
	for _, label := range sym.names {
		val := s.Field(sym.field[label.name].index)
		if label.omitempty && val.IsZero() {
			continue
		}
		t := m3u.Tag{Name: label.name}
		tags = append(tags, t)
	}
	return tags, nil
}

func unmarshalTag(s reflect.Value, t ...m3u.Tag) error {
	sym := register(s, false)
	for _, t := range t {
		f, ok := sym.field[t.Name]
		if ok && f.set != nil {
			f.set(s.Elem().Field(f.index), t, "")
		}
	}
	return nil
}

func unmarshalAttr(s reflect.Value, t m3u.Tag) error {
	sym := register(s, true)
	for name, lut := range sym.field {
		lut.set(s.Field(lut.index), t, name)
	}
	return nil
}

type sfield struct {
	// index of this field in the parent struct
	index int

	// attr is true if this is an attribute, indicating that
	// the parent struct is a tag, and this field is one of
	// its key value pairs
	attr bool

	// set knows how to set the value to the contents
	// of the tag. The final argument is an option key-value
	// used for attribute names
	set func(reflect.Value, m3u.Tag, string)

	// TODO(as): implement: should product a tag from a
	// reflect.Value, for marshalling
	tag func(reflect.Value, *m3u.Tag)
}

// sym is a logical union that represents a cached struct or
// struct field
type sym struct {
	field map[string]sfield
	names []label
}

// symtab is the symbol table
// currently, this is not safe for concurrent use unless all symbols are
// loaded at init time and are immutable. the encoding/json package
// uses a sync.Map, but we don't need to do that because we have control
// over all the types we see in this package.
var symtab = map[reflect.Type]sym{}

// register registers the reflect.Value as a symbol exactly once
// and returns the result. it calls itself recursively on all applicable
// types recognized by the package
func register(v reflect.Value, attr bool) sym {
	t := v.Type()
	s, ok := symtab[t]
	if ok {
		return s
	}
	s = sym{
		field: map[string]sfield{},
	}
	switch t.Kind() {
	case reflect.Ptr:
		return register(v.Elem(), attr)
	case reflect.Slice:
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			label := parselabel(t.Field(i))
			if label == nil {
				continue
			}
			s.field[label.name] = sfield{index: i, set: compile(v.Field(i)), attr: attr}
			s.names = append(s.names, *label)
		}
	default:
		return sym{}
	}
	symtab[t] = s
	return s
}

type label struct {
	name      string
	omitempty bool
}

func parselabel(sf reflect.StructField) *label {
	v, ok := sf.Tag.Lookup("hls")
	if !ok {
		return nil
	}
	a := strings.Split(v, ",")
	l := label{name: a[0]}
	for _, extra := range a[1:] {
		if extra == "omitempty" {
			l.omitempty = true
		}
	}
	return &l
}

func compile(rf reflect.Value) func(reflect.Value, m3u.Tag, string) {
	switch rf.Interface().(type) {
	case bool:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			val := t.Value(key)
			if val != "NO" && val != "FALSE" {
				rf.SetBool(true)
			}
		}
	case float32, float64:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			val := t.Value(key)
			f, _ := strconv.ParseFloat(val, 64)
			rf.SetFloat(float64(f))
		}
	case uint8, uint16, uint32, uint64, uint, int8, int16, int32, int64, int:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			val := t.Value(key)
			i, _ := strconv.Atoi(val)
			rf.SetInt(int64(i))
		}
	case string:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			val := t.Value(key)
			rf.SetString(val)
		}
	case []string:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			val := t.Value(key)
			rf.Set(reflect.ValueOf(setSlice(val)))
		}
	case time.Duration:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			val := t.Value(key)
			d, _ := time.ParseDuration(val + "s")
			rf.Set(reflect.ValueOf(d))
		}
	case image.Point:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			val := t.Value(key)
			p := image.Point{}
			fmt.Sscanf(val, "%dx%d", &p.X, &p.Y)
			rf.Set(reflect.ValueOf(p))
		}
	}
	switch t := rf.Type(); t.Kind() {
	case reflect.Struct:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			unmarshalAttr(rf, t)
		}
	case reflect.Slice:
		elem := reflect.New(t.Elem()).Elem()
		register(elem, false)
		return func(slice reflect.Value, t m3u.Tag, key string) {
			unmarshalAttr(elem, t)
			slice.Set(reflect.Append(slice, elem))
		}
	}
	return nil
}

func setSlice(s string) interface{} {
	a := strings.Split(s, ",")
	return a
}
