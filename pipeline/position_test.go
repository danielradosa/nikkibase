package pipeline

import (
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

func TestEnumerationIsTheGames(t *testing.T) {
	if len(subSlots) != 34 {
		t.Fatalf("%d wearable places, want 34", len(subSlots))
	}
	perSlot := map[scoring.Slot]int{}
	seen := map[string]bool{}
	for _, s := range subSlots {
		if seen[s.Name] {
			t.Errorf("%q is listed twice", s.Name)
		}
		seen[s.Name] = true
		if s.Display == "" {
			t.Errorf("%q has nothing to show a player", s.Name)
		}
		perSlot[s.Slot]++
	}
	for slot, want := range wearablePlaces {
		if perSlot[slot] != want {
			t.Errorf("%s has %d places, want %d", SlotName(slot), perSlot[slot], want)
		}
	}
}

func TestAliasesResolve(t *testing.T) {
	for raw, name := range aliases {
		s, ok := byName[name]
		if !ok {
			t.Errorf("%q maps to %q, which is not a place", raw, name)
			continue
		}
		got, err := ResolvePosition(raw, s.Slot)
		if err != nil {
			t.Errorf("ResolvePosition(%q, %s): %v", raw, SlotName(s.Slot), err)
			continue
		}
		if got.Name != name {
			t.Errorf("%q resolved to %q, want %q", raw, got.Name, name)
		}
	}
	for _, s := range subSlots {
		if _, ok := aliases[s.Name]; !ok && s.Name != "spirit" {
			reached := false
			for _, to := range aliases {
				if to == s.Name {
					reached = true
					break
				}
			}
			if !reached {
				t.Errorf("no source can reach %q", s.Name)
			}
		}
	}
}

func TestOneNameAcrossSources(t *testing.T) {
	for _, c := range []struct{ fandom, aojiao, want string }{
		{"accessory:necklace", "饰品-颈饰·项链", "accessory_necklace"},
		{"accessory:earrings", "饰品-耳饰", "accessory_earrings"},
		{"accessory:gloves", "饰品-手饰·双", "accessory_hand_both"},
		{"accessory:handheld (both)", "饰品-手持·双", "accessory_handheld_both"},
		{"hosiery", "袜子-袜子", "hosiery_socks"},
		{"hosiery:leglet", "袜子-袜套", "hosiery_leglet"},
	} {
		for _, raw := range []string{c.fandom, c.aojiao} {
			slot := scoring.Accessory
			if byName[c.want].Slot == scoring.Hosiery {
				slot = scoring.Hosiery
			}
			got, err := ResolvePosition(raw, slot)
			if err != nil {
				t.Errorf("ResolvePosition(%q): %v", raw, err)
				continue
			}
			if got.Name != c.want {
				t.Errorf("%q resolved to %q, want %q", raw, got.Name, c.want)
			}
		}
	}
}

func TestUnknownPlaceIsRefused(t *testing.T) {
	if _, err := ResolvePosition("monocle", scoring.Accessory); err == nil {
		t.Error("an invented place was accepted")
	}
	if _, err := ResolvePosition("accessory:necklace", scoring.Hosiery); err == nil {
		t.Error("a necklace was accepted as hosiery")
	}
}

func TestCanonicaliseReportsEveryBadPlace(t *testing.T) {
	entries := []Entry{
		{Item: scoring.Item{ID: 1, Slot: scoring.Accessory}, Position: "accessory:necklace"},
		{Item: scoring.Item{ID: 2, Slot: scoring.Accessory}, Position: "monocle"},
		{Item: scoring.Item{ID: 3, Slot: scoring.Hosiery}, Position: "accessory:necklace"},
	}
	err := Canonicalise(entries)
	if err == nil {
		t.Fatal("two unplaceable items were accepted")
	}
	for _, want := range []string{"monocle", "necklace"} {
		if !contains(err.Error(), want) {
			t.Errorf("the error does not name %q: %v", want, err)
		}
	}
	if entries[0].Position != "accessory_necklace" {
		t.Errorf("a resolvable item was left at %q", entries[0].Position)
	}
}

func TestExclusionGroups(t *testing.T) {
	groups := map[string][]string{}
	for _, s := range subSlots {
		if s.Group != "" {
			groups[s.Group] = append(groups[s.Group], s.Name)
		}
	}
	if len(groups[GroupHandheld]) != 3 {
		t.Errorf("handheld holds %v, want the three hand-held places", groups[GroupHandheld])
	}
	if len(groups[GroupTorso]) != 3 {
		t.Errorf("torso holds %v, want dress, top and bottom", groups[GroupTorso])
	}
	if GroupIndex("") != 0 {
		t.Error("an ungrouped place must have group 0")
	}
	if GroupIndex(GroupHandheld) == GroupIndex(GroupTorso) {
		t.Error("the two groups share a number")
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
