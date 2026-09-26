package pipeline

import (
	"maps"
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

const packedSuitsSrc = `
var code2suit = ['甲套','乙套','丙套'];
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
	got, err := ParsePackedSuits([]byte(packedSuitsSrc), nil)
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
	known, err := ParsePackedSuits([]byte(packedSuitsSrc), map[int]bool{10001: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 1 || known[10001].Name != "甲套" {
		t.Errorf("with a known set: %v, want only 10001", known)
	}
}

func TestParsePackedSuitsRefusesASuitTheTableLacks(t *testing.T) {
	src := strings.Replace(packedSuitsSrc, "'庚|09|0||2|0|0|0'", "'庚|09|0||3|0|0|0'", 1)
	if _, err := ParsePackedSuits([]byte(src), nil); err == nil || !strings.Contains(err.Error(), "10009") {
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
	got, stats := LayerSuits(ids, WikiAcquisition{SuitOf: wiki, ChineseSuits: english}, packed)
	want := map[int]string{10001: "Alpha Suit", 10002: "Alpha Suit", 10003: "Beta Suit", 10006: "Alpha Suit", 30003: "Beta Suit"}
	if !maps.Equal(got, want) {
		t.Errorf("suits %v, want %v", got, want)
	}
	if want := (SuitStats{Wiki: 3, Packed: 2, Bases: 1, Unnamed: 1, None: 1, Suits: 2}); stats != want {
		t.Errorf("stats %+v, want %+v", stats, want)
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
