package pipeline

import (
	"slices"
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

func agreementEntry(id int, slot scoring.Slot, attrs [5]int8, grades [5]string) Entry {
	e := Entry{Item: scoring.Item{ID: id, Slot: slot, Attrs: attrs}, Grades: grades}
	for p, g := range grades {
		e.Item.Stats[p] = Stat(g, slot)
	}
	return e
}

func calcRow(attrs [5]int8, grades [5]string) CalcItem {
	var it CalcItem
	for p := range grades {
		it.Row[p] = Subgrade{Attr: attrs[p], Grade: grades[p]}
	}
	return it
}

func TestAgreementOverridesOnlyPairsBothOtherSourcesSettle(t *testing.T) {
	sides := [5]int8{1, 3, 5, 7, 8}
	flipped := [5]int8{0, 3, 5, 7, 8}
	a := [5]string{"A", "A", "A", "A", "A"}
	entries := []Entry{
		agreementEntry(10001, scoring.Hair, sides, a),
		agreementEntry(20002, scoring.Dress, flipped, a),
		agreementEntry(30003, scoring.Coat, sides, a),
		agreementEntry(40004, scoring.Top, sides, [5]string{"S", "A", "A", "A", "A"}),
		agreementEntry(50005, scoring.Bottom, sides, a),
		agreementEntry(60006, scoring.Shoes, sides, a),
	}
	given := map[int][5]string{
		10001: a, 20002: a, 30003: a,
		50005: {"A", "", "A", "A", "A"},
		60006: a,
	}
	packed := TableRows([]Entry{
		agreementEntry(10001, scoring.Hair, sides, [5]string{"S", "A", "A", "A", "A"}),
		agreementEntry(20002, scoring.Dress, sides, a),
		agreementEntry(30003, scoring.Coat, sides, [5]string{"S", "A", "A", "A", "A"}),
		agreementEntry(40004, scoring.Top, sides, [5]string{"S", "A", "A", "A", "A"}),
		agreementEntry(50005, scoring.Bottom, sides, [5]string{"A", "S", "A", "A", "A"}),
		agreementEntry(60006, scoring.Shoes, sides, [5]string{"S", "S", "S", "A", "A"}),
	})
	calc := map[int]CalcItem{
		10001: calcRow(sides, [5]string{"S-", "A", "A+", "A", "A"}),
		20002: calcRow(sides, [5]string{"A+", "A", "A", "A", "A"}),
		30003: calcRow(sides, [5]string{"A", "A", "A", "A", "A"}),
		40004: calcRow(sides, [5]string{"S", "A", "A", "A", "A"}),
		50005: calcRow(sides, [5]string{"A", "S", "A", "A", "A"}),
		60006: calcRow(sides, [5]string{"S+", "S", "S-", "A", "A"}),
	}
	stats := CorrectByAgreement(entries, given, packed, calc)

	want := map[int][5]string{
		10001: {"S", "A", "A", "A", "A"},
		20002: a,
		30003: a,
		40004: {"S", "A", "A", "A", "A"},
		50005: a,
		60006: {"S", "S", "S", "A", "A"},
	}
	for _, e := range entries {
		if e.Grades != want[e.Item.ID] {
			t.Errorf("%d: grades %v, want %v", e.Item.ID, e.Grades, want[e.Item.ID])
		}
		for p, g := range e.Grades {
			if e.Item.Stats[p] != Stat(g, e.Item.Slot) {
				t.Errorf("%d pair %d: stat %d, want %s's", e.Item.ID, p, e.Item.Stats[p], g)
			}
		}
	}
	if entries[1].Item.Attrs != sides {
		t.Errorf("20002: sides %v, want the side both other sources give, %v", entries[1].Item.Attrs, sides)
	}
	if stats.Pairs != 5 || stats.Sides != 1 || !slices.Equal(stats.Items, []int{10001, 20002, 60006}) ||
		!slices.Equal(stats.Heavy, []int{60006}) {
		t.Errorf("stats %+v", stats)
	}
}

func TestTableRowsDropIDsTheTableGivesTwoWays(t *testing.T) {
	sides := [5]int8{1, 3, 5, 7, 8}
	rows := TableRows([]Entry{
		agreementEntry(10001, scoring.Hair, sides, [5]string{"A", "A", "A", "A", "A"}),
		agreementEntry(10001, scoring.Hair, sides, [5]string{"S", "A", "A", "A", "A"}),
		agreementEntry(20002, scoring.Dress, sides, [5]string{"A", "A", "A", "A", "A"}),
		agreementEntry(20002, scoring.Dress, sides, [5]string{"A", "A", "A", "A", "A"}),
	})
	if _, ok := rows[10001]; ok || len(rows) != 1 {
		t.Errorf("rows %v, want only 20002", rows)
	}
}

func TestAgreementIsHeldUnderItsCeiling(t *testing.T) {
	if v := CheckAgreement(AgreementStats{Pairs: 2}, Coverage{MaxAgreedGrades: 2}); len(v) != 0 {
		t.Errorf("at the ceiling: %v", v)
	}
	v := CheckAgreement(AgreementStats{Pairs: 3}, Coverage{MaxAgreedGrades: 2})
	if len(v) != 1 || !strings.Contains(v[0], "ceiling is 2") {
		t.Errorf("over the ceiling: %v", v)
	}
}

func TestAgreementLeavesTheGradesTheCorrectionsKeep(t *testing.T) {
	c := corrections(t, `{"gradeOverrides": [{"id": 10001, "attribute": "Simple", "grade": "SS", "was": "A", "basis": "checked in game"}]}`)
	sides := [5]int8{1, 3, 5, 7, 8}
	a := [5]string{"A", "A", "A", "A", "A"}
	entries, err := c.Apply([]Entry{agreementEntry(10001, scoring.Hair, sides, a)})
	if err != nil {
		t.Fatal(err)
	}
	given := map[int][5]string{10001: a, 20002: a}
	c.HoldGrades(given)
	if given[10001] != [5]string{"", "A", "A", "A", "A"} || given[20002] != a {
		t.Fatalf("given = %v", given)
	}
	packed := TableRows([]Entry{agreementEntry(10001, scoring.Hair, sides, [5]string{"S", "S", "A", "A", "A"})})
	calc := map[int]CalcItem{10001: calcRow(sides, [5]string{"S-", "S", "A", "A", "A"})}
	stats := CorrectByAgreement(entries, given, packed, calc)
	if g := entries[0].Grades; g[0] != "SS" || g[1] != "S" {
		t.Errorf("grades = %v, want the kept SS and the agreed S", g)
	}
	if stats.Pairs != 1 {
		t.Errorf("pairs = %d, want 1", stats.Pairs)
	}
}
