package hls

import (
	"fmt"
	"time"

	"github.com/as/hls/m3u"
)

// EXT-X-SPLICEPOINT-SCTE35 and EXT-OATCLS-SCTE35 are also supported
// and are contained in hls.File as base64-encoded strings. They are in binary splice
// insert format.
//
// For reference, Google DAI supports the following binary messages
// under the EXT-OATCLS-SCTE35 tag:
//
// SCTE35 Binary Time Signal: Break Start/End (34/35)
// SCTE35 Binary Time Signal: Provider Ad Start/End (48/49)
// SCTE35 Binary Time Signal: Provider Placement Opportunity (52/53)
// SCTE35 Binary Splice Insert
//
// As well as these tagged HLS messages:
//
// EXT-X-CUE-IN (Cue)
// EXT-X-CUE-OUT (Cue)
// EXT-X-CUE (CueAdobe) [Adobe Prime Time]
// EXT-X-DATERANGE (Official HLS Standard)

type SCTE35 struct {
	ID       string        `hls:"ID,omitempty" json:",omitempty"`
	Cue      string        `hls:"CUE,omitempty" json:",omitempty"`
	Duration time.Duration `hls:"DURATION,omitempty" json:",omitempty"`
	Elapsed  time.Duration `hls:"ELAPSED,omitempty" json:",omitempty"`
	Time     time.Duration `hls:"TIME,omitempty" json:",omitempty"`
	Type     int           `hls:"TYPE,omitempty" json:",omitempty"`
	UPID     string        `hls:"UPID,omitempty" json:",omitempty"`
	Blackout string        `hls:"BLACKOUT,omitempty" json:",omitempty"`
	CueIn    string        `hls:"CUE-IN,omitempty" json:",omitempty"`
	CueOut   string        `hls:"CUE-OUT,omitempty" json:",omitempty"`
	SegNE    string        `hls:"SEGNE,omitempty" json:",omitempty"`
}

// IsAD returns true if the cue is a cue-in or cue-out point
func (c SCTE35) IsAD() bool {
	return c.CueIn != "" || c.CueOut != ""
}

// DateRange is part of the official HLS specification, located here:
//
// https://datatracker.ietf.org/doc/html/draft-pantos-hls-rfc8216bis#section-4.4.5.1
type DateRange struct {
	ID       string        `hls:"ID,omitempty" json:",omitempty"`
	Class    string        `hls:"CLASS,omitempty" json:",omitempty"`
	Start    time.Time     `hls:"START-DATE,omitempty" json:",omitempty"`
	Cue      string        `hls:"CUE,omitempty" json:",omitempty"`
	End      time.Time     `hls:"END-DATE,omitempty" json:",omitempty"`
	Duration time.Duration `hls:"DURATION" json:",omitempty"`
	Planned  time.Duration `hls:"PLANNED-DURATION" json:",omitempty"`
	CueIn    string        `hls:"SCTE35-IN,omitempty" json:",omitempty"`
	CueOut   string        `hls:"SCTE35-OUT,omitempty" json:",omitempty"`
	Cmd      string        `hls:"SCTE35-CMD,omitempty" json:",omitempty"`
	EndNext  bool          `hls:"END-ON-NEXT,omitempty" json:",omitempty"`
}

// IsAD returns true if the cue is a cue-in or cue-out point
func (c DateRange) IsAD() bool {
	return c.CueIn != "" || c.CueOut != ""
}

// Cue is used by EXT-X-CUE-IN / EXT-X-CUE-OUT pairs
// the ID field is supported by Google Ad Manager for CUE-OUTs
type Cue struct {
	Duration time.Duration `hls:"$1" json:",omitempty"`
	Elapsed time.Duration `hls:"ELAPSEDTIME" json:",omitempty"`
	ID       string        `hls:"BREAKID" json:",omitempty"`
	SCTE35  string        `hls:"SCTE35" json:",omitempty"`
	Set      bool          `json:",omitempty"`
	Kind     string        `json:",omitempty"`
}

// IsAD returns true if the cue is a cue-in or cue-out point
func (c Cue) IsAD() bool {
	return c.Set || c.Duration != 0
}

func (c Cue) settag(t *m3u.Tag) {
	t.Flag = map[string]m3u.Value{}
	if c.Duration != 0 {
		t.Flag["DURATION"] = m3u.Value{V: fmt.Sprint(c.Duration.Seconds())}
		t.Keys = append(t.Keys, "DURATION")
	}
	if c.ID != "" {
		t.Flag["BREAKID"] = m3u.Value{V: c.ID}
		t.Keys = append(t.Keys, "BREAKID")
	}
	if c.Elapsed != 0 {
		t.Flag["ELAPSEDTIME"] = m3u.Value{V: fmt.Sprint(c.Elapsed.Seconds())}
		t.Keys = append(t.Keys, "ELAPSEDTIME")
	}
	if c.SCTE35 != "" {
		t.Flag["SCTE35"] = m3u.Value{V: c.SCTE35}
		t.Keys = append(t.Keys, "SCTE35")
	}
}

func (c *Cue) decodetag(t m3u.Tag) {
	dur := time.Duration(0)
	for _, v := range t.Arg {
		dur, _ = time.ParseDuration(v.V + "s")
		if dur != 0 {
			break
		}
	}
	if dur == 0 {
		dur, _ = time.ParseDuration(t.Value("DURATION") + "s")
	}
	c.Duration = dur
	c.Elapsed, _ = time.ParseDuration(t.Value("ELAPSEDTIME")+"s")
	c.Set = true
	c.ID = t.Value("BREAKID")
	c.SCTE35 = t.Value("SCTE35")
}

// CueAdobe is used by Adobe Prime Time in EXT-X-CUE tags
type CueAdobe struct {
	ID       string        `hls:"ID" json:",omitempty"`
	Type     string        `hls:"TYPE" json:",omitempty"`
	Duration time.Duration `hls:"DURATION" json:",omitempty"`
	Time     time.Duration `hls:"TIME" json:",omitempty"`
	Elapsed  time.Duration `hls:"ELAPSED" json:",omitempty"`
}
