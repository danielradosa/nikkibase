package pipeline

import (
	"maps"
	"os"
	"strings"
	"testing"
)

func TestTheCommittedSuitPartAliasesRead(t *testing.T) {
	raw, err := os.ReadFile("../data/suit-part-aliases.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSuitPartAliases(raw); err != nil {
		t.Fatal(err)
	}
}

func TestSuitPartAliasesGiveEveryField(t *testing.T) {
	line := `{"suit": "Goldfish Girl", "part": "Goldfish's Summer Fantasy", "id": 21093, "basis": "list page"}`
	for name, c := range map[string]struct{ doc, want string }{
		"twice":    {`{"aliases": [` + line + `, ` + line + `]}`, "listed twice"},
		"no basis": {`{"aliases": [{"suit": "S", "part": "P", "id": 1}]}`, "basis"},
		"no id":    {`{"aliases": [{"suit": "S", "part": "P", "basis": "b"}]}`, "basis"},
		"no part":  {`{"aliases": [{"suit": "S", "id": 1, "basis": "b"}]}`, "basis"},
	} {
		if _, err := ReadSuitPartAliases([]byte(c.doc)); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want one naming %q", name, err, c.want)
		}
	}
}

const goldfishDump = "<mediawiki>\n<page>\n  <title>Goldfish Girl</title>\n  <ns>0</ns>\n  <revision><text>{{Suit Infobox\n" +
	"|type = Collection Suit\n|how to obtain = [[Firework Fair]] event\n}}\n==Wardrobe==\n" +
	"{{Suit Part|Lonely Hanabi|type=Hair}}\n{{Suit Part|Goldfish's Summer Fantasy|type=Dress|v=2}}</text></revision>\n</page>\n</mediawiki>"

func goldfishCatalogue(aliases ...SuitPartAlias) AcquisitionCatalogue {
	return AcquisitionCatalogue{Names: map[int]string{10001: "Lonely Hanabi", 21093: "Goldfish's Summer Fanstasy"}, PartAliases: aliases}
}

func TestASuitPartAliasPlacesWhatTheLookupCannot(t *testing.T) {
	got, stats, err := ParseFandomAcquisition(strings.NewReader(goldfishDump), nil, nil, goldfishCatalogue())
	if err != nil {
		t.Fatal(err)
	}
	if want := map[int]string{10001: "Goldfish Girl"}; !maps.Equal(got.SuitOf, want) || stats.UnknownParts != 1 {
		t.Fatalf("without the alias: suits %v and %d parts unmatched, want %v and the misspelt part", got.SuitOf, stats.UnknownParts, want)
	}
	alias := SuitPartAlias{Suit: "Goldfish Girl", Part: "Goldfish's Summer Fantasy", ID: 21093, Basis: "list page"}
	got, stats, err = ParseFandomAcquisition(strings.NewReader(goldfishDump), nil, nil, goldfishCatalogue(alias))
	if err != nil {
		t.Fatal(err)
	}
	if want := map[int]string{10001: "Goldfish Girl", 21093: "Goldfish Girl"}; !maps.Equal(got.SuitOf, want) || stats.UnknownParts != 0 {
		t.Errorf("suits %v and %d parts unmatched, want %v and none", got.SuitOf, stats.UnknownParts, want)
	}
	if lines := got.Suits[21093]; len(lines) != 1 || lines[0].Text != "Firework Fair event" {
		t.Errorf("21093 from its suit page: %+v, want the Firework Fair event line", lines)
	}
}

func TestStaleSuitPartAliasesRefuseTheBuild(t *testing.T) {
	for name, c := range map[string]struct {
		alias SuitPartAlias
		want  string
	}{
		"gone":         {SuitPartAlias{Suit: "Goldfish Girl", Part: "Goldfish's Summer Fantasy", ID: 21094}, "21094 is not in the catalogue"},
		"unlisted":     {SuitPartAlias{Suit: "Goldfish Girl", Part: "Goldfish Summer Fantasy", ID: 21093}, "no longer lists"},
		"another suit": {SuitPartAlias{Suit: "Koi Girl", Part: "Goldfish's Summer Fantasy", ID: 21093}, "no longer lists"},
		"matched":      {SuitPartAlias{Suit: "Goldfish Girl", Part: "Lonely Hanabi", ID: 10001}, "matches an item without it"},
	} {
		_, _, err := ParseFandomAcquisition(strings.NewReader(goldfishDump), nil, nil, goldfishCatalogue(c.alias))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want one naming %q", name, err, c.want)
		}
	}
}
