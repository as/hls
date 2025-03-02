package m3u

import (
	"bufio"
	"io"
	"strings"
)

func Parse(r io.Reader) (t []Tag, err error) {
	return newlex(r).Parse()
}

type lex struct {
	br  *bufio.Reader
	tag Tag

	err error
	cur []byte
	b   byte
}

func newlex(r io.Reader) *lex {
	return &lex{
		br: bufio.NewReaderSize(r, 1024),
		// Reader: bufio.NewReader(io.MultiReader(r, strings.NewReader("\n"))),
	}
}

func New(r io.Reader) *lex {
	return newlex(r)
}

func (l *lex) Parse() (t []Tag, err error) {
	for l.lexTag() {
		t = append(t, l.tag)
	}
	if l.err == io.EOF {
		l.err = nil
	}
	return append(t, l.tag), l.err
}

func (s *lex) Reset(r io.Reader) {
	if s.br == nil {
		s.br = bufio.NewReaderSize(r, 1024)
	} else {
		s.br.Reset(r)
	}
	s.err = nil
	s.cur = s.cur[:0]
	s.tag = Tag{}
}

func (s *lex) lexTag() bool {
	s.whitespace()
	if !s.ignore('#') {
		return false
	}
	s.tag = Tag{}
	s.untilAny(":\n")
	s.tag.Name = s.token()
	s.lexAttr()
	for s.lexArg() {
	}
	return s.ok()
}

func (s *lex) lexAttr() bool {
	if !s.ignore(':') {
		return false
	}

	delims := "=,\n"
	for s.untilAny(delims) {
		f := Value{V: s.token()}
		if !s.lexAttrValue(&f) {
			s.tag.Arg = append(s.tag.Arg, f)
			// after we encounter the first keyless field, stop looking
			// for equal signs, since they might be part of the url
			// in EXTINF tags
			delims = ",\n"
		}
		if s.skip() != ',' {
			break
		}
		//s.whitespace()
	}
	return s.ok()
}

func (s *lex) lexAttrValue(key *Value) bool {
	if !s.ignore('=') {
		return false
	}
	f := Value{}
	if s.ignore('"') {
		s.untilAny(`"`)
		f.V = s.token()
		f.Quote = true
		s.ignore('"')
	} else {
		s.untilAny(",\n")
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
	if s.next() == '#' {
		s.backup()
		return false
	}
	s.untilAny("\n")
	s.tag.Line = append(s.tag.Line, s.token())
	s.skip()
	return s.ok()
}

func (s *lex) ok() bool {
	return s.err == nil
}

func (s *lex) next() (r byte) {
	if s.b, s.err = s.br.ReadByte(); s.ok() {
		s.cur = append(s.cur, s.b)
	}
	return s.b
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
	b := s.next()
	for _, v := range []byte(valid) {
		if v == b {
			return true
		}
	}
	s.backup()
	return false
}

func (s *lex) untilAny(delims string) bool {
	for lim := 2048; s.ok() && lim != 0; lim-- {
		b := s.next()
		for _, d := range []byte(delims) {
			if b == d {
				s.backup()
				return true
			}
		}
	}
	return false
}

func (s *lex) backup() bool {
	s.br.UnreadByte()
	if n := len(s.cur); n > 0 {
		s.cur = s.cur[:n-1]
	}
	return s.ok()
}

func (s *lex) skip() (r byte) {
	r = s.next()
	s.advance()
	return r
}

func (s *lex) token() string {
	t := string(s.cur)
	s.cur = s.cur[:0]
	return t
}

func (s *lex) advance() bool {
	s.cur = s.cur[:0]
	return true
}

func (s *lex) ignore(c byte) bool {
	if s.next() == c {
		s.cur = s.cur[:0]
		return true
	}
	s.backup()
	return false
}
