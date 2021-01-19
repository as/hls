package hls

import (
	"image"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDecodeMaster(t *testing.T) {
	want := Master{
		M3U:         true,
		Version:     3,
		Independent: true,
		Stream: []StreamInfo{
			{URL: "m1.m3u8", Bandwidth: 1111, BandwidthAvg: 1000, Resolution: image.Pt(1, 1), Codecs: []string{"avc1.4D401F", "mp4a.40.2"}, Framerate: 29.97},
			{URL: "m2.m3u8", Bandwidth: 2222, BandwidthAvg: 2000, Resolution: image.Pt(2, 2), Codecs: []string{"avc1.4D401F", "mp4a.40.2"}, Framerate: 29.97},
			{URL: "m3.m3u8", Bandwidth: 3333, BandwidthAvg: 3000, Resolution: image.Pt(3, 3), Codecs: []string{"avc1.4D401F", "mp4a.40.2"}, Framerate: 29.97},
			{URL: "m4.m3u8", Bandwidth: 4444, BandwidthAvg: 4000, Resolution: image.Pt(4, 4), Codecs: []string{"avc1.4D401E", "mp4a.40.2"}, Framerate: 29.97},
			{URL: "m5.m3u8", Bandwidth: 5555, BandwidthAvg: 5000, Resolution: image.Pt(5, 5), Codecs: []string{"avc1.4D401E", "mp4a.40.2"}, Framerate: 29.97},
			{URL: "m6.m3u8", Bandwidth: 6666, BandwidthAvg: 6000, Resolution: image.Pt(6, 6), Codecs: []string{"avc1.4D400D", "mp4a.40.2"}, Framerate: 29.97},
		},
	}

	m := Master{}
	m.DecodeHLS(strings.NewReader(sampleMaster)) // init.go:/sampleMaster/
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("mismatch:\n\t\thave: %+v\n\t\twant: %+v", m, want)
	}
}

func TestDecodeMedia(t *testing.T) {
	tm, _ := time.Parse("2006-01-02T15:04:05.000Z", "2021-01-11T07:59:41.005Z")
	want := Media{
		MediaHeader: MediaHeader{
			M3U:           true,
			Version:       3,
			Independent:   true,
			Type:          "EVENT",
			Target:        10 * time.Second,
			Sequence:      1,
			Discontinuity: 2,
			Start:         Start{Offset: 25 * time.Second, Precise: true},
			End:           true,
		},
		File: []File{
			{Inf: Inf{10 * time.Second, "", "ad0.ts"}},
			{Inf: Inf{8 * time.Second, "", "ad1.ts?m=142"}},
			{Inf: Inf{10 * time.Second, "", "movieA.ts"}, Discontinuous: true, Time: tm},
			{Inf: Inf{10 * time.Second, "", "movieB.ts"}},
		},
	}

	m := Media{}
	m.DecodeHLS(strings.NewReader(sampleMedia)) // init.go:/sampleMedia/
	if m.Version != 3 {
		t.Fatalf("version: %v", m.Version)
	}
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("mismatch:\n\t\thave: %#v\n\t\twant: %#v", m, want)
	}
}

func BenchmarkDecodeMaster(b *testing.B) {
	benchDecode(b, &Master{}, strings.NewReader(sampleMaster))
}
func BenchmarkDecodeMedia(b *testing.B) {
	benchDecode(b, &Media{}, strings.NewReader(sampleMedia))
}

func benchDecode(b *testing.B, dst interface{ DecodeHLS(io.Reader) error }, src io.ReadSeeker) {
	for n := 0; n < b.N; n++ {
		src.Seek(0, 0)
		dst.DecodeHLS(src)
	}
}
