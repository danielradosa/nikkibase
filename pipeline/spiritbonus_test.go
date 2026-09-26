package pipeline

import (
	"slices"
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

const spiritDump = `<mediawiki>
<page>
  <title>Holy Scepter</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Spirit
|rarity = {{H|4}}
|wardrobe nr = 156
}}
==Skill Bonus==
'''[[Rosy Scent]]:'''
*'''Level 3:''' {{A|G}} attribute rating increases by 800 points
*'''Level 4:''' {{A|G}} attribute rating increases by 1,200 points

==[[Evolution]]==
===Evolved from:===
*{{IconItem|Immaculate Crown}}

==Attributes==
{{Attributes|G|SS|E|A|M|B|Se|S|W|A}}</text></revision>
</page>
<page>
  <title>Winter Prayer</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Spirit
|wardrobe nr = 163
}}
==Skill Bonus==
'''[[Wish Feather]]:'''
*'''Level 2:''' {{A|L}} attribute rating increases by 500 points
*'''Level 3:''' {{A|L}} attribute rating increases by 800 points

==Evolution==</text></revision>
</page>
<page>
  <title>Jade Flute</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Spirit
|wardrobe nr = 87
}}
== Skill bonus ==
'''[[Phoenix Music]]:'''

* '''Level 2:''' {{A|M}} attribute rating increases by 500 points
* '''Level 3:''' {{A|M}} attribute rating increases by 800 points

== Evolution ==</text></revision>
</page>
<page>
  <title>Destined Twins</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Spirit
|wardrobe nr = 109
}}
== Skill Bonus ==
'''[[Fate String]]:'''
* '''Level 1:''' {{A|P}} increased by 200 points
* '''Level 2:''' {{A|P}} increased by 500 points

== Evolution ==</text></revision>
</page>
<page>
  <title>Ambition</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Spirit
|wardrobe nr = 73
}}
'''Ambition''' is a [[Spirits|spirit]] item.

==Appearance==
Two sheathed swords encased in a blue bubble.

==Evolution==</text></revision>
</page>
<page>
  <title>Nikki's Pinky</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Hair
|wardrobe nr = 1
}}
== Attributes ==
{{Attributes|Simple|S|Lively|A|Cute|A|Pure|A|Warm|A}}</text></revision>
</page>
</mediawiki>`

func TestParseSpiritBonuses(t *testing.T) {
	bonuses, stats, err := ParseSpiritBonuses(strings.NewReader(spiritDump), nil)
	if err != nil {
		t.Fatal(err)
	}

	if stats.Pages != 5 {
		t.Errorf("Pages = %d, want 5", stats.Pages)
	}
	if stats.Parsed != 4 {
		t.Errorf("Parsed = %d, want 4", stats.Parsed)
	}
	if stats.NoSection != 1 {
		t.Errorf("NoSection = %d, want 1", stats.NoSection)
	}
	if stats.NoLevels != 0 || stats.Rejected != 0 {
		t.Errorf("NoLevels = %d, Rejected = %d, want 0 and 0", stats.NoLevels, stats.Rejected)
	}

	for _, tc := range []struct {
		id   int
		want int
		why  string
	}{
		{880156, 1200, "top of a ladder, written 1,200"},
		{880163, 800, "ladder stops below the spirit's own level"},
		{880087, 800, "lower-case heading"},
		{880109, 500, `"increased by" rather than "increases by"`},
	} {
		if got := bonuses[tc.id]; got != tc.want {
			t.Errorf("bonus[%d] = %d, want %d (%s)", tc.id, got, tc.want, tc.why)
		}
	}
	if _, ok := bonuses[880073]; ok {
		t.Error("a page with no Skill Bonus section should yield no bonus")
	}
	if _, ok := bonuses[10001]; ok {
		t.Error("the hair page should not reach the spirit bonuses")
	}
}

func TestSpiritBonusTakesTopRung(t *testing.T) {
	const ladder = `==Skill Bonus==
'''[[Rosy Scent]]:'''
*'''Level 1:''' {{A|G}} attribute rating increases by 200 points
*'''Level 2:''' {{A|G}} attribute rating increases by 500 points
*'''Level 3:''' {{A|G}} attribute rating increases by 800 points<ref group="Note">Does not show on the attributes panel.</ref>

==Evolution==
*'''Level 5:''' this line is outside the section and must not be read`

	got, ok := spiritBonus(ladder)
	if !ok || got != 800 {
		t.Errorf("spiritBonus = %d/%v, want 800", got, ok)
	}
	if _, ok := spiritBonus("==Appearance==\nA floating yellow orb."); ok {
		t.Error("a page with no Skill Bonus section should not resolve")
	}
	if _, ok := spiritBonus("==Skill Bonus==\n'''[[Rosy Scent]]:'''\n"); ok {
		t.Error("a section with no level row should not resolve")
	}
}

func TestCompareSpiritBonuses(t *testing.T) {
	wiki := map[int]int{
		880156: 1200,
		880087: 800,
		880163: 800,
		880073: 200,
	}
	spirit := func(id, bonus int) Entry {
		e := Entry{Position: "spirit"}
		e.Item.ID, e.Item.Slot, e.Item.FlatBonus = id, scoring.Spirit, bonus
		return e
	}
	packed := []Entry{
		spirit(880156, 1200),
		spirit(880087, 800),
		spirit(880163, 1200),
		spirit(880109, 500),
		spirit(880999, 0),
		{Item: scoring.Item{ID: 10001, Slot: scoring.Hair}},
	}

	check := CompareSpiritBonuses(wiki, packed)
	if check.Agree != 2 {
		t.Errorf("Agree = %d, want 2", check.Agree)
	}
	if !slices.Equal(check.Disagree, []int{880163}) {
		t.Errorf("Disagree = %v, want [880163]", check.Disagree)
	}
	if !slices.Equal(check.WikiOnly, []int{880073}) {
		t.Errorf("WikiOnly = %v, want [880073]", check.WikiOnly)
	}
	if !slices.Equal(check.PackedOnly, []int{880109}) {
		t.Errorf("PackedOnly = %v, want [880109]", check.PackedOnly)
	}
}

func TestParseSpiritBonusesRejectsUnknownIDs(t *testing.T) {
	_, stats, err := ParseSpiritBonuses(strings.NewReader(spiritDump), map[int]bool{880156: true})
	if err != nil {
		t.Fatal(err)
	}
	if stats.Parsed != 1 {
		t.Errorf("Parsed = %d, want 1", stats.Parsed)
	}
	if stats.Rejected != 4 {
		t.Errorf("Rejected = %d, want 4", stats.Rejected)
	}
}
