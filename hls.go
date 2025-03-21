// Package hls implement an HLS codec for Master and Media files in m3u format
// At this time, the codec only supports decoding
package hls

import (
	"errors"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/as/hls/m3u"
)

// Media playlist types
const (
	Vod   = "VOD"   // immutable
	Event = "EVENT" // append-only
	Live  = ""      // sliding-window
)

var (
	ErrHeader = errors.New("hls: no m3u8 tag")
	ErrEmpty  = errors.New("hls: empty playlist")
	ErrType   = errors.New("hls: playlist type mismatch")
)

// Decode reads an HLS playlist from the reader and tokenizes
// it into a list of tags. Master is true if and only if the input looks
// like a master playlist.
func Decode(r io.Reader) (t []m3u.Tag, master bool, err error) {
	t, err = m3u.Parse(r)
	for _, v := range t {
		switch v.Name {
		case "EXT-X-MEDIA":
			fallthrough
		case "EXT-X-STREAM-INF":
			fallthrough
		case "EXT-X-I-FRAME-STREAM-INF":
			return t, true, err // master
		case "EXTINF":
			return t, false, err // media
		}
	}
	// may be empty live media
	return t, false, err
}

// Runtime measures the cumulative duration of the given
// window of segments (files)
func Runtime(f ...File) (cumulative time.Duration) {
	for _, f := range f {
		cumulative += f.Inf.Duration
	}
	return
}

func location(base *url.URL, ref string) *url.URL {
	if base == nil {
		base = &url.URL{}
	}
	u, err := url.Parse(ref)
	if err != nil {
		return u
	}
	return base.ResolveReference(u)
}

// pathof is like location, except its not a pain to use. it returns the path to
// the file given an optional parent  path. If parent ends in a slash
// we assume parent is just the current working directory, otherwise the base
// name is stripped.
func pathof(parent string, self string) string {
	if strings.HasPrefix(self, "http://") {
		return self
	}
	base, err := url.Parse(parent)
	if base == nil {
		base = &url.URL{}
	}
	u, err := url.Parse(self)
	if err != nil {
		return self
	}
	return base.ResolveReference(u).String()
}
