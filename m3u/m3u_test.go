package m3u

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	_ "embed"
)

func TestLexBase64(t *testing.T) {
	tag, err := Parse(strings.NewReader(`#EXT-X-CUE-OUT-CONT:DURATION="120",ELAPSEDTIME=6,SCTE35=/DAlAAAAAyiYAP/wFAUAAIMRf+/+0bb+Av4AUmNiAAEBAQAAM5uUog==`))
	if err != nil {
		t.Fatal(err)
	}
	want := Tag{Name: "EXT-X-CUE-OUT-CONT", Keys: []string{"DURATION", "ELAPSEDTIME", "SCTE35"},
		Flag: map[string]Value{
			"DURATION":    {V: "120", Quote: true},
			"ELAPSEDTIME": {V: "6"},
			"SCTE35":      {V: "/DAlAAAAAyiYAP/wFAUAAIMRf+/+0bb+Av4AUmNiAAEBAQAAM5uUog=="},
		},
	}
	if !reflect.DeepEqual(want, tag[0]) {
		t.Fatalf("mismatch:\n\t\thave: %#v\n\t\twant: %#v", tag[0], want)
	}
}

func TestLexZeroLengthArg(t *testing.T) {
	tag, err := Parse(strings.NewReader("#EXTINF:10.0,\nfile"))
	if err != nil {
		t.Fatal(err)
	}
	want := Tag{Name: "EXTINF", Arg: []Value{{V: "10.0"}}, Line: []string{"file"}}
	if !reflect.DeepEqual(want, tag[0]) {
		t.Fatalf("mismatch:\n\t\thave: %#v\n\t\twant: %#v", tag[0], want)
	}
}

func TestLexZeroLengthArg2(t *testing.T) {
	tag, err := Parse(strings.NewReader("#EXTINF:10.0\nfile"))
	if err != nil {
		t.Fatal(err)
	}
	want := Tag{Name: "EXTINF", Arg: []Value{{V: "10.0"}}, Line: []string{"file"}}
	if !reflect.DeepEqual(want, tag[0]) {
		t.Fatalf("mismatch:\n\t\thave: %#v\n\t\twant: %#v", tag[0], want)
	}
}

func TestLexHeader(t *testing.T) {
	tag, err := Parse(strings.NewReader("#EXTM3U"))
	if err != nil {
		t.Fatal(err)
	}
	want := Tag{Name: "EXTM3U"}
	if !reflect.DeepEqual(want, tag[0]) {
		t.Fatalf("mismatch:\n\t\thave: %#v\n\t\twant: %#v", tag[0], want)
	}
}

func TestParse(t *testing.T) {
	var raw = `
#EXTM3U
#ABC:a=A,b=B,c=C,list="a,b,c",arg0,arg1,arg2
line0
line1
#DEF:10.0,desc
file0
#GHI:11.0,
file1
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=443680,RESOLUTION=400x224,CODECS="avc1.42e00d,mp4a.40.2"
https://02.m3u8
`
	tag, err := Parse(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	want := []Tag{
		{Name: "EXTM3U"},
		{
			Name: "ABC",
			Keys: []string{"a", "b", "c", "list"},
			Flag: map[string]Value{
				"a":    {V: "A"},
				"b":    {V: "B"},
				"c":    {V: "C"},
				"list": {V: "a,b,c", Quote: true},
			},
			Arg:  []Value{{V: "arg0"}, {V: "arg1"}, {V: "arg2"}},
			Line: []string{"line0", "line1"},
		},
		{
			Name: "DEF",
			Arg:  []Value{{V: "10.0"}, {V: "desc"}},
			Line: []string{"file0"},
		},
		{
			Name: "GHI",
			Arg:  []Value{{V: "11.0"}},
			Line: []string{"file1"},
		},
		{
			Name: "EXT-X-STREAM-INF",
			Keys: []string{"PROGRAM-ID", "BANDWIDTH", "RESOLUTION", "CODECS"},
			Flag: map[string]Value{
				"PROGRAM-ID": {V: "1"},
				"BANDWIDTH":  {V: "648224"},
				"RESOLUTION": {V: "640x360"},
				"CODECS":     {V: "avc1.4d401e,mp4a.40.2", Quote: true},
			},
			Line: []string{"https://01.m3u8"},
		},
		{
			Name: "EXT-X-STREAM-INF",
			Keys: []string{"PROGRAM-ID", "BANDWIDTH", "RESOLUTION", "CODECS"},
			Flag: map[string]Value{
				"PROGRAM-ID": {V: "1"},
				"BANDWIDTH":  {V: "443680"},
				"RESOLUTION": {V: "400x224"},
				"CODECS":     {V: "avc1.42e00d,mp4a.40.2", Quote: true},
			},
			Line: []string{"https://02.m3u8"},
		},
	}
	if !reflect.DeepEqual(tag, want) {
		t.Fatalf("mismatch:\n\t\thave: %#v\n\t\twant: %#v", tag, want)
	}
}

func BenchmarkParseEmpty(b *testing.B) { bench(b, "") }
func BenchmarkParseOne(b *testing.B)   { bench(b, "#EXTM3U") }
func BenchmarkParseFull(b *testing.B)  { bench(b, full) }
func BenchmarkParseJumbo(b *testing.B) { bench(b, jumbo) }

var jumbo = `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-TARGETDURATION:10
`

func init() {
	for i := 0; i < 8640; i++ {
		jumbo += fmt.Sprintf("#EXTINF:10.0000000000000,\nhttp://example.com/playout/video/segment%d.ts\n", i)
	}
}

func BenchmarkParse(b *testing.B) {
	bench(b, raw)
}

func bench(b *testing.B, input string) {
	b.Helper()
	b.SetBytes(int64(len(input)))
	r := strings.NewReader(input)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		r.Seek(0, 0)
		lex := New(r)
		lex.Parse()
	}
}

var raw = `
#EXTM3U
#ABC:a=A,b=B,c=C,list="a,b,c",arg0,arg1,arg2
line0
line1
#DEF:10.0,desc
file0
#GHI:11.0,
file1
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=443680,RESOLUTION=400x224,CODECS="avc1.42e00d,mp4a.40.2"
https://02.m3u8
`
var full = `
#EXTM3U
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=648224,RESOLUTION=640x360,CODECS="avc1.4d401e,mp4a.40.2"
https://01.m3u8
`
