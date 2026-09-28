package pipeline

import (
	"slices"
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

func TestCalcGradesFillOnlyItemsNoOtherSourceGrades(t *testing.T) {
	row := func(grades ...string) [5]Subgrade {
		attrs := [5]int8{1, 3, 5, 7, 8}
		var r [5]Subgrade
		for p, g := range grades {
			r[p] = Subgrade{Attr: attrs[p], Grade: g}
		}
		return r
	}
	items := map[int]CalcItem{
		10001: {Place: 0, Row: row("S", "S", "S", "S", "S")},
		20005: {Place: 2, Row: row("SS+", "A-", "B", "C+", "S"), Tags: []int{0, 43, 44, 0}},
		85001: {Place: 43, Row: row("A", "A", "A", "A", "A")},
		30961: {Place: 7, Row: row("A", "A", "A", "A", "A")},
		12000: {Place: 0, Row: row("A", "A", "", "A", "A")},
	}
	names := map[int]string{
		10001: "Graded Elsewhere", 20005: " Calc Dress ", 85001: "Calc Ears", 30961: "Top Filed As Coat",
		12000: "Half Graded", 13000: "No Record",
	}
	got, stats := CalcEntries(items, names, map[int]bool{10001: true})

	if len(got) != 2 || got[0].Item.ID != 20005 || got[1].Item.ID != 85001 {
		t.Fatalf("graded %+v, want 20005 then 85001", got)
	}
	dress := got[0]
	if dress.Name != "Calc Dress" || dress.Position != "dress" || dress.Item.Slot != scoring.Dress {
		t.Errorf("20005: name %q, place %q, slot %v", dress.Name, dress.Position, dress.Item.Slot)
	}
	if want := [5]string{"SS", "A", "B", "C", "S"}; dress.Grades != want {
		t.Errorf("20005: grades %v, want the sub-grades' letters %v", dress.Grades, want)
	}
	if want := [5]int8{1, 3, 5, 7, 8}; dress.Item.Attrs != want {
		t.Errorf("20005: sides %v, want %v", dress.Item.Attrs, want)
	}
	for p, g := range dress.Grades {
		if dress.Item.Stats[p] != Stat(g, scoring.Dress) {
			t.Errorf("20005 pair %d: stat %d, want %s's %d", p, dress.Item.Stats[p], g, Stat(g, scoring.Dress))
		}
	}
	sports, _ := TagID("Sports")
	street, _ := TagID("Street")
	if !slices.Equal(dress.Item.Tags, []int{sports, street}) {
		t.Errorf("20005: tags %v, want Sports and Street once each", dress.Item.Tags)
	}
	if got[1].Position != "accessory_ears" || len(got[1].Item.Tags) != 0 {
		t.Errorf("85001: place %q, tags %v", got[1].Position, got[1].Item.Tags)
	}
	if stats.Candidates != 5 || stats.Graded != 2 || stats.Tagged != 1 || stats.UnknownTags != 1 ||
		stats.NoRecord != 1 || stats.Incomplete != 1 || !slices.Equal(stats.Unplaced, []int{30961}) {
		t.Errorf("stats %+v", stats)
	}
}

func TestCalcTablesNameOnlyKnownTagsAndPlaces(t *testing.T) {
	for i, name := range calcTags {
		if _, ok := TagID(name); !ok {
			t.Errorf("tag %d %q is not in the tag table", i, name)
		}
	}
	for code, place := range calcPlaces {
		if _, ok := byName[place]; !ok {
			t.Errorf("slot code %d names %q, which is no wearable place", code, place)
		}
	}
}

func TestCalcGradesAreHeldUnderTheirCeiling(t *testing.T) {
	if v := CheckCalcGrades(CalcStats{Graded: 3}, Coverage{MaxCalcGradedItems: 3}); len(v) != 0 {
		t.Errorf("at the ceiling: %v", v)
	}
	v := CheckCalcGrades(CalcStats{Graded: 4}, Coverage{MaxCalcGradedItems: 3})
	if len(v) != 1 || !strings.Contains(v[0], "ceiling is 3") {
		t.Errorf("over the ceiling: %v", v)
	}
}
