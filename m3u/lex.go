package m3u

import (
	"bufio"
	"io"
	"strings"
)

func Parse(r io.Reader) (t []Tag, err error) {
	l := newlex(r)
	for l.lexTag() {
		t = append(t, l.tag)
	}
	if l.err == io.EOF {
		l.err = nil
	}
	return append(t, l.tag), l.err
}

type lex struct {
	*bufio.Reader
	tag Tag

	cur    string
	err    error
	b      byte
	cankey bool
}

func newlex(r io.Reader) *lex {
	return &lex{
		Reader: bufio.NewReader(io.MultiReader(r, strings.NewReader("\n"))),
	}
}

func (s *lex) lexTag() bool {
	s.whitespace()
	if !s.ignore("#") {
		return false
	}
	s.tag = Tag{}
	s.until(":\n")
	s.tag.Name = s.token()
	s.lexAttr()
	for s.lexArg() {
	}
	return s.ok()
}

func (s *lex) lexAttr() bool {
	if !s.ignore(":") {
		return false
	}

	delims := "=,\n"
	for s.until(delims) {
		f := Value{V: s.token()}
		if !s.lexAttrValue(&f) {
			s.tag.Arg = append(s.tag.Arg, f)
			// after we encounter the first keyless field, stop looking
			// for equal signs, since they might be part of the url
			// in EXTINF tags
			delims = ",\n"
		}
		if s.skip() != "," {
			break
		}
		//s.whitespace()
	}
	return s.ok()
}

func (s *lex) lexAttrValue(key *Value) bool {
	if !s.ignore("=") {
		return false
	}
	f := Value{}
	if s.ignore(`"`) {
		s.until(`"`)
		f.V = s.token()
		f.Quote = true
		s.ignore(`"`)
	} else {
		s.until(",\n")
		f.V = s.token()
	}
	if strings.Trim(f.V, "\n\r\t=, ") == "" {
		// this is an abuse of the m3u standard but some tags
		// have no key value pairs with trailing base64 padding
		// so bail out
		key.V += "=" + f.V
		return false
	}
	s.tag.Keys = append(s.tag.Keys, key.V)
	if s.tag.Flag == nil {
		s.tag.Flag = map[string]Value{}
	}
	s.tag.Flag[key.V] = f
	return true
}

func (s *lex) lexArg() bool {
	s.whitespace()
	if s.next() == "#" {
		s.backup()
		return false
	}
	s.until("\n")
	s.tag.Line = append(s.tag.Line, s.token())
	s.skip()
	return s.ok()
}

func (s *lex) ok() bool {
	return s.err == nil
}
func (s *lex) next() (r string) {
	if s.b, s.err = s.Reader.ReadByte(); s.ok() {
		r = string(s.b)
		s.cur += r
	}
	return r
}

func (s *lex) whitespace() bool {
	has := s.acceptRun(" \t\n\r")
	s.advance()
	return has
}

func (s *lex) acceptRun(valid string) bool {
	if !s.accept(valid) {
		return false
	}
	for s.accept(valid) {
	}
	return s.ok()
}

func (s *lex) accept(valid string) bool {
	if strings.ContainsAny(s.next(), valid) {
		return true
	}
	s.backup()
	return false
}
func (s *lex) until(delims string) bool {
	for lim := 4096; lim != 0; lim-- {
		if strings.ContainsAny(s.next(), delims) {
			s.backup()
			return true
		}
	}
	return false
}
func (s *lex) backup() bool {
	s.Reader.UnreadByte()
	if n := len(s.cur); n > 0 {
		s.cur = s.cur[:n-1]
	}
	return s.ok()
}
func (s *lex) skip() (r string) {
	defer s.advance()
	return s.next()
}
func (s *lex) token() string {
	defer s.advance()
	return s.cur
}
func (s *lex) advance() bool {
	s.cur = ""
	return true
}
func (s *lex) ignore(charset string) bool {
	return s.accept(charset) && s.advance()
}
