package pipeline

import (
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

const subgradeKeys = `{"H1": 0, "Z408": 1, "X5": 2, "D9": 7, "Q3": 8}`

func TestSubgradesMapToGameIDs(t *testing.T) {
	first := []byte(`{
		"0": {"slot": 0, "name": "Nikki's Pinky", "desc": "text", "grades": ["F", "F", "F", "F", "D+"],
			"attrs": [1, 3, 5, 7, 8], "niGrades": ["S-", "A-", "A-", "A-", "A"]},
		"1": {"slot": 12, "attrs": [0, 2, 4, 6, 9], "niGrades": ["SS+", "C", "B+", "SSS", ""]}}`)
	second := []byte(`{
		"2": {"slot": 30, "attrs": [1, 2, 5, 6, 8], "niGrades": ["S", "S+", "SS-", "C+", "B-"]},
		"3": {"slot": 0, "attrs": [1, 3, 5, 7, 8], "niGrades": ["A", "A", "A", "A", "A"]},
		"8": {"slot": 0, "attrs": [1, 3, 5, 7, 8], "niGrades": ["A", "A", "A", "A", "A"]}}`)
	subs, stats, err := ParseSubgrades([][]byte{first, second}, []byte(subgradeKeys))
	if err != nil {
		t.Fatal(err)
	}
	want := map[int][5]Subgrade{
		10001:  {{1, "S-"}, {3, "A-"}, {5, "A-"}, {7, "A-"}, {8, "A"}},
		170408: {{0, "SS+"}, {2, "C"}, {4, "B+"}, {6, "SSS"}, {9, ""}},
		880005: {{1, "S"}, {2, "S+"}, {5, "SS-"}, {6, "C+"}, {8, "B-"}},
	}
	if len(subs) != len(want) {
		t.Fatalf("parsed %d items %v, want %d", len(subs), subs, len(want))
	}
	for id, row := range want {
		if subs[id] != row {
			t.Errorf("%d: %v, want %v", id, subs[id], row)
		}
	}
	if stats.Files != 2 || stats.Records != 5 || stats.Unkeyed != 2 {
		t.Errorf("stats %+v, want 2 files, 5 records, 2 without a key", stats)
	}
}

func TestSubgradesRejectWhatTheTableCannotPrice(t *testing.T) {
	record := func(attrs, grades string) []byte {
		return []byte(`{"0": {"attrs": ` + attrs + `, "niGrades": ` + grades + `}}`)
	}
	for name, c := range map[string]struct {
		batches [][]byte
		keys    string
		want    string
	}{
		"C-":            {[][]byte{record(`[1, 3, 5, 7, 8]`, `["C-", "A", "A", "A", "A"]`)}, subgradeKeys, `"C-"`},
		"SSS+":          {[][]byte{record(`[1, 3, 5, 7, 8]`, `["A", "SSS+", "A", "A", "A"]`)}, subgradeKeys, `"SSS+"`},
		"SSS-":          {[][]byte{record(`[1, 3, 5, 7, 8]`, `["A", "A", "SSS-", "A", "A"]`)}, subgradeKeys, `"SSS-"`},
		"lower case":    {[][]byte{record(`[1, 3, 5, 7, 8]`, `["A", "A", "A", "s", "A"]`)}, subgradeKeys, `"s"`},
		"letter grade":  {[][]byte{record(`[1, 3, 5, 7, 8]`, `["A", "A", "A", "A", "D+"]`)}, subgradeKeys, `"D+"`},
		"other pair":    {[][]byte{record(`[1, 3, 7, 7, 8]`, `["A", "A", "A", "A", "A"]`)}, subgradeKeys, "pair 2"},
		"short":         {[][]byte{record(`[1, 3, 5, 7]`, `["A", "A", "A", "A"]`)}, subgradeKeys, "4 attributes"},
		"two batches":   {[][]byte{record(`[1, 3, 5, 7, 8]`, `["A", "A", "A", "A", "A"]`), record(`[1, 3, 5, 7, 8]`, `["A", "A", "A", "A", "A"]`)}, subgradeKeys, "two item batches"},
		"shared index":  {[][]byte{record(`[1, 3, 5, 7, 8]`, `["A", "A", "A", "A", "A"]`)}, `{"H1": 0, "H2": 0}`, "H1 and H2"},
		"not an index":  {[][]byte{[]byte(`{"x": {"attrs": [1, 3, 5, 7, 8], "niGrades": ["A", "A", "A", "A", "A"]}}`)}, subgradeKeys, `"x"`},
		"not an object": {[][]byte{[]byte(`[]`)}, subgradeKeys, "batch 1 of 1"},
		"without a key": {[][]byte{[]byte(`{"9": {"attrs": [1, 3, 5, 7, 8], "niGrades": ["A", "A", "A", "A", "SSS+"]}}`)}, subgradeKeys, "record 9"},
	} {
		if _, _, err := ParseSubgrades(c.batches, []byte(c.keys)); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want one naming %s", name, err, c.want)
		}
	}
}

