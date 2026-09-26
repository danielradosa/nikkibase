package pipeline

import (
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

func TestMergeTakesAPairOnlyWhenThePreferredRowLeavesItUngraded(t *testing.T) {
	preferred := graded(60001, scoring.Hosiery, "Test Socks", "hosiery", [5]string{"", "", "S", "", ""})
	preferred.Item.Attrs = [5]int8{0, 2, 4, 6, 8}
	packed := graded(60001, scoring.Hosiery, "测试袜子", "hosiery", [5]string{"A", "S", "SS", "SS", "B"})
	packed.Item.Attrs = [5]int8{0, 3, 5, 6, 9}

	got := MergeEntries([]Entry{preferred}, []Entry{packed})
	if len(got) != 1 {
		t.Fatalf("merged into %d rows, want 1", len(got))
	}
	e := got[0]
	if e.Name != "Test Socks" {
		t.Errorf("name %q; the preferred row's name must stand", e.Name)
	}
	want := struct {
		grades [5]string
		attrs  [5]int8
	}{[5]string{"A", "S", "S", "SS", "B"}, [5]int8{0, 3, 4, 6, 9}}
	if e.Grades != want.grades || e.Item.Attrs != want.attrs {
		t.Errorf("grades %v sides %v, want %v %v", e.Grades, e.Item.Attrs, want.grades, want.attrs)
	}
	for p, g := range e.Grades {
		if s := Stat(g, scoring.Hosiery); e.Item.Stats[p] != s {
			t.Errorf("pair %d: stat %d, and grade %s on hosiery is %d", p, e.Item.Stats[p], g, s)
		}
	}
	if v := checkStats(got); v != nil {
		t.Errorf("the merged row fails the stats invariant: %v", v)
	}
}

func TestMergeKeepsEveryPairThePreferredRowGrades(t *testing.T) {
	wiki := graded(10001, scoring.Hair, "Test Hair", "hair", [5]string{"S", "A", "B", "C", "SS"})
	wiki.Item.Attrs = [5]int8{1, 2, 5, 7, 8}
	other := graded(10001, scoring.Hair, "Test Hair", "hair", [5]string{"SSS", "SSS", "SSS", "SSS", "SSS"})
	other.Item.Attrs = [5]int8{0, 3, 4, 6, 9}

	got := MergeEntries([]Entry{wiki}, []Entry{other})
	if got[0].Grades != wiki.Grades || got[0].Item.Attrs != wiki.Item.Attrs || got[0].Item.Stats != wiki.Item.Stats {
		t.Errorf("merged %v %v %v, want the preferred row's %v %v %v", got[0].Grades, got[0].Item.Attrs,
			got[0].Item.Stats, wiki.Grades, wiki.Item.Attrs, wiki.Item.Stats)
	}
}
