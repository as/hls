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

// symtab is the symbol table
// currently, this is not safe for concurrent use unless all symbols are
// loaded at init time and are immutable. the encoding/json package
// uses a sync.Map, but we don't need to do that because we have control
// over all the types we see in this package.
var symtab = map[reflect.Type]sym{}

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

	kid *sfield
}

// sym is a logical union that represents a cached struct or
// struct field
type sym struct {
	field map[string]sfield
	names []label
}

var extratag = map[string]func() reflect.Value{}

func RegisterTag(name string, value interface{}) {
	register(reflect.ValueOf(value), false)
	t := reflect.TypeOf(value)
	new := func() reflect.Value {
		return reflect.Indirect(reflect.New(t))
	}
	extratag[name] = new
}

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
		s := register(v.Elem(), attr)
		symtab[t] = s
		return s
	case reflect.Slice:
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			label := parselabel(t.Field(i))
			if label == nil {
				continue
			}
			//		fmt.Printf("label.name=%v\n", label.name)
			if label.embed {
				vv := v.Field(i)
				z := reflect.New(vv.Type().Elem())
				vv.Set(z)
				kid := register(vv, false)
				for _, label := range kid.names {
					s.names = append(s.names, label)
					p := kid.field[label.name]
					s.field[label.name] = sfield{index: i, kid: &p, attr: attr}
				}
			} else {
				s.field[label.name] = sfield{index: i, set: compileDec(v.Field(i)), attr: attr}
				s.names = append(s.names, *label)
			}
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
	embed     bool
	quote     bool
	aggr      bool
}

func parselabel(sf reflect.StructField) *label {
	v, ok := sf.Tag.Lookup("hls")
	if !ok {
		return nil
	}
	a := strings.Split(v, ",")
	l := label{name: a[0]}
	crc := 0
	for _, extra := range a[1:] {
		switch extra {
		case "omitempty":
			l.omitempty = true
		case "aggr":
			l.aggr = true
		case "embed":
			l.embed = true
		case "quote":
			crc++
			l.quote = true
		case "noquote":
			crc++
			l.quote = false
		}
	}
	if crc > 1 {
		panic(fmt.Sprintf("hls tag in struct field %s: quote/noquote specified more than once", sf.Name))
	}
	if crc == 0 {
		switch sf.Type.Kind() {
		case reflect.Slice:
			l.quote = true
		case reflect.String:
			l.quote = true
		default:
			l.quote = false
		}
	}
	return &l
}

func settag(rf reflect.Value, t *m3u.Tag, quote bool) {
	type tagsetter interface {
		settag(t *m3u.Tag)
	}
	w := m3u.Value{Quote: quote}
	switch val := rf.Interface().(type) {
	case m3u.Tag:
		*t = val
	case tagsetter:
		val.settag(t)
	case time.Duration:
		w.V = fmt.Sprint(val.Seconds())
	case time.Time:
		w.V = fmt.Sprint(val.Format(time.RFC3339Nano))
	case float32, float64:
		w.V = fmt.Sprint(val)
	case string:
		w.V = fmt.Sprint(val)
	case uint8, uint16, uint32, uint64, uint, int8, int16, int32, int64, int:
		w.V = fmt.Sprint(val)
	case image.Point:
		w.V = fmt.Sprintf("%dx%d", val.X, val.Y)
	case interface{}:
		switch val := rf.Type(); val.Kind() {
		case reflect.Struct:
			sym := register(rf, false)
			if t.Flag == nil {
				t.Flag = map[string]m3u.Value{}
			}
			for _, label := range sym.names {
				sf := sym.field[label.name]
				attr := tostring(rf.Field(sf.index), label.omitempty)
				if attr == "" {
					continue
				}
				if label.name == "$file" {
					t.Line = append(t.Line, attr)
				} else if strings.HasPrefix(label.name, "$") {
					t.Arg = append(t.Arg, m3u.Value{V: attr, Quote: false})
				} else {
					t.Keys = append(t.Keys, label.name)
					t.Flag[label.name] = m3u.Value{V: attr, Quote: label.quote}
				}
			}
			return
		default:
			return
		}
	}
	t.Arg = append(t.Arg, w)
}

func tostring(rf reflect.Value, omitempty bool) string {
	if omitempty && rf.IsZero() {
		return ""
	}
	switch t := rf.Interface().(type) {
	case bool:
		if t {
			return "YES"
		}
		return "NO"
	case []string:
		return strings.Join(t, ",")
	case image.Point:
		return fmt.Sprintf("%dx%d", t.X, t.Y)
	case time.Time:
		return fmt.Sprint(t.Format(time.RFC3339Nano))
	case time.Duration:
		return fmt.Sprint(t.Seconds())
	case interface{}:
		return fmt.Sprint(t)
	}
	return ""
}

// compileDec returns a func that can decode the m3u.Tag into the
// type represented by the input argument rf
func compileDec(rf reflect.Value) func(reflect.Value, m3u.Tag, string) {
	type tagdecoder interface {
		decodetag(t m3u.Tag)
	}
	if rf.CanAddr() {
		switch rf.Addr().Interface().(type) {
		case tagdecoder:
			return func(rf reflect.Value, t m3u.Tag, key string) {
				td, _ := rf.Addr().Interface().(tagdecoder)
				if td != nil {
					td.decodetag(t)
				}
			}
		}
	}

	switch rf.Interface().(type) {
	case m3u.Tag:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			rf.Set(reflect.ValueOf(t))
		}
	case bool:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			if key == "" {
				rf.SetBool(true)
			} else {
				val := t.Value(key)
				if val != "NO" && val != "FALSE" && val != "" {
					rf.SetBool(true)
				}
			}
		}
	case float32, float64:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			f, _ := strconv.ParseFloat(t.Value(key), 64)
			rf.SetFloat(float64(f))
		}
	case uint8, uint16, uint32, uint64, uint, int8, int16, int32, int64, int:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			i, _ := strconv.Atoi(t.Value(key))
			rf.SetInt(int64(i))
		}
	case string:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			rf.SetString(t.Value(key))
		}
	case []string:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			rf.Set(reflect.ValueOf(setSlice(t.Value(key))))
		}
	case time.Time:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			tm, _ := time.Parse(time.RFC3339Nano, t.Value(key))
			rf.Set(reflect.ValueOf(tm))
		}
	case time.Duration:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			d, _ := time.ParseDuration(t.Value(key) + "s")
			rf.Set(reflect.ValueOf(d))
		}
	case image.Point:
		return func(rf reflect.Value, t m3u.Tag, key string) {
			p := image.Point{}
			fmt.Sscanf(t.Value(key), "%dx%d", &p.X, &p.Y)
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
