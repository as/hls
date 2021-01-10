package m3u

import (
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	var raw = `
#EXTM3U
#ABC:a=A,b=B,c=C,list="a,b,c",arg0,arg1,arg2
line0
line1
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
			Line: []string{"https://02.m3u8", "\n"},
		},
	}
	if !reflect.DeepEqual(tag, want) {
		t.Fatalf("mismatch:\n\t\thave: %#v\n\t\twant: %#v", tag, want)
	}
}
