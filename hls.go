// Package hls implements an HLS codec for Master and Media files in m3u format
// At this time, the codec only supports decoding
package hls

import (
	"image"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/as/hls/m3u"
)

// Master is a master playlist. It contains a list of streams (variants) and
// media information associated by group id. By convention, the master playlist is immutable.
type Master struct {
	Version     int          `hls:"EXT-X-VERSION"`
	Independent bool         `hls:"EXT-X-INDEPENDENT-SEGMENTS"`
	Media       []MediaInfo  `hls:"EXT-X-MEDIA"`
	Stream      []StreamInfo `hls:"EXT-X-STREAM-INF"`
}

func (m *Master) DecodeHLS(r io.Reader) error {
	t, err := m3u.Parse(r)
	if err != nil {
		return err
	}
	return unmarshalTag0(m, t...)
}

// Media is a media playlist. It consists of a header and one or more files. A file
// is EXTINF and the content of any additional tags that apply to that EXTINF tag.
type Media struct {
	MediaHeader
	File []File `hls:""`
}

type MediaHeader struct {
	Version       int           `hls:"EXT-X-VERSION"`
	Independent   bool          `hls:"EXT-X-INDEPENDENT-SEGMENTS"`
	Type          string        `hls:"EXT-X-PLAYLIST-TYPE"`
	Duration      time.Duration `hls:"EXT-X-TARGETDURATION"`
	Start         Start         `hls:"EXT-X-START"`
	Sequence      int           `hls:"EXT-X-MEDIA-SEQUENCE"`
	Discontinuity int           `hls:"EXT-X-DISCONTINUITY-SEQUENCE"`
	End           bool          `hls:"EXT-X-ENDLIST"`
}

func (m Media) MarshalHLS() (t []m3u.Tag, err error) {
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

func (m *Media) DecodeHLS(r io.Reader) error {
	t, err := m3u.Parse(r)
	if err != nil {
		return err
	}
	if err := unmarshalTag0(&m.MediaHeader, t...); err != nil {
		return err
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
	return nil
}

type File struct {
	Discontinuous bool  `hls:"EXT-X-DISCONTINUITY,omitempty"`
	Range         Range `hls:"EXT-X-BYTERANGE,omitempty"`
	Map           Map   `hls:"EXT-X-MAP,omitempty"`
	Key           Key   `hls:"EXT-X-KEY,omitempty"`
	Inf           Inf   `hls:"EXTINF"`
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
	URI string `hls:"URI,attr"`
}

type Start struct {
	Offset  time.Duration `hls:"TIME-OFFSET"`
	Precise bool          `hls:"PRECISE"`
}

type Inf struct {
	Duration time.Duration `hls:"0"`
	URL      string        `hls:"1"`
}

type MediaInfo struct {
	Type       string `hls:"TYPE,attr"`
	Group      string `hls:"GROUP-ID,attr"`
	Name       string `hls:"NAME,attr"`
	Default    bool   `hls:"DEFAULT,attr"`
	Autoselect bool   `hls:"AUTOSELECT,attr"`
	Forced     bool   `hls:"FORCED,attr"`
	Lang       string `hls:"LANGUAGE,attr"`
	URI        string `hls:"URI,attr"`
}

type StreamInfo struct {
	URL string `hls:""`

	Index        int         `hls:"PROGRAM-ID,attr"`
	Framerate    float64     `hls:"FRAME-RATE,attr"`
	Bandwidth    int         `hls:"BANDWIDTH,attr"`
	BandwidthAvg int         `hls:"AVERAGE-BANDWIDTH,attr"`
	Codecs       []string    `hls:"CODECS,attr"`
	Resolution   image.Point `hls:"RESOLUTION,attr"`
	VideoRange   string      `hls:"VIDEO-RANGE,attr"`
	HDCP         string      `hls:"HDCP-LEVEL,attr"`
}
