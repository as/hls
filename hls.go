// Package hls implement an HLS codec for Master and Media files in m3u format
// At this time, the codec only supports decoding
package hls

import (
	"errors"
	"image"
	"io"
	"net/url"
	"strconv"
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

// Master is a master playlist. It contains a list of streams (variants) and
// media information associated by group id. By convention, the master playlist is immutable.
type Master struct {
	M3U         bool         `hls:"EXTM3U"`
	Version     int          `hls:"EXT-X-VERSION"`
	Independent bool         `hls:"EXT-X-INDEPENDENT-SEGMENTS"`
	Media       []MediaInfo  `hls:"EXT-X-MEDIA"`
	Stream      []StreamInfo `hls:"EXT-X-STREAM-INF"`
}

// Decode decodes the master playlist into m.
func (m *Master) Decode(r io.Reader) error {
	t, master, err := Decode(r)
	if err != nil {
		return err
	}
	if !master {
		return ErrType
	}
	return m.DecodeTag(t...)
}

func (m *Master) DecodeTag(t ...m3u.Tag) error {
	if err := unmarshalTag0(m, t...); err != nil {
		return err
	}
	if !m.M3U {
		return ErrHeader
	}
	if len(m.Stream) == 0 {
		return ErrEmpty
	}
	return nil
}

// Len returns the number of variant streams
func (m *Master) Len() int {
	return len(m.Stream)
}

// Media is a media playlist. It consists of a header and one or more files. A file
// is EXTINF and the content of any additional tags that apply to that EXTINF tag.
type Media struct {
	MediaHeader
	File []File `hls:""`
}

type MediaHeader struct {
	M3U           bool          `hls:"EXTM3U"`
	Version       int           `hls:"EXT-X-VERSION"`
	Independent   bool          `hls:"EXT-X-INDEPENDENT-SEGMENTS"`
	Type          string        `hls:"EXT-X-PLAYLIST-TYPE"`
	Target        time.Duration `hls:"EXT-X-TARGETDURATION"`
	Start         Start         `hls:"EXT-X-START"`
	Sequence      int           `hls:"EXT-X-MEDIA-SEQUENCE"`
	Discontinuity int           `hls:"EXT-X-DISCONTINUITY-SEQUENCE"`
	End           bool          `hls:"EXT-X-ENDLIST"`
}

// Runtime measures the cumulative duration of the given
// window of segments (files)
func Runtime(f ...File) (cumulative time.Duration) {
	for _, f := range f {
		cumulative += f.Inf.Duration
	}
	return
}

// Decode decodes the playlist in r and stores the
// result in m. It returns ErrEmpty if the playlist is
// well-formed, but contains no variant streams.
func (m *Media) Decode(r io.Reader) error {
	t, master, err := Decode(r)
	if err != nil {
		return err
	}
	if master {
		return ErrType
	}
	return m.DecodeTag(t...)
}

// DecodeTag decodes the list of tags as a media playlist
func (m *Media) DecodeTag(t ...m3u.Tag) error {
	if err := unmarshalTag0(&m.MediaHeader, t...); err != nil {
		return err
	}
	if !m.M3U {
		return ErrHeader
	}
	file := File{}
	i := 0
	for j := range t {
		if t[j].Name != "EXTINF" {
			continue
		}
		if err := unmarshalTag0(&file, t[i:j+1]...); err != nil {
			return err
		}
		i = j
		m.File = append(m.File, file)
		file = file.sticky()
	}

	if m.Len() == 0 {
		return ErrEmpty
	}
	return nil
}

// Current returns the most-recent segment in the stream
func (m *Media) Current() (f File) {
	if len(m.File) == 0 {
		return
	}
	return m.File[len(m.File)-1]
}

// Len returns the number of segments visibile to the playlist
func (m *Media) Len() int {
	return len(m.File)
}

func (m Media) EncodeTag() (t []m3u.Tag, err error) {
	if t, err = marshalTag0(m.MediaHeader); err != nil {
		return t, err
	}
	for _, v := range m.File {
		tmp, err := marshalTag0(v)
		t = append(t, tmp...)
		if err != nil {
			return t, err
		}
	}
	return t, err
}

type File struct {
	Discontinuous bool      `hls:"EXT-X-DISCONTINUITY,omitempty"`
	Time          time.Time `hls:"EXT-X-PROGRAM-DATE-TIME,omitempty"`
	Range         Range     `hls:"EXT-X-BYTERANGE,omitempty"`
	Map           Map       `hls:"EXT-X-MAP,omitempty"`
	Key           Key       `hls:"EXT-X-KEY,omitempty"`
	Inf           Inf       `hls:"EXTINF"`
}

// Location returns the media URL relative to base. It conditionally
// applies the base URL in cases where the media URL is a relative
// path. Base may be nil. This function never returns nil, but may
// return an empty URL. For error handling, process f.Inf.URL manually
func (f File) Location(base *url.URL) *url.URL {
	return location(base, f.Inf.URL)
}

// Init returns the initialization segment for fragmented mp4 files
// as an absolute url relative to base. If there is no initialization
// segment it returns an empty URL.
func (f File) Init(base *url.URL) *url.URL {
	u := f.Map.URI
	if u == "" {
		return &url.URL{}
	}
	return location(base, u)
}

// Duration returns the segment duration. An optional target can
// be provided as a fallback in case the duration was not set.
func (f File) Duration(target time.Duration) time.Duration {
	if f.Inf.Duration == 0 {
		return target
	}
	return f.Inf.Duration
}

// sticky returns a copy of f with only sticky field set
// a sticky field is a field that propagates across Inf blocks
//
func (f File) sticky() File {
	return File{
		Map: f.Map,
		Key: f.Key,
	}
}

type Range struct {
	V string `hls:""`
}

func (r Range) Value(n int) (at, size int, err error) {
	a := strings.Split(r.V, "@")
	if size, err = strconv.Atoi(a[0]); err != nil {
		return at, size, err
	}
	if len(a) == 1 {
		return n, size, nil
	}
	at, err = strconv.Atoi(a[1])
	return at, size, err
}

type Key struct {
}

type Map struct {
	URI string `hls:"URI"`
}

type Start struct {
	Offset  time.Duration `hls:"TIME-OFFSET"`
	Precise bool          `hls:"PRECISE"`
}

type Inf struct {
	Duration    time.Duration `hls:"$1"`
	Description string        `hls:"$2"`

	URL string `hls:"$file"`
}

type MediaInfo struct {
	Type       string `hls:"TYPE"`
	Group      string `hls:"GROUP-ID"`
	Name       string `hls:"NAME"`
	Default    bool   `hls:"DEFAULT"`
	Autoselect bool   `hls:"AUTOSELECT"`
	Forced     bool   `hls:"FORCED"`
	Lang       string `hls:"LANGUAGE"`
	URI        string `hls:"URI"`
}

type StreamInfo struct {
	URL string `hls:"$file"`

	Index        int         `hls:"PROGRAM-ID"`
	Framerate    float64     `hls:"FRAME-RATE"`
	Bandwidth    int         `hls:"BANDWIDTH"`
	BandwidthAvg int         `hls:"AVERAGE-BANDWIDTH"`
	Codecs       []string    `hls:"CODECS"`
	Resolution   image.Point `hls:"RESOLUTION"`
	VideoRange   string      `hls:"VIDEO-RANGE"`
	HDCP         string      `hls:"HDCP-LEVEL"`

	Audio    string `hls:"AUDIO"`
	Video    string `hls:"VIDEO"`
	Subtitle string `hls:"SUBTITLES"`
	Caption  string `hls:"CLOSED-CAPTIONS"`
}

// Location returns the stream URL relative to base. It conditionally
// applies the base URL in cases where the stream URL is a relative
// path. Base may be nil. This function never returns nil, but may
// return an empty URL. For error handling, process s.URLmanually.
func (s StreamInfo) Location(base *url.URL) *url.URL {
	return location(base, s.URL)
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
