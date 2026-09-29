package pipeline

import (
	"os"
	"strings"
	"testing"
)

func TestTheCommittedIngredientAliasesRead(t *testing.T) {
	raw, err := os.ReadFile("../data/ingredient-aliases.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadIngredientAliases(raw); err != nil {
		t.Fatal(err)
	}
}

func TestIngredientAliasesGiveEveryField(t *testing.T) {
	line := `{"page": "Horn of Cloud", "name": "Bow Braclet-Red", "id": 80784, "basis": "used to craft"}`
	for name, c := range map[string]struct{ doc, want string }{
		"twice":    {`{"aliases": [` + line + `, ` + line + `]}`, "listed twice"},
		"no basis": {`{"aliases": [{"page": "P", "name": "N", "id": 1}]}`, "basis"},
		"no id":    {`{"aliases": [{"page": "P", "name": "N", "basis": "b"}]}`, "basis"},
		"no name":  {`{"aliases": [{"page": "P", "id": 1, "basis": "b"}]}`, "basis"},
	} {
		if _, err := ReadIngredientAliases([]byte(c.doc)); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want one naming %q", name, err, c.want)
		}
	}
}

var hornDump = "<mediawiki>\n" +
	dumpPage("Horn of Cloud", clothing("Accessory, Ears", 3686, "[[Crafting]]"),
		"=== Crafted From: ===\n{{Recipe\n|item1 = Noble Necklace\n|item1_quantity = 5\n|item2 = Bow Braclet-Red\n|item2_quantity = 4\n|source = Time Diary\n}}") +
	"</mediawiki>"

func hornCatalogue(aliases ...IngredientAlias) AcquisitionCatalogue {
	return AcquisitionCatalogue{Names: map[int]string{83686: "Horn of Cloud", 80328: "Noble Necklace", 80784: "Bow Bracelet-Red"}, IngredientAliases: aliases}
}

func TestAnIngredientAliasLinksWhatTheLookupCannot(t *testing.T) {
	got, stats, err := ParseFandomAcquisition(strings.NewReader(hornDump), nil, nil, hornCatalogue())
	if err != nil {
		t.Fatal(err)
	}
	if lines := got.Items[83686]; len(lines) != 1 || len(lines[0].From) != 1 || stats.Unresolved != 1 {
		t.Fatalf("without the alias: %+v and %d names unmatched, want the misspelt ingredient left unlinked", lines, stats.Unresolved)
	}
	alias := IngredientAlias{Page: "Horn of Cloud", Name: "Bow Braclet-Red", ID: 80784, Basis: "used to craft"}
	got, stats, err = ParseFandomAcquisition(strings.NewReader(hornDump), nil, nil, hornCatalogue(alias))
	if err != nil {
		t.Fatal(err)
	}
	want := Acquisition{Kind: "craft", Text: "Craft: 5× Noble Necklace, 4× Bow Bracelet-Red",
		From: []Ingredient{{ID: 80328, Qty: 5}, {ID: 80784, Qty: 4}}, Recipe: "Time Diary"}
	if lines := got.Items[83686]; len(lines) != 1 || !sameAcquisition(lines[0], want) || stats.Unresolved != 0 {
		t.Errorf("%+v and %d names unmatched, want %+v and none", lines, stats.Unresolved, want)
	}
}

func TestStaleIngredientAliasesRefuseTheBuild(t *testing.T) {
	for name, c := range map[string]struct {
		alias IngredientAlias
		want  string
	}{
		"gone":         {IngredientAlias{Page: "Horn of Cloud", Name: "Bow Braclet-Red", ID: 80785}, "80785 is not in the catalogue"},
		"unnamed":      {IngredientAlias{Page: "Horn of Cloud", Name: "Bow Bracelt-Red", ID: 80784}, "no longer names"},
		"another page": {IngredientAlias{Page: "Tide of Fog", Name: "Bow Braclet-Red", ID: 80784}, "no longer names"},
		"matched":      {IngredientAlias{Page: "Horn of Cloud", Name: "Noble Necklace", ID: 80328}, "matches an item without it"},
	} {
		_, _, err := ParseFandomAcquisition(strings.NewReader(hornDump), nil, nil, hornCatalogue(c.alias))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want one naming %q", name, err, c.want)
		}
	}
}
