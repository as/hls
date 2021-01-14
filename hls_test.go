package hls

import (
	"image"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDecodeMaster(t *testing.T) {
	sample := `
	#EXTM3U
	#EXT-X-VERSION:3
	#EXT-X-INDEPENDENT-SEGMENTS
	#EXT-X-STREAM-INF:BANDWIDTH=1111,AVERAGE-BANDWIDTH=1000,RESOLUTION=1x1,FRAME-RATE=29.970,CODECS="avc1.4D401F,mp4a.40.2"
	m1.m3u8
	#EXT-X-STREAM-INF:BANDWIDTH=2222,AVERAGE-BANDWIDTH=2000,RESOLUTION=2x2,FRAME-RATE=29.970,CODECS="avc1.4D401F,mp4a.40.2"
	m2.m3u8
	#EXT-X-STREAM-INF:BANDWIDTH=3333,AVERAGE-BANDWIDTH=3000,RESOLUTION=3x3,FRAME-RATE=29.970,CODECS="avc1.4D401F,mp4a.40.2"
	m3.m3u8
	#EXT-X-STREAM-INF:BANDWIDTH=4444,AVERAGE-BANDWIDTH=4000,RESOLUTION=4x4,FRAME-RATE=29.970,CODECS="avc1.4D401E,mp4a.40.2"
	m4.m3u8
	#EXT-X-STREAM-INF:BANDWIDTH=5555,AVERAGE-BANDWIDTH=5000,RESOLUTION=5x5,FRAME-RATE=29.970,CODECS="avc1.4D401E,mp4a.40.2"
	m5.m3u8
	#EXT-X-STREAM-INF:BANDWIDTH=6666,AVERAGE-BANDWIDTH=6000,RESOLUTION=6x6,FRAME-RATE=29.970,CODECS="avc1.4D400D,mp4a.40.2"
	m6.m3u8
	`
	want := Master{
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
	m.DecodeHLS(strings.NewReader(sample))
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("mismatch:\n\t\thave: %+v\n\t\twant: %+v", m, want)
	}
}

func TestDecodeMedia(t *testing.T) {
	sample := `
	#EXTM3U
	#EXT-X-VERSION:3
	#EXT-X-INDEPENDENT-SEGMENTS
	#EXT-X-PLAYLIST-TYPE:EVENT
	#EXT-X-START:TIME-OFFSET=25,PRECISE=YES
	#EXT-X-TARGETDURATION:10
	#EXT-X-MEDIA-SEQUENCE:1
	#EXT-X-DISCONTINUITY-SEQUENCE:2
	#EXTINF:10.0,
	ad0.ts
	#EXTINF:8.0,
	ad1.ts?m=142
	#EXT-X-DISCONTINUITY
	#EXT-X-PROGRAM-DATE-TIME:2021-01-11T07:59:41.005Z
	#EXTINF:10.0,
	movieA.ts
	#EXTINF:10.0,
	movieB.ts
	#EXT-X-ENDLIST
	`
	tm, _ := time.Parse("2006-01-02T15:04:05.000Z", "2021-01-11T07:59:41.005Z")
	want := Media{
		MediaHeader: MediaHeader{
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
	m.DecodeHLS(strings.NewReader(sample))
	if m.Version != 3 {
		t.Fatalf("version: %v", m.Version)
	}
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("mismatch:\n\t\thave: %#v\n\t\twant: %#v", m, want)
	}
}
