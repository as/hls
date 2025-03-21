package hls

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/as/hls/m3u"
)

// File is an HLS segment in a media playlist and all of its associated tags
type File struct {
	Comment       string    `hls:"#,omitempty" json:",omitempty"`
	Discontinuous bool      `hls:"EXT-X-DISCONTINUITY,omitempty" json:",omitempty"`
	Time          time.Time `hls:"EXT-X-PROGRAM-DATE-TIME,omitempty" json:",omitempty"`
	TimeMap       TimeMap   `hls:"EXT-X-TIMESTAMP-MAP,omitempty" json:",omitempty"`
	Range         Range     `hls:"EXT-X-BYTERANGE,omitempty" json:",omitempty"`
	Map           Map       `hls:"EXT-X-MAP,omitempty" json:",omitempty"`
	Key           Key       `hls:"EXT-X-KEY,omitempty" json:",omitempty"`

	// Asset and other AD-related insertion fields. Most of these can be used to signal
	// AD-insertion and many are redundant. The decoder only initializes [AD] if any of
	// its tags are detected during decoding.
	Asset        m3u.Tag `hls:"EXT-X-ASSET,omitempty" json:",omitempty"`
	PlacementOpp bool    `hls:"EXT-X-PLACEMENT-OPPORTUNITY,omitempty" json:",omitempty"`
	AD           *AD     `hls:",embed,omitempty" json:",omitempty"`

	Extra map[string]interface{} `hls:"*,omitempty" json:",omitempty"`
	Inf   Inf                    `hls:"EXTINF" json:",omitempty"`
}

// IsAD returns true if the segment looks like an AD-break. This currently only handles
// the three standard EXT-X-CUE-OUT, EXT-X-CUE-OUT-CONT, and EXT-X-CUE-IN
// tags. Examine the SCTE35 fields manually to handle other formats
func (f *File) IsAD() bool {
	return f.AD.IsAD()
}

// Location returns the media URL relative to base. It conditionally
// applies the base URL in cases where the media URL is a relative
// path. Base may be nil. This function never returns nil, but may
// return an empty URL. For error handling, process f.Inf.URL manually
//
// NOTE: Don't use this, use File.Path instead
func (f File) Location(base *url.URL) (u *url.URL) {
	return location(base, f.Inf.URL)
}

func (f *File) Path(parent string) string {
	return pathof(parent, f.Inf.URL)
}

// Init returns the initialization segment for fragmented mp4 files
// as an absolute url relative to base. If there is no initialization
// segment it returns an empty URL.
//
// NOTE: Don't use this, use File.Map.Path instead
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

func (f *File) AddExtra(tag string, value interface{}) {
	if f.Extra == nil {
		f.Extra = map[string]interface{}{}
	}
	f.Extra[tag] = value
}

type Key struct {
	Method   string `hls:"METHOD,noquote" json:",omitempty"`
	URI      string `hls:"URI,omitempty" json:",omitempty"`
	IV       string `hls:"IV,omitempty" json:",omitempty"`
	Format   string `hls:"KEYFORMAT,omitempty" json:",omitempty"`
	Versions string `hls:"KEYFORMATVERSIONS,omitempty" json:",omitempty"`
}

func (m *Key) Path(parent string) string {
	return pathof(parent, m.URI)
}

type Map struct {
	URI       string `hls:"URI,omitempty" json:",omitempty"`
	Byterange string `hls:"BYTERANGE,omitempty" json:",omitempty"`
}

func (m *Map) Path(parent string) string {
	return pathof(parent, m.URI)
}

type Start struct {
	Offset  time.Duration `hls:"TIME-OFFSET" json:",omitempty"`
	Precise bool          `hls:"PRECISE,omitempty" json:",omitempty"`
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
