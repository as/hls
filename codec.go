package hls

import (
	"reflect"
	"sort"

	"github.com/as/hls/m3u"
)

func marshalTag0(s interface{}) ([]m3u.Tag, error) {
	return marshalTag(reflect.ValueOf(s))
}

func unmarshalTag0(s interface{}, t ...m3u.Tag) error {
	unmarshalTag(reflect.ValueOf(s), t...)
	return nil
}

func marshalTag(s reflect.Value) ([]m3u.Tag, error) {
	sym := register(s, false)
	// access all the fields that have m3u tags
	// all of these will be tags themselves
	tags := []m3u.Tag{}
	for _, label := range sym.names {
		f := sym.field[label.name]
		val := s.Field(f.index)
		if label.omitempty && val.IsZero() {
			continue
		}
		if f.kid != nil {
			val = val.Elem().Field(f.kid.index)
			//			fmt.Printf("label=%s %d->%d got embed of type %T\n", label.name, f.index, f.kid.index, val.Interface())
			if label.omitempty && val.IsZero() {
				continue
			}
		}
		if label.name == "*" {
			extra, _ := val.Interface().(map[string]interface{})
			extratags := []m3u.Tag{}
			for k, v := range extra {
				if t, ok := v.(m3u.Tag); ok {
					extratags = append(extratags, t)
					continue
				}
				t := m3u.Tag{Name: k}
				settag(reflect.ValueOf(v), &t, false)
				extratags = append(extratags, t)
			}
			if len(extratags) > 0 {
				sort.Slice(extratags, func(i, j int) bool {
					return extratags[i].Name < extratags[j].Name
				})
			}
			tags = append(tags, extratags...)
		} else if label.aggr {
			for i := 0; i < val.Len(); i++ {
				t := m3u.Tag{Name: label.name}
				settag(val.Index(i), &t, label.quote)
				tags = append(tags, t)
			}
		} else {
			t := m3u.Tag{Name: label.name}
			settag(val, &t, label.quote)
			tags = append(tags, t)
		}
	}
	return tags, nil
}

func unmarshalTag(s reflect.Value, t ...m3u.Tag) reflect.Value {
	type extra interface {
		AddExtra(tag string, value interface{})
	}
	sym := register(s, false)
	for _, t := range t {
		f, ok := sym.field[t.Name]
		if ok && f.set != nil {
			f.set(s.Elem().Field(f.index), t, "")
		}
		if ok && f.kid != nil {
			ptr := s.Elem().Field(f.index)
			if ptr.IsZero() {
				z := reflect.New(ptr.Type().Elem())
				ptr.Set(z)
			}
			//fmt.Printf("set %#v field %d\n", ptr, f.kid.index)
			f.kid.set(ptr.Elem().Field(f.kid.index), t, "")
		}
		if !ok {
			file, ok := s.Interface().(extra)
			if ok {
				new := extratag[t.Name]
				if new != nil {
					tag := new()
					unmarshalAttr(tag, t)
					file.AddExtra(t.Name, tag.Interface())
				}
			}
		}
	}
	return s
}

func unmarshalAttr(s reflect.Value, t m3u.Tag) error {
	sym := register(s, true)
	for name, lut := range sym.field {
		lut.set(s.Field(lut.index), t, name)
	}
	return nil
}
