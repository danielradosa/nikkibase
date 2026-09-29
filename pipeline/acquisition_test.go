package pipeline

import (
	"slices"
	"strings"
	"testing"
)

func TestMergeAcquisitionTakesTheWikiThenThePackedTableThenTheSuitPage(t *testing.T) {
	cat := AcquisitionCatalogue{Names: map[int]string{1: "A", 2: "B", 3: "C", 4: "D"}}
	wiki := map[int][]Acquisition{1: {{Kind: "store", Text: "Clothes Store"}}, 9: {{Kind: "store", Text: "Clothes Store"}}}
	packed := map[int][]Acquisition{
		1: {{Kind: "event", Text: "Limited event", CN: true}},
		2: {{Kind: "craft", Text: "Crafting", CN: true}},
	}
	suits := map[int][]Acquisition{2: {{Kind: "recharge", Text: "Recharge"}}, 3: {{Kind: "recharge", Text: "Recharge"}}}
	got, stats := MergeAcquisition(cat, WikiAcquisition{Items: wiki, Suits: suits}, packed)
	want := `{"version":"v","items":{"1":[{"k":"store","t":"Clothes Store"}],"2":[{"k":"craft","t":"Crafting","cn":1}],"3":[{"k":"recharge","t":"Recharge"}]}}`
	if out := string(WriteAcquisition("v", got)); out != want {
		t.Errorf("\n got %s\nwant %s", out, want)
	}
	if stats != (AcquisitionStats{Catalogue: 4, Covered: 3, FromWiki: 1, FromPacked: 1, FromSuits: 1}) {
		t.Errorf("stats = %+v", stats)
	}
}

func TestMergeAcquisitionSharpensVagueLinesFromTheOtherSources(t *testing.T) {
	cat := AcquisitionCatalogue{Names: map[int]string{1: "A", 2: "B", 3: "C", 4: "D", 5: "E"}}
	wiki := map[int][]Acquisition{4: {{Kind: "evolve", Text: "Evolution"}, {Kind: "store", Text: "Clothes Store"}}}
	packed := map[int][]Acquisition{
		1: {{Kind: "event", Text: "Limited event", CN: true}, {Kind: "recharge", Text: "Recharge", CN: true}},
		2: {{Kind: "event", Text: "Limited event", CN: true}, {Kind: "craft", Text: "Crafting", CN: true}},
		3: {{Kind: "event", Text: "Limited event", CN: true}},
		4: {{Kind: "evolve", Text: "Evolve: E", From: []Ingredient{{ID: 5, Qty: 2}}, CN: true}},
	}
	suits := map[int][]Acquisition{
		1: {{Kind: "event", Text: "Starry Night event", Past: true}, {Kind: "store", Text: "Clothes Store"}, {Kind: "pavilion", Text: "Wish Gate"}},
		2: {{Kind: "event", Text: "Starry Night event", Past: true}},
		3: {{Kind: "recharge", Text: "Recharge"}},
	}
	got, stats := MergeAcquisition(cat, WikiAcquisition{Items: wiki, Suits: suits}, packed)
	want := `{"version":"v","items":{` +
		`"1":[{"k":"event","t":"Starry Night event","past":1},{"k":"store","t":"Clothes Store"}],` +
		`"2":[{"k":"event","t":"Limited event","cn":1},{"k":"craft","t":"Crafting","cn":1}],` +
		`"3":[{"k":"event","t":"Limited event","cn":1}],` +
		`"4":[{"k":"evolve","t":"Evolve: E","from":[[5,2]],"cn":1},{"k":"store","t":"Clothes Store"}]}}`
	if out := string(WriteAcquisition("v", got)); out != want {
		t.Errorf("\n got %s\nwant %s", out, want)
	}
	if stats.Named != 1 || stats.Based != 1 || stats.FromWiki != 1 || stats.FromPacked != 3 {
		t.Errorf("stats = %+v", stats)
	}
}

func TestMergeAcquisitionNamesAllVagueItemsFromTheirEventPage(t *testing.T) {
	cat := AcquisitionCatalogue{Names: map[int]string{1: "A", 2: "B", 3: "C", 4: "D", 5: "E"}}
	gala := []Acquisition{{Kind: "event", Text: "Starry Gala event", Past: true}}
	hundred := []Acquisition{{Kind: "recharge", Text: "$100 Recharge event", Past: true}}
	wiki := WikiAcquisition{
		Items:  map[int][]Acquisition{1: {{Kind: "recharge", Text: "Recharge", Past: true}}},
		Suits:  map[int][]Acquisition{3: {{Kind: "event", Text: "Crystal event", Past: true}}},
		Events: map[int][]Acquisition{1: hundred, 2: gala, 3: gala, 4: gala, 5: gala},
	}
	packed := map[int][]Acquisition{
		2: {{Kind: "event", Text: "Limited event", CN: true}, {Kind: "recharge", Text: "Event recharge", CN: true}},
		3: {{Kind: "event", Text: "Limited event", CN: true}},
		4: {{Kind: "store", Text: "Clothes Store", CN: true}},
		5: {{Kind: "recharge", Text: "Recharge", CN: true}},
	}
	got, stats := MergeAcquisition(cat, wiki, packed)
	want := `{"version":"v","items":{` +
		`"1":[{"k":"recharge","t":"$100 Recharge event","past":1}],` +
		`"2":[{"k":"event","t":"Starry Gala event","past":1},{"k":"recharge","t":"Event recharge","cn":1}],` +
		`"3":[{"k":"event","t":"Crystal event","past":1}],` +
		`"4":[{"k":"store","t":"Clothes Store","cn":1}],` +
		`"5":[{"k":"recharge","t":"Recharge","cn":1}]}}`
	if out := string(WriteAcquisition("v", got)); out != want {
		t.Errorf("\n got %s\nwant %s", out, want)
	}
	if stats.Events != 2 || stats.Named != 1 || stats.Covered != 5 {
		t.Errorf("stats = %+v, want 2 named by their event page, each vague line by a line of its kind, after 1 named by its suit page", stats)
	}
}

