package m3u

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func Parse(r io.Reader) (t []Tag, err error) {
	return newlex(r).Parse()
}

type lex struct {
	line []byte
	i, j int
	sc   *bufio.Scanner
	tag  Tag
}

func newlex(r io.Reader) *lex {
	return &lex{
		sc: bufio.NewScanner(r),
	}
}

func New(r io.Reader) *lex {
	return newlex(r)
}

func (l *lex) Parse() (t []Tag, err error) {
	for l.sc.Scan() {
		l.line = l.sc.Bytes()
		l.i, l.j = 0, 0
		l.whitespace()
		switch l.peek() {
		case '#':
			if l.lexTag() {
				t = append(t, l.tag)
			}
		case 0:
		default:
			l.tag.Line = append(l.tag.Line, string(l.line[l.i:]))
			if len(t) > 0 {
				t[len(t)-1] = l.tag
			}
		}
	}
	err = l.sc.Err()
	if err == io.EOF {
		err = nil
	}
	return t, err
}

func (s *lex) lexTag() bool {
	if !s.ignore('#') {
		return false
	}
	s.tag = Tag{}
	s.until(':')
	s.tag.Name = s.token()
	s.lexAttr()
	return true
}

func (s *lex) lexAttr() bool {
	if !s.ignore(':') {
		return false
	}

	delims := ",="
	for s.untilAny(delims) || s.i < s.j {
		f := Value{V: s.token()}
		if !s.lexAttrValue(&f) {
			s.tag.Arg = append(s.tag.Arg, f)
			// after we encounter the first keyless field, stop looking
			// for equal signs, since they might be part of the url
			// in EXTINF tags
			delims = ","
		}
		if s.skip() != ',' {
			break
		}
		//s.whitespace()
	}
	return s.ok()
}

func (s *lex) debug(fm string, a ...any) {
	fmt.Fprintf(os.Stderr, fm, a...)
}

func (s *lex) lexAttrValue(key *Value) bool {
	if !s.ignore('=') {
		return false
	}
	f := Value{}
	if s.ignore('"') {
		s.until('"')
		f.V = s.token()
		f.Quote = true
		s.ignore('"')
	} else {
		s.until(',')
		f.V = s.token()
	}
	if strings.Trim(f.V, "\t=, ") == "" {
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

func (s *lex) ok() bool {
	return s.j <= len(s.line)
}

func (s *lex) peek() (r byte) {
	if s.j+1 < len(s.line) {
		r = s.line[s.j]
	}
	return
}

func (s *lex) next() (r byte) {
	s.j++
	if s.j > len(s.line) {
		return
	}
	return s.line[s.j-1]
}

func (s *lex) whitespace() bool {
	has := s.acceptRun(" \t")
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

func (s *lex) until(delim byte) bool {
	for s.j < len(s.line) {
		if s.line[s.j] == delim {
			return true
		}
		s.j++
	}
	return false
}

func (s *lex) untilAny(delims string) bool {
	if len(delims) == 1 {
		return s.until(delims[0])
	}
	for s.j < len(s.line) {
		for _, d := range []byte(delims) {
			if s.line[s.j] == d {
				return true
			}
		}
		s.j++
	}
	return false
}

func (s *lex) backup() bool {
	if s.j > s.i {
		s.j--
	}
	return s.ok()
}

func (s *lex) skip() (r byte) {
	r = s.next()
	s.advance()
	return r
}

func (s *lex) token() string {
	t := string(s.line[s.i:s.j])
	s.i = s.j
	return t
}

func (s *lex) advance() bool {
	s.i = s.j
	return s.j < len(s.line)
}

func (s *lex) ignore(c byte) bool {
	if s.next() == c {
		s.advance()
		return true
	}
	s.backup()
	return false
}
