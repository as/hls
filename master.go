package hls

import (
	"fmt"
	"image"
	"io"
	"net/url"

	"github.com/as/hls/m3u"
)

// Master is a master playlist. It contains a list of streams (variants) and
// media information associated by group id. By convention, the master playlist is immutable.
type Master struct {
	M3U         bool         `hls:"EXTM3U" json:",omitempty"`
	Version     int          `hls:"EXT-X-VERSION" json:",omitempty"`
	Independent bool         `hls:"EXT-X-INDEPENDENT-SEGMENTS,omitempty" json:",omitempty"`
	Steering    Steering     `hls:"EXT-X-CONTENT-STEERING,omitempty" json:",omitempty"`
	Media       []MediaInfo  `hls:"EXT-X-MEDIA,aggr,omitempty" json:",omitempty"`
	Stream      []StreamInfo `hls:"EXT-X-STREAM-INF,aggr,omitempty" json:",omitempty"`
	IFrame      []StreamInfo `hls:"EXT-X-I-FRAME-STREAM-INF,aggr,omitempty" json:",omitempty"`

	URL string `json:",omitempty"`
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

// Encode encodes the master
func (m Master) Encode(w io.Writer) (err error) {
	tags, err := m.EncodeTag()
	for _, t := range tags {
		fmt.Fprintln(w, t)
	}
	return err
}

func (m Master) EncodeTag() (t []m3u.Tag, err error) {
	if t, err = marshalTag0(m); err != nil {
		return t, err
	}
	return t, err
}

// Len returns the number of variant streams
func (m *Master) Len() int {
	return len(m.Stream)
}

type Steering struct {
	URI     string `hls:"SERVER-URI,omitempty" json:",omitempty"`
	Pathway string `hls:"PATHWAY-ID,omitempty" json:",omitempty"`
}

type MediaInfo struct {
	Type       string   `hls:"TYPE,noquote,omitempty" json:",omitempty"`
	Group      string   `hls:"GROUP-ID,omitempty" json:",omitempty"`
	Name       string   `hls:"NAME,omitempty" json:",omitempty"`
	StableID   string   `hls:"STABLE-RENDITION-ID,omitempty" json:",omitempty"`
	Default    bool     `hls:"DEFAULT" json:",omitempty"`
	Autoselect bool     `hls:"AUTOSELECT" json:",omitempty"`
	Character  []string `hls:"CHARACTERISTICS" json:",omitempty"`
	Codecs     []string `hls:"CODECS,omitempty" json:",omitempty"`
	Lang       string   `hls:"LANGUAGE,omitempty" json:",omitempty"`
	Instream   string   `hls:"INSTREAM-ID,omitempty" json:",omitempty"`
	Bitdepth   int      `hls:"BIT-DEPTH,omitempty" json:",omitempty"`
	Samplerate int      `hls:"SAMPLE-RATE,omitempty" json:",omitempty"`
	Channels   string   `hls:"CHANNELS,omitempty" json:",omitempty"`
	URI        string   `hls:"URI,omitempty" json:",omitempty"`
}

type StreamInfo struct {
	URL string `hls:"$file" json:",omitempty"`

	Index        int         `hls:"PROGRAM-ID" json:",omitempty"`
	Framerate    float64     `hls:"FRAME-RATE,omitempty" json:",omitempty"`
	Bandwidth    int         `hls:"BANDWIDTH,omitempty" json:",omitempty"`
	BandwidthAvg int         `hls:"AVERAGE-BANDWIDTH,omitempty" json:",omitempty"`
	Codecs       []string    `hls:"CODECS,omitempty" json:",omitempty"`
	Resolution   image.Point `hls:"RESOLUTION" json:",omitempty"`
	VideoRange   string      `hls:"VIDEO-RANGE,noquote,omitempty" json:",omitempty"`
	HDCP         string      `hls:"HDCP-LEVEL,noquote,omitempty" json:",omitempty"`

	Audio    string `hls:"AUDIO,omitempty" json:",omitempty"`
	Video    string `hls:"VIDEO,omitempty" json:",omitempty"`
	Subtitle string `hls:"SUBTITLES,omitempty" json:",omitempty"`
	Pathway  string `hls:"PATHWAY-ID,omitempty" json:",omitempty"`

	// Caption is unquoted if the value is NONE, this absolute mess of a datatype
	// is handled explicitly in m3u/m3u.go:/CLOSED-CAPTIONS/
	Caption string `hls:"CLOSED-CAPTIONS,ambiguous,omitempty" json:",omitempty"`

	// URI is only set in IFrame stream infos
	URI string `hls:"URI,omitempty" json:",omitempty"`
}

// Path is Path
func (m Master) Path(parent string) string {
	return pathof(parent, m.URL)
}

// Path is Path
func (m MediaInfo) Path(parent string) string {
	return pathof(parent, m.URI)
}

// Location returns the stream URL relative to base. It conditionally
// applies the base URL in cases where the stream URL is a relative
// path. Base may be nil. This function never returns nil, but may
// return an empty URL. For error handling, process s.URLmanually.
func (s StreamInfo) Location(base *url.URL) *url.URL {
	return location(base, s.URL)
}

// Path is like Location, except its not a pain to use. It returns the path to
// the stream given an optional parent master path. If parent ends in a slash
// we assume parent is just the current working directory, otherwise the base
// name is stripped.
func (s StreamInfo) Path(parent string) string {
	if s.URI != "" {
		return pathof(parent, s.URI)
	}
	return pathof(parent, s.URL)
}

// Path is Path
func (s Steering) Path(parent string) string {
	return pathof(parent, s.URI)
}
