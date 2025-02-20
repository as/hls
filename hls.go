// Package hls implement an HLS codec for Master and Media files in m3u format
// At this time, the codec only supports decoding
package hls

import (
	"errors"
	"fmt"
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
	M3U         bool         `hls:"EXTM3U" json:",omitempty"`
	Version     int          `hls:"EXT-X-VERSION" json:",omitempty"`
	Independent bool         `hls:"EXT-X-INDEPENDENT-SEGMENTS" json:",omitempty"`
	Media       []MediaInfo  `hls:"EXT-X-MEDIA" json:",omitempty"`
	Stream      []StreamInfo `hls:"EXT-X-STREAM-INF" json:",omitempty"`
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
	File []File `hls:"" json:",omitempty"`
}

type MediaHeader struct {
	M3U           bool          `hls:"EXTM3U" json:",omitempty"`
	Version       int           `hls:"EXT-X-VERSION" json:",omitempty"`
	Independent   bool          `hls:"EXT-X-INDEPENDENT-SEGMENTS,omitempty" json:",omitempty"`
	Type          string        `hls:"EXT-X-PLAYLIST-TYPE,omitempty" json:",omitempty"`
	Target        time.Duration `hls:"EXT-X-TARGETDURATION" json:",omitempty"`
	Start         Start         `hls:"EXT-X-START,omitempty" json:",omitempty"`
	Sequence      int           `hls:"EXT-X-MEDIA-SEQUENCE,omitempty" json:",omitempty"`
	Discontinuity int           `hls:"EXT-X-DISCONTINUITY-SEQUENCE,omitempty" json:",omitempty"`
	End           bool          `hls:"EXT-X-ENDLIST,omitempty" json:",omitempty"`
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

func (m Media) Trunc(dur time.Duration) Media {
	file := m.File
	for i := len(file) - 1; i >= 0; i-- {
		dur -= file[i].Duration(0)
		if dur < 0 {
			break
		}
		file = file[:i]
	}
	m.File = file
	return m
}

func (m Media) Encode(w io.Writer) (err error) {
	return writeplaylist(m, w)
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
	Comment       string    `hls:"#,omitempty" json:",omitempty"`
	Discontinuous bool      `hls:"EXT-X-DISCONTINUITY,omitempty" json:",omitempty"`
	Time          time.Time `hls:"EXT-X-PROGRAM-DATE-TIME,omitempty" json:",omitempty"`
	TimeMap       TimeMap   `hls:"EXT-X-TIMESTAMP-MAP,omitempty" json:",omitempty"`
	Range         Range     `hls:"EXT-X-BYTERANGE,omitempty" json:",omitempty"`
	Map           Map       `hls:"EXT-X-MAP,omitempty" json:",omitempty"`
	Key           Key       `hls:"EXT-X-KEY,omitempty" json:",omitempty"`

	// Asset and other AD-related insertion fields. Most of these can be used to signal
	// AD-insertion and many are redundant.
	Asset              m3u.Tag   `hls:"EXT-X-ASSET,omitempty" json:",omitempty"`
	PlacementOpp       bool      `hls:"EXT-X-PLACEMENT-OPPORTUNITY,omitempty" json:",omitempty"`
	CueOut             Cue       `hls:"EXT-X-CUE-OUT,omitempty" json:",omitempty"`
	CueCont            Cue       `hls:"EXT-X-CUE-OUT-CONT,omitempty" json:",omitempty"`
	CueIn              Cue       `hls:"EXT-X-CUE-IN,omitempty" json:",omitempty"`
	CueAdobe           CueAdobe  `hls:"EXT-X-CUE,omitempty" json:",omitempty"`
	SCTE35             SCTE35    `hls:"EXT-X-SCTE35,omitempty" json:",omitempty"`
	DateRange          DateRange `hls:"EXT-X-DATERANGE,omitempty" json:",omitempty"`
	SCTE35Splice       string    `hls:"EXT-X-SPLICEPOINT-SCTE35,omitempty" json:",omitempty"`
	SCTE35OatclsSplice string    `hls:"EXT-OATCLS-SCTE35,omitempty" json:",omitempty"`

	Extra map[string]interface{} `hls:"*,omitempty" json:",omitempty"`
	Inf   Inf                    `hls:"EXTINF" json:",omitempty"`
}

// IsAD returns true if the segment looks like an AD-break. This currently only handles
// the three standard EXT-X-CUE-OUT, EXT-X-CUE-OUT-CONT, and EXT-X-CUE-IN
// tags. Examine the SCTE35 fields manually to handle other formats
func (f *File) IsAD() bool {
	return f.CueOut.IsAD() || f.CueCont.IsAD() || f.CueOut.IsAD()
}

// Cue returns the value of the EXT-X-CUE-OUT, EXT-X-CUE-OUT-CONT,
// and EXT-X-CUE-IN tags. The Cue.Kind field is set to "in", "out", "cont" or
// the empty string if there is no queue.
//
// The SCTE35 field is set to the OatcltSplice or SCTE35Splice field in the File
// if not set in the Cue natively. This can be in binary, hex, or base64 format.
//
// Use: github.com/as/scte35.Parse(...) to decode the bitstream
//
// Example:
//
// if f.IsAD() { fmt.Println("cue is", f.Cue()) }
func (f *File) Cue() (c Cue) {
	defer func() {
		if !c.Set || c.SCTE35 != "" {
			return
		}
		for _, splice := range []string{c.SCTE35, f.SCTE35OatclsSplice, f.SCTE35Splice} {
			if splice != "" {
				c.SCTE35 = splice
				return
			}
		}
	}()
	c = f.CueOut
	if c.IsAD() {
		c.Set = true
		c.Kind = "out"
		return c
	}
	c = f.CueCont
	if c.IsAD() {
		c.Set = true
		c.Kind = "cont"
		return c
	}
	c = f.CueIn
	if c.IsAD() {
		c.Set = true
		c.Kind = "in"
		return c
	}
	return c
}

func (f *File) AddExtra(tag string, value interface{}) {
	if f.Extra == nil {
		f.Extra = map[string]interface{}{}
	}
	f.Extra[tag] = value
}

// Location returns the media URL relative to base. It conditionally
// applies the base URL in cases where the media URL is a relative
// path. Base may be nil. This function never returns nil, but may
// return an empty URL. For error handling, process f.Inf.URL manually
func (f File) Location(base *url.URL) (u *url.URL) {
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
func (f File) sticky() File {
	return File{
		Map: f.Map,
		Key: f.Key,
	}
}

type Range struct {
	V string `hls:"" json:",omitempty"`
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
	Method string `hls:"METHOD" json:",omitempty"`
	URI    string `hls:"URI" json:",omitempty"`
	IV     string `hls:"IV" json:",omitempty"`
}

type Map struct {
	URI string `hls:"URI" json:",omitempty"`
}

type Start struct {
	Offset  time.Duration `hls:"TIME-OFFSET" json:",omitempty"`
	Precise bool          `hls:"PRECISE" json:",omitempty"`
}

type TimeMap struct {
	MPEG  int       `hls:"MPEGTS" json:",omitempty"`
	Local time.Time `hls:"LOCAL" json:",omitempty"`
}

type Inf struct {
	Duration    time.Duration `hls:"$1" json:",omitempty"`
	Description string        `hls:"$2" json:",omitempty"`

	URL string `hls:"$file" json:",omitempty"`
}

func (h Inf) settag(t *m3u.Tag) {
	t.Arg = []m3u.Value{
		{V: fmt.Sprint(h.Duration.Seconds())},
		{V: fmt.Sprint(h.Description)},
	}
	if h.Description == "" {
		t.Arg = t.Arg[:1]
	}
	t.Line = append(t.Line, h.URL)
}

type MediaInfo struct {
	Type       string `hls:"TYPE" json:",omitempty"`
	Group      string `hls:"GROUP-ID" json:",omitempty"`
	Name       string `hls:"NAME" json:",omitempty"`
	Default    bool   `hls:"DEFAULT" json:",omitempty"`
	Autoselect bool   `hls:"AUTOSELECT" json:",omitempty"`
	Forced     bool   `hls:"FORCED" json:",omitempty"`
	Lang       string `hls:"LANGUAGE" json:",omitempty"`
	URI        string `hls:"URI" json:",omitempty"`
}

type StreamInfo struct {
	URL string `hls:"$file" json:",omitempty"`

	Index        int         `hls:"PROGRAM-ID" json:",omitempty"`
	Framerate    float64     `hls:"FRAME-RATE" json:",omitempty"`
	Bandwidth    int         `hls:"BANDWIDTH" json:",omitempty"`
	BandwidthAvg int         `hls:"AVERAGE-BANDWIDTH" json:",omitempty"`
	Codecs       []string    `hls:"CODECS" json:",omitempty"`
	Resolution   image.Point `hls:"RESOLUTION" json:",omitempty"`
	VideoRange   string      `hls:"VIDEO-RANGE" json:",omitempty"`
	HDCP         string      `hls:"HDCP-LEVEL" json:",omitempty"`

	Audio    string `hls:"AUDIO" json:",omitempty"`
	Video    string `hls:"VIDEO" json:",omitempty"`
	Subtitle string `hls:"SUBTITLES" json:",omitempty"`
	Caption  string `hls:"CLOSED-CAPTIONS" json:",omitempty"`
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
