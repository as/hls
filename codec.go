package hls

import (
	"reflect"

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
