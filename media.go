package hls

import (
	"io"
	"time"

	"github.com/as/hls/m3u"
)

// Media is a media playlist. It consists of a header and one or more files. A file
// is EXTINF and the content of any additional tags that apply to that EXTINF tag.
type Media struct {
	MediaHeader
	File []File `hls:"" json:",omitempty"`

	URL string `json:",omitempty"`
}

type MediaHeader struct {
	M3U           bool          `hls:"EXTM3U" json:",omitempty"`
	Version       int           `hls:"EXT-X-VERSION" json:",omitempty"`
	Independent   bool          `hls:"EXT-X-INDEPENDENT-SEGMENTS,omitempty" json:",omitempty"`
	Type          string        `hls:"EXT-X-PLAYLIST-TYPE,noquote,omitempty" json:",omitempty"`
	Target        time.Duration `hls:"EXT-X-TARGETDURATION,omitempty" json:",omitempty"`
	Start         Start         `hls:"EXT-X-START,omitempty" json:",omitempty"`
	Sequence      int           `hls:"EXT-X-MEDIA-SEQUENCE,omitempty" json:",omitempty"`
	Discontinuity int           `hls:"EXT-X-DISCONTINUITY-SEQUENCE,omitempty" json:",omitempty"`
	End           bool          `hls:"EXT-X-ENDLIST,omitempty" json:",omitempty"`
}

// Path is Path
func (m Media) Path(parent string) string {
	return pathof(parent, m.URL)
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

func (m Media) Encode(w io.Writer) (err error) {
	return writeplaylist(m, w)
}

func (m Media) EncodeTag() (t []m3u.Tag, err error) {
	if t, err = marshalTag0(m.MediaHeader); err != nil {
		return t, err
	}
	var trailer []m3u.Tag
	if len(t) > 0 && t[len(t)-1].Name == "EXT-X-ENDLIST" {
		trailer = append(trailer, t[len(t)-1])
		t = t[:len(t)-1]
	}
	for _, v := range m.File {
		tmp, err := marshalTag0(v)
		t = append(t, tmp...)
		if err != nil {
			return t, err
		}
	}
	return append(t, trailer...), err
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