func TestMergeAcquisitionNamesPlainRechargeItemsFromTheirReruns(t *testing.T) {
	cat := AcquisitionCatalogue{Names: map[int]string{1: "A", 2: "B", 3: "C", 4: "D", 5: "E"}}
	reruns := []Acquisition{{Kind: "recharge", Text: "Abyssal Island (last Mar 2024)", Past: true}, {Kind: "recharge", Text: "One-Dollar Sale", Past: true}}
	wiki := WikiAcquisition{
		Items:  map[int][]Acquisition{5: {{Kind: "recharge", Text: "Abyssal Island", Past: true}}},
		Events: map[int][]Acquisition{4: {{Kind: "recharge", Text: "$100 Recharge event", Past: true}}},
		Reruns: map[int][]Acquisition{1: reruns, 2: reruns, 3: reruns, 4: reruns, 5: reruns},
	}
	packed := map[int][]Acquisition{
		1: {{Kind: "recharge", Text: "Recharge", CN: true}},
		2: {{Kind: "recharge", Text: "Event recharge", CN: true}, {Kind: "recharge", Text: "Recharge", CN: true}},
		3: {{Kind: "recharge", Text: "Recharge", CN: true}, {Kind: "event", Text: "Limited event", CN: true}},
		4: {{Kind: "recharge", Text: "Recharge", CN: true}},
	}
	got, stats := MergeAcquisition(cat, wiki, packed)
	want := `{"version":"v","items":{` +
		`"1":[{"k":"recharge","t":"Abyssal Island (last Mar 2024)","past":1},{"k":"recharge","t":"One-Dollar Sale","past":1}],` +
		`"2":[{"k":"recharge","t":"Event recharge","cn":1},{"k":"recharge","t":"Abyssal Island (last Mar 2024)","past":1},{"k":"recharge","t":"One-Dollar Sale","past":1}],` +
		`"3":[{"k":"recharge","t":"Recharge","cn":1},{"k":"event","t":"Limited event","cn":1}],` +
		`"4":[{"k":"recharge","t":"$100 Recharge event","past":1}],` +
		`"5":[{"k":"recharge","t":"Abyssal Island","past":1}]}}`
	if out := string(WriteAcquisition("v", got)); out != want {
		t.Errorf("\n got %s\nwant %s", out, want)
	}
	if stats.Reruns != 2 || stats.Events != 1 {
		t.Errorf("stats = %+v, want 2 named by their reruns after 1 named by its event page", stats)
	}
}

func TestEventAndRerunLinesAreHeldBetweenTheirFloorsAndCeilings(t *testing.T) {
	at := Coverage{EventItems: 2, MaxEventItems: 2, RerunItems: 3, MaxRerunItems: 3}
	if v := CheckAcquisitionFallbacks(AcquisitionStats{Events: 2, Reruns: 3}, at); len(v) != 0 {
		t.Errorf("at the floors and ceilings: %v", v)
	}
	if v := CheckAcquisitionFallbacks(AcquisitionStats{Reruns: 3}, at); len(v) != 1 || !strings.Contains(v[0], "0 items") || !strings.Contains(v[0], "floor is 2") {
		t.Errorf("events below their floor: %v", v)
	}
	if v := CheckAcquisitionFallbacks(AcquisitionStats{Events: 2, Reruns: 1}, at); len(v) != 1 || !strings.Contains(v[0], "1 items") || !strings.Contains(v[0], "floor is 3") {
		t.Errorf("reruns below their floor: %v", v)
	}
	if v := CheckAcquisitionFallbacks(AcquisitionStats{Events: 3}, Coverage{MaxEventItems: 2}); len(v) != 1 || !strings.Contains(v[0], "3 items") {
		t.Errorf("events above their ceiling: %v", v)
	}
	if v := CheckAcquisitionFallbacks(AcquisitionStats{Reruns: 4}, Coverage{MaxRerunItems: 3}); len(v) != 1 || !strings.Contains(v[0], "4 items") {
		t.Errorf("reruns above their ceiling: %v", v)
	}
}

func TestCategoriesNameAPlainRechargeLine(t *testing.T) {
	plain := []Acquisition{{Kind: "recharge", Text: "Recharge"}}
	for name, c := range map[string]struct {
		list       []Acquisition
		categories []string
		want       []Acquisition
	}{
		"abyssal":  {plain, []string{"Clothing", "abyssal island"}, []Acquisition{{Kind: "recharge", Text: "Abyssal Island", Past: true}}},
		"lasting":  {plain, []string{"First Recharge Giftpack"}, []Acquisition{{Kind: "recharge", Text: "First Recharge Giftpack"}}},
		"no match": {plain, []string{"Clothing"}, plain},
		"specific": {[]Acquisition{plain[0], {Kind: "store", Text: "Clothes Store"}}, []string{"Abyssal Island"}, []Acquisition{plain[0], {Kind: "store", Text: "Clothes Store"}}},
		"no plain": {[]Acquisition{{Kind: "event", Text: "Limited event"}}, []string{"Abyssal Island"}, []Acquisition{{Kind: "event", Text: "Limited event"}}},
	} {
		same := func(a, b Acquisition) bool { return sameAcquisition(a, b) && a.Past == b.Past }
		if got := categoryRecharge(c.list, c.categories); !slices.EqualFunc(got, c.want, same) {
			t.Errorf("%s: %+v, want %+v", name, got, c.want)
		}
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
