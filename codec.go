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
		val := s.Field(sym.field[label.name].index)
		if label.omitempty && val.IsZero() {
			continue
		}
		if label.name == "*" {
			extra, _ := val.Interface().(map[string]interface{})
			extratags := []m3u.Tag{}
			for k, v := range extra {
				if t, ok := v.(m3u.Tag); ok{
					extratags = append(extratags, t)
					continue
				}
				t := m3u.Tag{Name: k}
				settag(reflect.ValueOf(v), &t)
				extratags = append(extratags, t)
			}
			if len(extratags) > 0 {
				sort.Slice(extratags, func(i, j int) bool {
					return extratags[i].Name < extratags[j].Name
				})
			}
			tags = append(tags, extratags...)
		} else {
			t := m3u.Tag{Name: label.name}
			settag(val, &t)
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
