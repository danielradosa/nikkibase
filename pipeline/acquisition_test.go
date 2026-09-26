package pipeline

import (
	"strings"
	"testing"
)

func TestMergeAcquisitionNeverMixesSources(t *testing.T) {
	cat := AcquisitionCatalogue{Names: map[int]string{1: "A", 2: "B", 3: "C", 4: "D"}}
	wiki := map[int][]Acquisition{1: {{Kind: "store", Text: "Clothes Store"}}, 9: {{Kind: "store", Text: "Clothes Store"}}}
	packed := map[int][]Acquisition{
		1: {{Kind: "event", Text: "Limited event", CN: true}},
		2: {{Kind: "craft", Text: "Crafting", CN: true}},
	}
	suits := map[int][]Acquisition{2: {{Kind: "recharge", Text: "Recharge"}}, 3: {{Kind: "recharge", Text: "Recharge"}}}
	got, stats := MergeAcquisition(cat, wiki, packed, suits)
	want := `{"version":"v","items":{"1":[{"k":"store","t":"Clothes Store"}],"2":[{"k":"craft","t":"Crafting","cn":1}],"3":[{"k":"recharge","t":"Recharge"}]}}`
	if out := string(WriteAcquisition("v", got)); out != want {
		t.Errorf("\n got %s\nwant %s", out, want)
	}
	if stats != (AcquisitionStats{Catalogue: 4, Covered: 3, FromWiki: 1, FromPacked: 1, FromSuits: 1}) {
		t.Errorf("stats = %+v", stats)
	}
}

func TestCheckAcquisition(t *testing.T) {
	cat := AcquisitionCatalogue{Names: map[int]string{1: "A", 2: "B"}, Stages: map[string]bool{"Story/1-1": true}}
	good := map[int][]Acquisition{
		1: {{Kind: "stage", Text: "Story 1-1 (Maiden)", Stage: "Story/1-1", Level: "Maiden"}},
		2: {{Kind: "evolve", Text: "Evolve: A", From: []Ingredient{{ID: 1, Qty: 0}}}},
	}
	if v := CheckAcquisition(good, cat, Coverage{AcquisitionItems: 2}); len(v) != 0 {
		t.Errorf("a sound table failed: %v", v)
	}
	if v := CheckAcquisition(good, cat, Coverage{AcquisitionItems: 3}); len(v) != 1 || !strings.Contains(v[0], "floor is 3") {
		t.Errorf("the coverage floor was not enforced: %v", v)
	}
	for name, a := range map[string]Acquisition{
		"kind":       {Kind: "shop", Text: "Shop"},
		"text":       {Kind: "store", Text: " "},
		"Chinese":    {Kind: "store", Text: "店·金币"},
		"stage":      {Kind: "stage", Text: "Story 9-9", Stage: "Story/9-9"},
		"level":      {Kind: "stage", Text: "Story 1-1", Stage: "Story/1-1", Level: "Queen"},
		"ingredient": {Kind: "craft", Text: "Craft: C", From: []Ingredient{{ID: 3, Qty: 1}}},
		"cost":       {Kind: "store", Text: "Clothes Store", Cost: []Cost{{Amount: 0, Unit: "Gold"}}},
	} {
		v := CheckAcquisition(map[int][]Acquisition{1: {a}}, cat, Coverage{})
		if len(v) != 1 {
			t.Errorf("%s: %v", name, v)
		}
	}
	if v := CheckAcquisition(map[int][]Acquisition{7: {{Kind: "store", Text: "Clothes Store"}}}, cat, Coverage{}); len(v) != 1 {
		t.Errorf("an item outside the catalogue passed: %v", v)
	}
}

func TestCheckAcquisitionRefusesImpossibleBases(t *testing.T) {
	cat := AcquisitionCatalogue{Names: map[int]string{20054: "Pink", 20055: "Blue", 30769: "Coat", 40713: "Top"}}
	for name, c := range map[string]struct {
		id int
		a  Acquisition
	}{
		"itself":        {20055, Acquisition{Kind: "customize", Text: "Customize: Blue", From: []Ingredient{{ID: 20055, Qty: 1}}}},
		"another slot":  {40713, Acquisition{Kind: "customize", Text: "Customize: Coat", From: []Ingredient{{ID: 30769, Qty: 1}}}},
		"evolve itself": {20054, Acquisition{Kind: "evolve", Text: "Evolve: Pink", From: []Ingredient{{ID: 20054, Qty: 0}}}},
	} {
		if v := CheckAcquisition(map[int][]Acquisition{c.id: {c.a}}, cat, Coverage{}); len(v) != 1 {
			t.Errorf("%s: %v", name, v)
		}
	}
	sound := map[int][]Acquisition{
		20055: {{Kind: "customize", Text: "Customize: Pink", From: []Ingredient{{ID: 20054, Qty: 1}}}},
		40713: {{Kind: "craft", Text: "Craft: Coat", From: []Ingredient{{ID: 30769, Qty: 2}}}},
	}
	if v := CheckAcquisition(sound, cat, Coverage{}); len(v) != 0 {
		t.Errorf("a sound base or a recipe from another slot failed: %v", v)
	}
}

func TestThousands(t *testing.T) {
	for n, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 18314: "18,314", 1234567: "1,234,567"} {
		if got := thousands(n); got != want {
			t.Errorf("thousands(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestWriteAcquisitionEscapesAndOrders(t *testing.T) {
	got := string(WriteAcquisition("2026-09-25", map[int][]Acquisition{
		180001: {{Kind: "other", Text: `Say "hi"`, Past: true, CN: true}},
		10001:  {{Kind: "craft", Text: "Craft", Recipe: "Time Diary", Cost: []Cost{{Amount: 2, Unit: "Material"}}}},
		10002:  nil,
	}))
	want := `{"version":"2026-09-25","items":{"10001":[{"k":"craft","t":"Craft","cost":[[2,"Material"]],"recipe":"Time Diary"}],"180001":[{"k":"other","t":"Say \"hi\"","past":1,"cn":1}]}}`
	if got != want {
		t.Errorf("\n got %s\nwant %s", got, want)
	}
}
