package pipeline

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

const packedSuitsSrc = `
var code2suit = ['甲套','乙套','丙套'];
var code2src = ['赠送'];
var codewardrobe = [
'甲|01|0||0|0|0|0',
'乙|02|0||*1|0|0|0',
'丙|03|0||@2|0|0|0',
'丁|04|0||!0|0|0|0',
'戊|05|0|||0|0|0',
'己|01|0||1|0|0|0',
'庚|09|0||2|0|0|0',
];
`

func TestParsePackedSuits(t *testing.T) {
	_, got, err := ParsePackedSources([]byte(packedSuitsSrc), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]PackedSuit{
		10001: {Name: "甲套"},
		10002: {Name: "乙套"},
		10003: {Name: "丙套"},
		10004: {Name: "甲套", Base: true},
		10009: {Name: "丙套"},
	}
	if !maps.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	_, known, err := ParsePackedSources([]byte(packedSuitsSrc), map[int]bool{10001: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 1 || known[10001].Name != "甲套" {
		t.Errorf("with a known set: %v, want only 10001", known)
	}
}

func TestParsePackedSuitsRefusesASuitTheTableLacks(t *testing.T) {
	src := strings.Replace(packedSuitsSrc, "'庚|09|0||2|0|0|0'", "'庚|09|0||3|0|0|0'", 1)
	if _, _, err := ParsePackedSources([]byte(src), nil); err == nil || !strings.Contains(err.Error(), "10009") {
		t.Errorf("err = %v, want the row naming suit 3 of 3 refused", err)
	}
}

func TestSuitsLayerTheWikiOverThePackedTable(t *testing.T) {
	ids := []int{10001, 10002, 10003, 10004, 10005, 10006, 10007, 30003}
	wiki := map[int]string{10001: "  Alpha   Suit ", 10006: "Alpha Suit", 30003: "Beta Suit"}
	packed := map[int]PackedSuit{
		10001: {Name: "乙套"},
		10002: {Name: "甲套"},
		10003: {Name: "乙套"},
		10004: {Name: "甲套", Base: true},
		10005: {Name: "丁套"},
		99999: {Name: "甲套"},
	}
	english := map[string]string{"甲套": "Alpha Suit", "乙套": " Beta  Suit"}
	got, stats := LayerSuits(ids, WikiAcquisition{SuitOf: wiki, ChineseSuits: english}, packed, nil)
	want := map[int]string{10001: "Alpha Suit", 10002: "Alpha Suit", 10003: "Beta Suit", 10006: "Alpha Suit", 30003: "Beta Suit"}
	if !maps.Equal(got, want) {
		t.Errorf("suits %v, want %v", got, want)
	}
	if want := (SuitStats{Wiki: 3, Packed: 2, Bases: 1, Unnamed: 1, None: 1, Suits: 2}); stats != want {
		t.Errorf("stats %+v, want %+v", stats, want)
	}
}

func TestSuitsTakeTheSuitTheirOtherPiecesHold(t *testing.T) {
	ids := []int{10001, 10002, 10003, 10004, 10005, 10006, 10007, 10008, 10009}
	wiki := map[int]string{10001: "Sexy Bad Girl", 10002: "Sexy Bad Girl", 10004: "Solo Suit", 10007: "Left Suit", 10008: "Right Suit"}
	packed := map[int]PackedSuit{
		10001: {Name: "甲"}, 10002: {Name: "甲"}, 10003: {Name: "甲"}, 10006: {Name: "甲", Base: true},
		10004: {Name: "乙"}, 10005: {Name: "乙"},
		10007: {Name: "丙"}, 10008: {Name: "丙"}, 10009: {Name: "丙"},
	}
	got, stats := LayerSuits(ids, WikiAcquisition{SuitOf: wiki}, packed, nil)
	if got[10003] != "Sexy Bad Girl" || stats.Members != 1 {
		t.Errorf("10003 in %q with %d placed by their other pieces, want Sexy Bad Girl and 1", got[10003], stats.Members)
	}
	for _, id := range []int{10005, 10006, 10009} {
		if got[id] != "" {
			t.Errorf("%d placed in %q, want no suit", id, got[id])
		}
	}
}

func TestSuitsTakeNikkiCalcNamesWhereTheWikiHasNone(t *testing.T) {
	ids := []int{10001, 20001, 20002, 20003, 20004, 20005, 20006, 20007, 20008, 20009, 20010, 20011, 20012, 20013, 20014}
	wiki := WikiAcquisition{
		SuitOf: map[int]string{10001: "Old Suit (Hidden Suit)", 20010: "Star Shadow (Hidden Suit)"},
		Packs:  map[string]bool{"Lucky Pack": true},
	}
	packed := map[int]PackedSuit{
		20001: {Name: "丁"}, 20002: {Name: "丁"}, 20003: {Name: "丁"},
		20004: {Name: "戊"}, 20005: {Name: "戊"},
		20006: {Name: "己"},
		20007: {Name: "庚"},
		20008: {Name: "辛"}, 20009: {Name: "壬"},
		20010: {Name: "癸"}, 20011: {Name: "癸"},
	}
	calc := map[int][]string{
		20001: {"New Suit"}, 20002: {"New Suit"},
		20004: {"Alpha"}, 20005: {"Beta"},
		20006: {"Old Suit"},
		20007: {"Lucky Pack"},
		20008: {"Twin"}, 20009: {"Twin"},
		20011: {"Star Shadow"},
		20012: {"New Suit"}, 20013: {"old  suit"}, 20014: {"Alpha", "Beta"},
	}
	got, stats := LayerSuits(ids, wiki, packed, calc)
	want := map[int]string{
		10001: "Old Suit (Hidden Suit)", 20010: "Star Shadow (Hidden Suit)",
		20001: "New Suit", 20002: "New Suit", 20003: "New Suit",
		20011: "Star Shadow (Hidden Suit)",
		20012: "New Suit", 20013: "Old Suit (Hidden Suit)",
	}
	if !maps.Equal(got, want) {
		t.Errorf("suits %v, want %v", got, want)
	}
	if want := (SuitStats{Wiki: 2, Calc: 6, Unnamed: 6, None: 1, Suits: 3}); stats != want {
		t.Errorf("stats %+v, want %+v", stats, want)
	}
}

func TestCalcSuitsMapMembersToGameIDsAndDropSharedNames(t *testing.T) {
	table := []byte(`{"names": ["New Suit", "Twin", " Twin ", "Solo"], "offsets": [0, 2, 3, 4], "counts": [2, 1, 1, 2],
		"clothes": [0, 1, 1, 2, 0, 9], "icons": [0, 0, 0, 0]}`)
	got, stats, err := ParseCalcSuits(table, []byte(`{"H1": 0, "D2": 1, "C3": 2}`))
	if err != nil {
		t.Fatal(err)
	}
	want := map[int][]string{10001: {"New Suit", "Solo"}, 20002: {"New Suit"}}
	if !maps.EqualFunc(got, want, slices.Equal) {
		t.Errorf("suits %v, want %v", got, want)
	}
	if stats != (CalcSuitStats{Suits: 2, Ambiguous: 2, Items: 2, Unkeyed: 1}) {
		t.Errorf("stats %+v", stats)
	}
	if _, _, err := ParseCalcSuits([]byte(`{"names": ["A"], "offsets": [0], "counts": [3], "clothes": [0]}`), []byte(`{}`)); err == nil {
		t.Error("a suit running past the clothes list was accepted")
	}
}

func TestSuitsReachEveryRowOfItemsJSON(t *testing.T) {
	entries := []Entry{
		{Item: scoring.Item{ID: 10001, Slot: scoring.Hair}, Name: "Graded Hair"},
		{Item: scoring.Item{ID: 20002, Slot: scoring.Dress}, Name: "Plain Dress"},
	}
	suits := map[int]string{10001: "Alpha Suit", 30003: "Alpha Suit"}
	ApplySuits(entries, suits)
	if entries[0].Suit != "Alpha Suit" || entries[1].Suit != "" {
		t.Errorf("entries carry %q and %q, want Alpha Suit and nothing", entries[0].Suit, entries[1].Suit)
	}
	rows := writtenItems(t, WriteItems(entries, ItemNames{Calc: map[int]string{30003: "Named Only", 40004: "Suitless"}}, nil, suits))
	for id, want := range map[int]string{10001: "Alpha Suit", 20002: "", 30003: "Alpha Suit", 40004: ""} {
		if got := rows[id][14]; got != want {
			t.Errorf("%d: suit %q, want %q", id, got, want)
		}
	}
}

func TestSuitsAreHeldToTheirFloorAndToEnglish(t *testing.T) {
	suits := map[int]string{10001: "Alpha Suit", 10002: "Alpha Suit", 10003: "Beta Suit"}
	if v := CheckSuits(suits, Coverage{SuitItems: 3}); len(v) != 0 {
		t.Errorf("at the floor: %v", v)
	}
	if v := CheckSuits(suits, Coverage{SuitItems: 4}); len(v) != 1 || !strings.Contains(v[0], "3 items") {
		t.Errorf("below the floor: %v", v)
	}
	suits[10004] = "甲套"
	if v := CheckSuits(suits, Coverage{}); len(v) != 1 || !strings.Contains(v[0], "10004") {
		t.Errorf("a Chinese suit name: %v", v)
	}
}
