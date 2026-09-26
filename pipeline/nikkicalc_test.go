package pipeline

import (
	"maps"
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

const calcKeys = `{"H1": 0, "D844": 1, "X12": 2, "A5": 3}`

func TestParseNikkicalcReadsNamesAndRarityInOnePass(t *testing.T) {
	items := []byte(`["001", 0, 0, 2, "Nikki's Pinky", "844", 1, 0, 11, "Violin Poet",
		"012", 2, 0, 6, "Spirit of Test", "005", 3, 5, 7, ""]`)
	names, rarity, err := ParseNikkicalc(items, []byte(calcKeys))
	if err != nil {
		t.Fatal(err)
	}
	if want := map[int]int{10001: 2, 20844: 5, 880012: 6, 80005: 1}; !maps.Equal(rarity, want) {
		t.Errorf("rarities %v, want %v", rarity, want)
	}
	want := map[int]string{10001: "Nikki's Pinky", 20844: "Violin Poet", 880012: "Spirit of Test"}
	if !maps.Equal(names, want) {
		t.Errorf("names %v, want %v", names, want)
	}
}

func TestParseNikkicalcRefusesUnknownRarityCodes(t *testing.T) {
	for _, code := range []string{"0", "13", "-1", `"5"`, "4.5"} {
		items := []byte(`["001", 0, 0, 2, "Nikki's Pinky", "844", 1, 0, ` + code + `, "Violin Poet"]`)
		_, _, err := ParseNikkicalc(items, []byte(calcKeys))
		if err == nil || !strings.Contains(err.Error(), "rarity") {
			t.Errorf("code %s: err = %v, want a refusal naming the rarity", code, err)
		}
	}
}

func TestFillRarityOnlyFillsBlanks(t *testing.T) {
	entries := []Entry{
		{Item: scoring.Item{ID: 10001, Slot: scoring.Hair}, Rarity: 0},
		{Item: scoring.Item{ID: 14118, Slot: scoring.Hair}, Rarity: 5},
		{Item: scoring.Item{ID: 880012, Slot: scoring.Spirit}, Rarity: 0},
		{Item: scoring.Item{ID: 20002, Slot: scoring.Dress}, Rarity: 0},
	}
	calc := map[int]string{10001: "Rated", 30003: "Named Only", 30004: "Unrated Name"}
	filled := FillRarity(entries, calc, map[int]int{10001: 3, 14118: 4, 880012: 6, 30003: 2})
	if filled != 3 {
		t.Errorf("filled %d, want 3: two blank items and one name-only row", filled)
	}
	for i, want := range []int{3, 5, 6, 0} {
		if entries[i].Rarity != want {
			t.Errorf("%d: rarity %d, want %d", entries[i].Item.ID, entries[i].Rarity, want)
		}
	}
}
