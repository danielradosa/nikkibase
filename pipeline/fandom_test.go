package pipeline

import (
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

const dump = `<mediawiki>
<page>
  <title>Nikki's Pinky</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|caption = Nikki's favorite hairstyle.
|type = Hair
|attributes = {{A|Si}}{{A|P}}
|rarity = {{H|2}}
|wardrobe nr = 1
}}
== Attributes ==
{{Attributes|Simple|S|Lively|A|Cute|A|Pure|A|Warm|A}}</text></revision>
</page>
<page>
  <title>Pearl Necklace</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Accessory, Necklace
|wardrobe nr = 1067
|style = {{S|Chi}}
}}
{{Attributes|G|SS|E|A|M|B|Se|C|Co|A}}</text></revision>
</page>
<page>
  <title>Talk about something</title>
  <ns>1</ns>
  <revision><text>{{Clothing|type = Hair|wardrobe nr = 5}}{{Attributes|Simple|S|Lively|A|Cute|A|Pure|A|Warm|A}}</text></revision>
</page>
<page>
  <title>Undocumented Thing</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Dress
|wardrobe nr = 99
}}
No attributes section at all.</text></revision>
</page>
<page>
  <title>Mistyped Thing</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Wingsuit
|wardrobe nr = 4
}}
{{Attributes|Simple|S|Lively|A|Cute|A|Pure|A|Warm|A}}</text></revision>
</page>
</mediawiki>`

func TestParseFandomDump(t *testing.T) {
	known := map[int]bool{10001: true, 81067: true}
	entries, stats, err := ParseFandomDump(strings.NewReader(dump), known)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(entries), entries)
	}
	if stats.Pages != 4 {
		t.Errorf("Pages = %d, want 4", stats.Pages)
	}
	if stats.NoGrades != 1 {
		t.Errorf("NoGrades = %d, want 1", stats.NoGrades)
	}
	if stats.UnknownAny != 1 {
		t.Errorf("UnknownAny = %d, want 1", stats.UnknownAny)
	}

	hair := entries[0]
	if hair.Name != "Nikki's Pinky" || hair.Item.ID != 10001 || hair.Item.Slot != scoring.Hair {
		t.Errorf("hair = %q/%d/%v, want Nikki's Pinky/10001/Hair", hair.Name, hair.Item.ID, hair.Item.Slot)
	}
	if hair.Item.Attrs[0] != scoring.Simple {
		t.Errorf("hair pair 0 = %d, want Simple", hair.Item.Attrs[0])
	}
	if got := hair.Item.Stats[0]; got != 70 {
		t.Errorf("hair Simple stat = %d, want 70", got)
	}

	necklace := entries[1]
	if necklace.Item.ID != 81067 || necklace.Item.Slot != scoring.Accessory {
		t.Errorf("necklace = %d/%v, want 81067/Accessory", necklace.Item.ID, necklace.Item.Slot)
	}
	if necklace.Position != "accessory:necklace" {
		t.Errorf("position = %q, want accessory:necklace", necklace.Position)
	}
	if len(necklace.Item.Tags) != 1 {
		t.Errorf("tags = %v, want one", necklace.Item.Tags)
	}
}

func TestAttributeSidesResolvePerPair(t *testing.T) {
	for _, tc := range []struct {
		pair int
		side string
		want int
	}{
		{0, "S", scoring.Simple},
		{3, "S", scoring.Sexy},
		{2, "C", scoring.Cute},
		{4, "C", scoring.Cool},
		{0, "G", scoring.Gorgeous},
		{1, "Elegance", scoring.Elegant},
		{1, "lively", scoring.Lively},
		{4, "W", scoring.Warm},
	} {
		got, ok := attributeCode(tc.pair, tc.side)
		if !ok || got != tc.want {
			t.Errorf("attributeCode(%d, %q) = %d/%v, want %d", tc.pair, tc.side, got, ok, tc.want)
		}
	}
	if _, ok := attributeCode(0, "Lively"); ok {
		t.Error("a side from the wrong pair should not resolve")
	}
}

func TestFandomIDs(t *testing.T) {
	for _, tc := range []struct {
		slot scoring.Slot
		n    int
		want int
	}{
		{scoring.Hair, 1, 10001},
		{scoring.Accessory, 1067, 81067},
		{scoring.Accessory, 13015, 183015},
		{scoring.Spirit, 106, 880106},
	} {
		if got := fandomID(tc.slot, tc.n); got != tc.want {
			t.Errorf("fandomID(%v, %d) = %d, want %d", tc.slot, tc.n, got, tc.want)
		}
	}
}
