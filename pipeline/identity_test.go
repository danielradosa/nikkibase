package pipeline

import (
	"maps"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

func TestCommittedCorrectionsParse(t *testing.T) {
	raw, err := os.ReadFile("../data/id-corrections.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadIDCorrections(raw); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile("../data/stage-corrections.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadStageCorrections(raw); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadAcknowledged(raw); err != nil {
		t.Fatal(err)
	}
}

func TestCommittedCorrectionsSettleTheMisnumberedPages(t *testing.T) {
	raw, err := os.ReadFile("../data/id-corrections.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := ReadIDCorrections(raw)
	if err != nil {
		t.Fatal(err)
	}
	if c.Owner[51204] != "Guardian of Time - Glow" || c.Drop[51204] != "Voice of Hunting - Rare" || c.RealID[51204] != 50540 {
		t.Errorf("51204 keeps %q and drops %q to %d; want Guardian of Time - Glow, and Voice of Hunting - Rare read at 50540",
			c.Owner[51204], c.Drop[51204], c.RealID[51204])
	}
	for id, want := range map[int][2]string{89121: {"Memory of Summer", "Depicting Summer"}, 89126: {"Depicting Summer", "Memory of Summer"}} {
		if got := c.nameOf(id, want[0]); got != want[1] {
			t.Errorf("the wiki's %q on %d is named %q, want %q", want[0], id, got, want[1])
		}
	}
}

func TestCommittedNightFormsAddOnlyTheirQualifier(t *testing.T) {
	raw, err := os.ReadFile("../data/id-corrections.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := ReadIDCorrections(raw)
	if err != nil {
		t.Fatal(err)
	}
	nights := 0
	for id, n := range c.Shown {
		plain, ok := strings.CutSuffix(n.Name, " (Night)")
		if !ok {
			continue
		}
		nights++
		if n.Was != plain && n.Was != plain+"/Night" {
			t.Errorf("%d: %q becomes %q, which changes more than the night qualifier", id, n.Was, n.Name)
		}
	}
	if nights != 27 {
		t.Errorf("%d night forms, want 27", nights)
	}
}

func TestReadIDCorrectionsRefusesBadEntries(t *testing.T) {
	for name, doc := range map[string]string{
		"own ID as the dropped garment's": `{"duplicates": [{"id": 51204, "keep": "A", "drop": "B", "correctIdForDropped": 51204, "basis": "b"}]}`,
		"negative correct ID":             `{"duplicates": [{"id": 51204, "keep": "A", "drop": "B", "correctIdForDropped": -1, "basis": "b"}]}`,
		"name without basis":              `{"nameOverrides": [{"id": 89121, "name": "A", "was": "B"}]}`,
		"name without an ID":              `{"nameOverrides": [{"name": "A", "was": "B", "basis": "b"}]}`,
		"name without the old name":       `{"nameOverrides": [{"id": 89121, "name": "A", "basis": "b"}]}`,
		"name without the new name":       `{"nameOverrides": [{"id": 89121, "was": "B", "basis": "b"}]}`,
		"name to the same garment":        `{"nameOverrides": [{"id": 89121, "name": "Memory of Summer", "was": "memory of summer", "basis": "b"}]}`,
		"renamed twice":                   `{"nameOverrides": [{"id": 89121, "name": "A", "was": "B", "basis": "b"}, {"id": 89121, "name": "C", "was": "D", "basis": "b"}]}`,
		"shown name without basis":        `{"displayNames": [{"id": 20001, "name": "A", "was": "B"}]}`,
		"shown name without an ID":        `{"displayNames": [{"name": "A", "was": "B", "basis": "b"}]}`,
		"shown name without the old name": `{"displayNames": [{"id": 20001, "name": "A", "basis": "b"}]}`,
		"shown name without the new name": `{"displayNames": [{"id": 20001, "was": "B", "basis": "b"}]}`,
		"shown name unchanged":            `{"displayNames": [{"id": 20001, "name": "A", "was": "A", "basis": "b"}]}`,
		"shown name given twice":          `{"displayNames": [{"id": 20001, "name": "A", "was": "B", "basis": "b"}, {"id": 20001, "name": "C", "was": "B", "basis": "b"}]}`,
	} {
		if _, err := ReadIDCorrections([]byte(doc)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestMisnumberedPageMovesToItsOwnID(t *testing.T) {
	c := corrections(t, `{"duplicates": [{"id": 51204, "keep": "Guardian of Time - Glow",
		"drop": "Voice of Hunting - Rare", "correctIdForDropped": 50540, "basis": "two sources agree"}]}`)
	page := [5]string{"B", "SS", "A", "SS", "A"}
	wiki := []Entry{graded(51204, scoring.Bottom, "Voice of Hunting - Rare", "bottom", page)}
	table := graded(50540, scoring.Bottom, "Voice of Hunting · Rare", "bottom", page)
	table.Rarity, table.Item.Tags = 4, []int{7}
	guardian := graded(51204, scoring.Bottom, "Guardian of Time - Glow", "bottom", [5]string{"A", "S", "S", "B", "S"})

	got, err := c.Apply(MergeEntries(c.DropDuplicates(wiki), []Entry{table, guardian}))
	if err != nil {
		t.Fatal(err)
	}
	byID := map[int]Entry{}
	for _, e := range got {
		byID[e.Item.ID] = e
	}
	if len(got) != 2 {
		t.Fatalf("%d rows, want 50540 and 51204: %+v", len(got), got)
	}
	if e := byID[50540]; e.Name != "Voice of Hunting - Rare" || e.Rarity != 4 || !slices.Equal(e.Item.Tags, []int{7}) {
		t.Errorf("50540 is %q, rarity %d, tags %v; want the page's row with the table's rarity 4 and tags [7]", e.Name, e.Rarity, e.Item.Tags)
	}
	if e := byID[51204]; e.Name != "Guardian of Time - Glow" || e.Grades != guardian.Grades {
		t.Errorf("51204 is %q %v, want the table's Guardian of Time - Glow", e.Name, e.Grades)
	}
	if v := checkIdentity(got); v != nil {
		t.Errorf("the moved row fails the identity invariant: %v", v)
	}
}

func TestMisnumberedPageStaysOutWhereItCannotMove(t *testing.T) {
	grades := [5]string{"A", "A", "A", "A", "A"}
	for name, c := range map[string]struct {
		doc  string
		rows []Entry
	}{
		"no ID of its own": {
			`{"duplicates": [{"id": 51204, "keep": "Keep", "drop": "Page", "basis": "b"}]}`,
			[]Entry{graded(51204, scoring.Bottom, "Page", "bottom", grades)},
		},
		"its ID is in another slot": {
			`{"duplicates": [{"id": 31367, "keep": "Keep", "drop": "Page", "correctIdForDropped": 41367, "basis": "b"}]}`,
			[]Entry{graded(31367, scoring.Coat, "Page", "coat", grades)},
		},
		"its source already has a row there": {
			`{"duplicates": [{"id": 51204, "keep": "Keep", "drop": "Page", "correctIdForDropped": 50540, "basis": "b"}]}`,
			[]Entry{graded(51204, scoring.Bottom, "Page", "bottom", grades), graded(50540, scoring.Bottom, "Other", "bottom", grades)},
		},
		"its ID is left out": {
			`{"duplicates": [{"id": 51204, "keep": "Keep", "drop": "Page", "correctIdForDropped": 50540, "basis": "b"}],
			  "excluded": [{"id": 50540, "reason": "r", "effect": "e"}]}`,
			[]Entry{graded(51204, scoring.Bottom, "Page", "bottom", grades)},
		},
	} {
		fix := corrections(t, c.doc)
		early, err := fix.Apply(fix.DropDuplicates(slices.Clone(c.rows)))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		late, err := fix.Apply(slices.Clone(c.rows))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for pass, got := range map[string][]Entry{"DropDuplicates": early, "Apply": late} {
			for _, e := range got {
				if e.Name == "Page" {
					t.Errorf("%s, %s: the page's row is kept at %d", name, pass, e.Item.ID)
				}
			}
		}
	}
}

func TestSwappedPagesBothMove(t *testing.T) {
	c := corrections(t, `{"duplicates": [
		{"id": 10001, "keep": "One", "drop": "Two", "correctIdForDropped": 10002, "basis": "b"},
		{"id": 10002, "keep": "Two", "drop": "One", "correctIdForDropped": 10001, "basis": "b"}]}`)
	grades := [5]string{"A", "A", "A", "A", "A"}
	got, err := c.Apply(c.DropDuplicates([]Entry{
		graded(10001, scoring.Hair, "Two", "hair", grades),
		graded(10002, scoring.Hair, "One", "hair", grades),
	}))
	if err != nil {
		t.Fatal(err)
	}
	names := map[int]string{}
	for _, e := range got {
		names[e.Item.ID] = e.Name
	}
	if len(got) != 2 || names[10001] != "One" || names[10002] != "Two" {
		t.Errorf("got %v, want One at 10001 and Two at 10002", names)
	}
}

func TestNameOverrideRenamesOnlyThePageItNames(t *testing.T) {
	c := corrections(t, `{"nameOverrides": [{"id": 89121, "name": "Depicting Summer", "was": "Memory of Summer", "basis": "b"}]}`)
	grades := [5]string{"A", "SS", "A", "SS", "A"}
	got, err := c.Apply([]Entry{
		graded(89121, scoring.Accessory, "Memory of Summer", "accessory_earrings", grades),
		graded(89126, scoring.Accessory, "Memory of Summer", "accessory_earrings", grades),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Name != "Depicting Summer" || got[1].Name != "Memory of Summer" {
		t.Errorf("names %q and %q, want Depicting Summer at 89121 and 89126 left alone", got[0].Name, got[1].Name)
	}
	if got[0].Item.Stats != graded(89121, scoring.Accessory, "", "", grades).Item.Stats || got[0].Position != "accessory_earrings" {
		t.Errorf("the renamed row changed more than its name: %+v", got[0])
	}
	other := corrections(t, `{"nameOverrides": [{"id": 89121, "name": "Depicting Summer", "was": "Memory of Summer", "basis": "b"}]}`)
	kept, err := other.Apply([]Entry{graded(89121, scoring.Accessory, "Something Else", "accessory_earrings", grades)})
	if err != nil {
		t.Fatal(err)
	}
	if kept[0].Name != "Something Else" {
		t.Errorf("a row the override does not name became %q", kept[0].Name)
	}
}

func TestDisplayNamesReplaceTheNamesTheSourcesGive(t *testing.T) {
	c := corrections(t, `{"displayNames": [
		{"id": 20001, "name": "Test Gown", "was": "Tset Gown", "basis": "b"},
		{"id": 180001, "name": "Far Away Test · Winter", "was": "Far Test · Winter", "basis": "b"},
		{"id": 30003, "name": "Named Only (Coat)", "was": "Named Only", "basis": "b"},
		{"id": 30004, "name": "Named · Only", "was": "Named · Alone", "basis": "b"}]}`)
	entries := []Entry{
		{Item: scoring.Item{ID: 20001, Slot: scoring.Dress}, Name: "Tset Gown"},
		{Item: scoring.Item{ID: 20002, Slot: scoring.Dress}, Name: "Tset Gown"},
		{Item: scoring.Item{ID: 180001, Slot: scoring.Accessory}, Name: "Far Test·Winter"},
	}
	names := ItemNames{Calc: map[int]string{30003: "Named Only", 30004: "Named·Alone", 20002: "Other"}}
	got, err := c.DisplayNames(entries, names)
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]string{20001: "Test Gown", 180001: "Far Away Test · Winter", 30003: "Named Only (Coat)", 30004: "Named · Only"}
	if !maps.Equal(got, want) {
		t.Errorf("shown names %v, want %v", got, want)
	}
	for name, rows := range map[string][]Entry{
		"a name the sources no longer give": {
			{Item: scoring.Item{ID: 20001, Slot: scoring.Dress}, Name: "Test Gown"},
			{Item: scoring.Item{ID: 180001, Slot: scoring.Accessory}, Name: "Far Test·Winter"},
		},
		"an item not in the catalogue": {
			{Item: scoring.Item{ID: 180001, Slot: scoring.Accessory}, Name: "Far Test·Winter"},
		},
	} {
		if _, err := c.DisplayNames(rows, names); err == nil || !strings.Contains(err.Error(), "20001") {
			t.Errorf("%s: err = %v, want a refusal naming 20001", name, err)
		}
	}
}

func graded(id int, slot scoring.Slot, name, place string, grades [5]string) Entry {
	e := Entry{Item: scoring.Item{ID: id, Slot: slot}, Name: name, Position: place, Grades: grades}
	for p, g := range grades {
		e.Item.Stats[p] = Stat(g, slot)
	}
	return e
}

func corrections(t *testing.T, doc string) *IDCorrections {
	t.Helper()
	c, err := ReadIDCorrections([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestSlotOverrideRecomputesStats(t *testing.T) {
	c := corrections(t, `{"slotOverrides": [{"id": 30001, "name": "Test Coat", "slot": "coat",
		"was": "top", "basis": "the ID", "position": "coat", "positionBasis": "one place"}]}`)
	grades := [5]string{"S", "A", "B", "C", "SS"}
	got, err := c.Apply([]Entry{graded(30001, scoring.Top, "Test Coat", "top", grades)})
	if err != nil {
		t.Fatal(err)
	}
	e := got[0]
	if e.Item.Slot != scoring.Coat || e.Position != "coat" {
		t.Fatalf("slot %s, place %q; want coat, coat", SlotName(e.Item.Slot), e.Position)
	}
	for p, g := range grades {
		if want := Stat(g, scoring.Coat); e.Item.Stats[p] != want {
			t.Errorf("pair %d: %d, and grade %s on a coat is %d", p, e.Item.Stats[p], g, want)
		}
	}
	if v := checkStats(got); v != nil {
		t.Errorf("the corrected item fails the stats invariant: %v", v)
	}
}

func TestDuplicateGoesBeforeTheMerge(t *testing.T) {
	c := corrections(t, `{"duplicates": [{"id": 11470, "keep": "Wind and Moon (Hair)",
		"drop": "Warm as Spring", "basis": "two sources agree"}]}`)
	grades := [5]string{"A", "A", "A", "A", "A"}
	wiki := func() []Entry {
		return []Entry{
			graded(11470, scoring.Hair, "Wind and Moon (Hair)", "hair", grades),
			graded(11470, scoring.Hair, "Warm as Spring", "hair", grades),
		}
	}
	table := graded(11470, scoring.Hair, "Wind and Moon", "hair", grades)
	table.Rarity, table.Suit = 5, "Wind and Moon"

	late, err := c.Apply(MergeEntries(wiki(), []Entry{table}))
	if err != nil {
		t.Fatal(err)
	}
	if late[0].Rarity != 0 {
		t.Fatalf("resolving after the merge kept rarity %d; the fixture no longer shows the problem", late[0].Rarity)
	}

	early, err := c.Apply(MergeEntries(c.DropDuplicates(wiki()), []Entry{table}))
	if err != nil {
		t.Fatal(err)
	}
	if len(early) != 1 || early[0].Name != "Wind and Moon (Hair)" {
		t.Fatalf("kept %+v, want only Wind and Moon (Hair)", early)
	}
	if early[0].Rarity != 5 || early[0].Suit != "Wind and Moon" {
		t.Errorf("rarity %d, suit %q; want the fallback row's 5 and Wind and Moon", early[0].Rarity, early[0].Suit)
	}
}

func TestOwnerIsKeptWhicheverComesFirst(t *testing.T) {
	c := corrections(t, `{"duplicates": [{"id": 11470, "keep": "Wind and Moon (Hair)",
		"drop": "Warm as Spring", "basis": "two sources agree"}]}`)
	grades := [5]string{"A", "A", "A", "A", "A"}
	for name, rows := range map[string][]Entry{
		"owner first": {
			graded(11470, scoring.Hair, "Wind and Moon (Hair)", "hair", grades),
			graded(11470, scoring.Hair, "Warm as Spring", "hair", grades),
		},
		"owner second": {
			graded(11470, scoring.Hair, "Warm as Spring", "hair", grades),
			graded(11470, scoring.Hair, "Wind and Moon (Hair)", "hair", grades),
		},
		"unlisted row first": {
			graded(11470, scoring.Hair, "Wind & Moon", "hair", grades),
			graded(11470, scoring.Hair, "Wind and Moon (Hair)", "hair", grades),
		},
	} {
		early := c.DropDuplicates(slices.Clone(rows))
		late, err := c.Apply(slices.Clone(rows))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for pass, got := range map[string][]Entry{"DropDuplicates": early, "Apply": late} {
			if len(got) != 1 || got[0].Name != "Wind and Moon (Hair)" {
				t.Errorf("%s, %s kept %v", name, pass, got)
			}
		}
	}
}

func TestLoneMisnumberedRowGoesBeforeTheMerge(t *testing.T) {
	c := corrections(t, `{"duplicates": [{"id": 89667, "keep": "Glazed Feather",
		"drop": "Blade of Judgment (Polar Day Echo)", "basis": "two sources agree"}]}`)
	wiki := []Entry{graded(89667, scoring.Accessory, "Blade of Judgment (Polar Day Echo)",
		"accessory_handheld_right", [5]string{"A", "A", "SS", "SS", "A"})}
	table := graded(89667, scoring.Accessory, "Glazed Feather", "accessory_headwear",
		[5]string{"S", "SS", "S", "A", "A"})

	got, err := c.Apply(MergeEntries(c.DropDuplicates(wiki), []Entry{table}))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "Glazed Feather" || got[0].Position != "accessory_headwear" {
		t.Errorf("kept %+v, want only Glazed Feather in accessory_headwear", got)
	}
}

func TestDropDuplicatesLeavesUnlistedClashes(t *testing.T) {
	c := corrections(t, `{}`)
	grades := [5]string{"B", "B", "B", "B", "B"}
	in := []Entry{
		graded(10001, scoring.Hair, "One", "hair", grades),
		graded(10001, scoring.Hair, "Two", "hair", grades),
	}
	if got := c.DropDuplicates(in); len(got) != 2 {
		t.Errorf("an unlisted clash was settled early: %+v", got)
	}
	if _, err := c.Apply(in); err == nil {
		t.Error("Apply accepted an ID naming two garments")
	}
}

func TestCheckGarments(t *testing.T) {
	a := [5]string{"A", "A", "A", "A", "A"}
	b := [5]string{"S", "B", "S", "B", "S"}
	entries := []Entry{
		graded(20003, scoring.Dress, "Test Page Gown", "dress", a),
		graded(82829, scoring.Accessory, "Test Pendant/Night", "accessory_earrings", a),
		graded(71100, scoring.Shoes, "Tester's Whisper - Hidden", "shoes", a),
		graded(10001, scoring.Hair, "Same Grades", "hair", a),
		graded(10003, scoring.Hair, "Wiki Name", "hair", a),
		graded(10004, scoring.Hair, "Unbacked", "hair", a),
		graded(10005, scoring.Hair, "表中名", "hair", b),
	}
	tables := map[int]Placed{
		20003: {Name: "真礼服", Grades: b},
		82829: {Name: "测试吊坠", Grades: b},
		71100: {Name: "测试低语·隐", Grades: b},
		10001: {Name: "同评", Grades: a},
		10003: {Name: "表名", Grades: b},
		10004: {Name: "表名", Grades: b},
		10005: {Name: "表中名", Grades: b},
	}
	names := map[int]string{
		20003: "Test Real Gown",
		82829: "Test Pendant",
		71100: "Tester’s Whisper · Hidden",
		10003: "Wiki Name",
	}
	got := CheckGarments(entries, tables, names)
	if len(got) != 2 || !strings.Contains(got[0], "10004") || !strings.Contains(got[1], "20003") {
		t.Errorf("CheckGarments = %v, want 10004 and 20003", got)
	}
}

func TestPlacesOfDropsASourceThatContradictsItself(t *testing.T) {
	grades := [5]string{"B", "B", "B", "B", "B"}
	got := PlacesOf([]Entry{
		graded(80001, scoring.Accessory, "x", "accessory_scarf", grades),
		graded(80001, scoring.Accessory, "x", "accessory_waist", grades),
		graded(80002, scoring.Accessory, "y", "accessory_scarf", grades),
		graded(80002, scoring.Accessory, "y", "accessory_scarf", grades),
	})
	if _, ok := got[80001]; ok {
		t.Error("an ID filed in two places was kept")
	}
	if p := got[80002]; p.Slot != scoring.Accessory || p.Position != "accessory_scarf" || p.Name != "y" {
		t.Errorf("80002 = %+v", p)
	}
}
