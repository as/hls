package m3u

import (
	"fmt"
	"strings"
)

type Tag struct {
	Name string
	Flag map[string]Value
	Keys []string
	Arg  []Value
	Line []string
}

func (t Tag) Value(key string) (val string) {
	n := len(t.Arg)
	switch key {
	case "$file":
		// $file is the first line after the tag, associated data
		if len(t.Line) > 0 {
			return t.Line[0]
		}
	case "":
		n = 0 // key empty, tags value is its first argument
	case "$1":
		n = 0 // explicit args $1-$4
	case "$2":
		n = 1
	case "$3":
		n = 2
	case "$4":
		n = 3
	case "$5":
		n = 4
	}
	if n < len(t.Arg) {
		return t.Arg[n].V
	}

	// key value lookup for key=value args
	s, _ := t.Flag[key]
	return s.V

}

func (t Tag) String() string {
	s := "#" + t.Name
	if len(t.Keys)+len(t.Arg) == 0 {
		return s
	}
	sep := ":"
	if t.Name == "#"{
		sep = ""
	}
	for _, v := range t.Arg {
		val := v.String()
		if val == ""{
			continue
		}
		s += sep + val
		sep = ","
	}
	for _, k := range t.Keys {
		s += sep + fmt.Sprintf("%s=%s", k, t.Flag[k])
		sep = ","
	}
	s = strings.TrimSuffix(s, ",")
	sep = "\n"
	for _, f := range t.Line {
		s += sep + f
	}
	if len(s) > 0 && s[len(s)-1] != '\n' {
		//s += "\n"
	}
	return s
}

type Value struct {
	V     string
	Quote bool
	Wrap  bool
}

func (v Value) String() string {
	if v.Quote {
		return fmt.Sprintf("%q", v.V)
	}
	if v.Wrap {
		return "\n" + v.V
	}
	return v.V
}
