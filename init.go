package hls

import "strings"

func init() {
	m0 := Master{}
	m0.Decode(strings.NewReader(sampleMaster))
	m1 := Media{}
	m1.Decode(strings.NewReader(sampleMedia))
}

var sampleMedia = `
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
var sampleMaster = `
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
