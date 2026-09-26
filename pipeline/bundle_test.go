package pipeline

import (
	"encoding/json"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

func writtenItems(t *testing.T, raw []byte) map[int][]any {
	t.Helper()
	var doc struct {
		Items [][]any `json:"items"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	out := make(map[int][]any, len(doc.Items))
	for _, row := range doc.Items {
		out[int(row[0].(float64))] = row
	}
	return out
}

func TestItemNamesFollowTheOtherSourcesHyphens(t *testing.T) {
	entries := []Entry{
		{Item: scoring.Item{ID: 13298, Slot: scoring.Hair}, Name: "Honey-Soaked Song"},
		{Item: scoring.Item{ID: 12457, Slot: scoring.Hair}, Name: "Fair Lady-Gorgeous"},
		{Item: scoring.Item{ID: 20001, Slot: scoring.Dress}, Name: "Doll Dress-Blue"},
		{Item: scoring.Item{ID: 20002, Slot: scoring.Dress}, Name: "Day-Night Concerto"},
		{Item: scoring.Item{ID: 20003, Slot: scoring.Dress}, Name: "昼夜协奏曲"},
		{Item: scoring.Item{ID: 20004, Slot: scoring.Dress}, Name: "Fox Talk-Me"},
		{Item: scoring.Item{ID: 20005, Slot: scoring.Dress}, Name: "Moonlight -White"},
	}
	calc := map[int]string{13298: "Honey-Soaked Song", 12457: "Fair Lady-Gorgeous", 20001: "Doll Dress·Blue",
		20003: "Day-Night Concerto", 20004: "Fox Talk·Me", 30006: "Test Dance·Purple", 30007: "Salt&Pepper-Joy"}
	want := map[int]string{
		13298: "Honey-Soaked Song", 12457: "Fair Lady-Gorgeous", 20001: "Doll Dress - Blue",
		20002: "Day - Night Concerto", 20003: "Day-Night Concerto", 20004: "Fox Talk - Me", 20005: "Moonlight - White",
		30006: "Test Dance · Purple", 30007: "Salt & Pepper-Joy",
	}

	names := ItemNames{Calc: calc}
	rows := writtenItems(t, WriteItems(entries, names, nil, nil))
	cat := NewAcquisitionCatalogue(entries, names, nil)
	for id, name := range want {
		if got := rows[id][1]; got != name {
			t.Errorf("items.json names %d %q, want %q", id, got, name)
		}
		if got := cat.Names[id]; got != name {
			t.Errorf("acquire.json names %d %q, want %q", id, got, name)
		}
	}
}

func TestNameOnlyRowsCarryNikkiCalcRarity(t *testing.T) {
	entries := []Entry{{Item: scoring.Item{ID: 10001, Slot: scoring.Hair}, Name: "Rated", Rarity: 3}}
	calc := map[int]string{10001: "Rated", 30003: "Named Only", 880012: "Spirit Only"}
	rows := writtenItems(t, WriteItems(entries, ItemNames{Calc: calc}, map[int]int{10001: 5, 30003: 4, 880012: 6}, nil))
	for id, want := range map[int]float64{10001: 3, 30003: 4, 880012: 6} {
		if got := rows[id][13]; got != want {
			t.Errorf("%d: rarity %v, want %v", id, got, want)
		}
	}
}

func TestShownNamesReachItemsJSONAndTheLinesThatPrintThem(t *testing.T) {
	entries := []Entry{
		{Item: scoring.Item{ID: 20001, Slot: scoring.Dress}, Name: "Tset Gown"},
		{Item: scoring.Item{ID: 20002, Slot: scoring.Dress}, Name: "Plain-Gesture"},
	}
	names := ItemNames{
		Calc:  map[int]string{30003: "Named Only"},
		Shown: map[int]string{20001: "Test Gown", 30003: "Named Only (Coat)"},
	}
	rows := writtenItems(t, WriteItems(entries, names, nil, nil))
	cat := NewAcquisitionCatalogue(entries, names, nil)
	for id, want := range map[int][3]string{
		20001: {"Test Gown", "Tset Gown", "Test Gown"},
		20002: {"Plain - Gesture", "Plain - Gesture", "Plain - Gesture"},
		30003: {"Named Only (Coat)", "Named Only", "Named Only (Coat)"},
	} {
		if got := rows[id][1]; got != want[0] {
			t.Errorf("items.json names %d %q, want %q", id, got, want[0])
		}
		if got := cat.Names[id]; got != want[1] {
			t.Errorf("acquire.json matches %d by %q, want the sources' %q", id, got, want[1])
		}
		if got := cat.shown(id); got != want[2] {
			t.Errorf("acquire.json prints %d as %q, want %q", id, got, want[2])
		}
	}
}