func TestSubgradesSetStatsOnlyWithinTheirLetterAndSide(t *testing.T) {
	hair := graded(10001, scoring.Hair, "Hair", "hair", [5]string{"S", "A", "A", "A", "A"})
	hair.Item.Attrs = [5]int8{1, 3, 5, 7, 8}
	unsourced := graded(10002, scoring.Hair, "Unsourced", "hair", [5]string{"B", "B", "B", "B", "B"})
	ungraded := graded(10003, scoring.Hair, "Name only", "hair", [5]string{})
	entries := []Entry{hair, unsourced, ungraded}
	before := append([]Entry(nil), entries...)

	subs := map[int][5]Subgrade{
		10001: {{1, "S-"}, {2, "A+"}, {5, "S"}, {7, "A"}, {8, ""}},
		10003: {{0, "SS"}, {2, "SS"}, {4, "SS"}, {6, "SS"}, {8, "SS"}},
		10004: {{0, "A"}, {2, "A"}, {4, "A"}, {6, "A"}, {8, "A"}},
	}
	var stats SubgradeStats
	ApplySubgrades(entries, subs, &stats)

	got := entries[0]
	if want := [5]int{64, Stat("A", scoring.Hair), Stat("A", scoring.Hair), 57, Stat("A", scoring.Hair)}; got.Item.Stats != want {
		t.Errorf("stats %v, want %v: S- and plain A set, the other side, another letter and no sub-grade keep the letter's stat", got.Item.Stats, want)
	}
	if want := [5]string{"S-", "", "", "A", ""}; got.Subgrades != want {
		t.Errorf("sub-grades %v, want %v", got.Subgrades, want)
	}
	if Stat("S", scoring.Hair) != 70 || Stat("A", scoring.Hair) != 56 {
		t.Errorf("the letter table moved: S %d, A %d on hair", Stat("S", scoring.Hair), Stat("A", scoring.Hair))
	}
	for i, e := range entries {
		if e.Grades != before[i].Grades || e.Item.Attrs != before[i].Item.Attrs {
			t.Errorf("%d: letters or sides changed to %v %v", e.Item.ID, e.Grades, e.Item.Attrs)
		}
	}
	if entries[1].Item.Stats != before[1].Item.Stats || entries[2].Item.Stats != before[2].Item.Stats {
		t.Errorf("an item without a matching sub-grade changed: %v %v", entries[1].Item.Stats, entries[2].Item.Stats)
	}
	want := SubgradeStats{Applied: 2, OtherSide: 1, OtherLetter: 1, Missing: 6, NotInCatalogue: 1}
	if stats != want {
		t.Errorf("stats %+v, want %+v", stats, want)
	}
	if v := checkStats(entries); v != nil {
		t.Errorf("sub-graded items fail the stats invariant: %v", v)
	}
}

func TestSubStatPricesEveryStep(t *testing.T) {
	for grade, want := range map[string]int{
		"C": 81, "C+": 109, "B-": 134, "B": 163, "B+": 185, "A-": 206, "A": 228, "A+": 251,
		"S-": 257, "S": 280, "S+": 310, "SS-": 336, "SS": 366, "SS+": 400, "SSS": 427,
	} {
		if got := SubStat(grade, scoring.Dress); got != want {
			t.Errorf("%s on a dress = %d, want %d", grade, got, want)
		}
	}
	for _, grade := range []string{"", "C-", "SSS+", "D", "s"} {
		if got := SubStat(grade, scoring.Dress); got != 0 {
			t.Errorf("%q on a dress = %d, want 0", grade, got)
		}
	}
}

func TestStatsInvariantAcceptsASubgradeOnlyWithinItsLetter(t *testing.T) {
	e := graded(10001, scoring.Hair, "Hair", "hair", [5]string{"S", "A", "A", "A", "A"})
	e.Subgrades[0], e.Item.Stats[0] = "S-", SubStat("S-", scoring.Hair)
	if v := checkStats([]Entry{e}); v != nil {
		t.Errorf("a sub-graded stat failed: %v", v)
	}

	stale := e
	stale.Item.Stats[0] = Stat("S", scoring.Hair)
	if v := checkStats([]Entry{stale}); len(v) != 1 || !strings.Contains(v[0], `grade "S-"`) {
		t.Errorf("violations = %v, want one naming the sub-grade", v)
	}
	outside := e
	outside.Subgrades[0], outside.Item.Stats[0] = "A+", SubStat("A+", scoring.Hair)
	if v := checkStats([]Entry{outside}); len(v) != 1 || !strings.Contains(v[0], "outside its grade") {
		t.Errorf("violations = %v, want one naming a sub-grade outside its letter", v)
	}
	unclaimed := e
	unclaimed.Subgrades[0] = ""
	if checkStats([]Entry{unclaimed}) == nil {
		t.Error("a sub-grade stat with no sub-grade behind it passed")
	}
}

func TestSubgradeCoverageBindsOnlyBuildsThatReadThem(t *testing.T) {
	e := graded(10001, scoring.Hair, "Hair", "hair", [5]string{"S", "A", "A", "A", "A"})
	e.Subgrades = [5]string{"S-", "A", "", "", ""}
	entries := []Entry{e}
	want := Coverage{SubgradedCells: 3, MaxSubgradeFallbacks: 2}
	fell := &SubgradeStats{OtherSide: 1, OtherLetter: 1, Missing: 1}

	if v := checkSubgrades(entries, nil, want); v != nil {
		t.Errorf("a build without sub-grades failed their floor: %v", v)
	}
	v := checkSubgrades(entries, fell, want)
	if len(v) != 2 || !strings.Contains(v[0], "2 item stats come from a sub-grade") || !strings.Contains(v[1], "3 graded item stats") {
		t.Errorf("violations = %v, want the floor and the ceiling", v)
	}
	want.SubgradedCells, want.MaxSubgradeFallbacks = 2, 3
	if v := checkSubgrades(entries, fell, want); v != nil {
		t.Errorf("a build at its floor and ceiling failed: %v", v)
	}
}
