package pipeline

import (
	"strings"
	"testing"
)

func TestHandCheckedLinesFillOnlyItemsTheSourcesLeaveOut(t *testing.T) {
	extra, err := ReadAcquisitionExtra([]byte(`{"note": "n", "items": [
		{"id": 1, "name": "Neon Dance Steps", "kind": "event", "text": "Cosmic Murmurs event", "basis": "event page"},
		{"id": 2, "name": "Gifty's Wish", "kind": "recharge", "text": "One-Dollar Sale", "basis": "timeline"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	cat := AcquisitionCatalogue{Names: map[int]string{1: "Neon Dance Steps", 2: "Gifty's Wish", 3: "Other"}}
	acq := map[int][]Acquisition{3: {{Kind: "store", Text: "Clothes Store"}}}
	if err := ApplyAcquisitionExtra(acq, cat, extra); err != nil {
		t.Fatal(err)
	}
	want := `{"version":"v","items":{"1":[{"k":"event","t":"Cosmic Murmurs event","past":1}],` +
		`"2":[{"k":"recharge","t":"One-Dollar Sale","past":1}],"3":[{"k":"store","t":"Clothes Store"}]}}`
	if got := string(WriteAcquisition("v", acq)); got != want {
		t.Errorf("\n got %s\nwant %s", got, want)
	}
}

func TestHandCheckedLinesGoStaleLoudly(t *testing.T) {
	line := `{"id": 1, "name": "Neon Dance Steps", "kind": "event", "text": "Cosmic Murmurs event", "basis": "event page"}`
	for name, c := range map[string]struct {
		doc  string
		cat  map[int]string
		acq  map[int][]Acquisition
		want string
	}{
		"twice":      {`{"items": [` + line + `, ` + line + `]}`, nil, nil, "listed twice"},
		"kind":       {`{"items": [{"id": 1, "name": "A", "kind": "shop", "text": "t", "basis": "b"}]}`, nil, nil, `kind "shop"`},
		"no basis":   {`{"items": [{"id": 1, "name": "A", "kind": "event", "text": "t"}]}`, nil, nil, "basis"},
		"gone":       {`{"items": [` + line + `]}`, map[int]string{}, nil, "not in the catalogue"},
		"renamed":    {`{"items": [` + line + `]}`, map[int]string{1: "Other Item"}, nil, `is "Other Item"`},
		"has a line": {`{"items": [` + line + `]}`, map[int]string{1: "Neon Dance Steps"}, map[int][]Acquisition{1: {{Kind: "store", Text: "Clothes Store"}}}, "now has a line"},
	} {
		extra, err := ReadAcquisitionExtra([]byte(c.doc))
		if err == nil {
			acq := c.acq
			if acq == nil {
				acq = map[int][]Acquisition{}
			}
			err = ApplyAcquisitionExtra(acq, AcquisitionCatalogue{Names: c.cat}, extra)
		}
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want one naming %q", name, err, c.want)
		}
	}
}
