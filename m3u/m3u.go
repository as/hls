package m3u

import "fmt"

type Tag struct {
	Name string
	Flag map[string]Value
	Keys []string
	Arg  []Value
	Line []string
}

func (t Tag) Value(key string) string {
	n := len(t.Arg)
	switch key {
	case "", "0":
		n = 0
	case "1":
		n = 1
	}
	if n < len(t.Arg) {
		return t.Arg[n].V
	}
	if key == "" && len(t.Line) > 0 {
		return t.Line[0]
	}

	s, _ := t.Flag[key]
	return s.V

}

func (t Tag) String() string {
	s := "#" + t.Name
	if len(t.Keys)+len(t.Arg) == 0 {
		return s
	}
	sep := ":"
	for _, v := range t.Arg {
		s += sep + v.String()
		sep = ","
	}
	for _, k := range t.Keys {
		s += sep + fmt.Sprintf("%s=%s", k, t.Flag[k])
		sep = ","
	}
	sep = "\n"
	for _, f := range t.Line {
		s += sep + f
	}
	if len(s) > 0 && s[len(s)-1] != '\n' {
		s += "\n"
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
