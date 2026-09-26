package pipeline

import (
	"bytes"
	"maps"
	"strconv"
	"strings"
	"testing"
)

const acquisitionDump = `<mediawiki>
<page>
  <title>Template:Currency</title>
  <ns>10</ns>
  <revision><text>{{#switch: {{{1|}}}
|D = [[File:Diamond.png|18px|link=Currency#Types of Currency|Diamonds]]
|G = [[File:Gold.png|18px|link=Currency#Types of Currency|Gold]]
|SC = [[File:Starlight Coin.png|18px|link=Currency#Types of Currency|Starlight Coin]]
|AC = [[File:Association Coin.png|23px|link=Currency#Types of Currency|Association Coin|alt=Association Coin]]
}}</text></revision>
</page>
<page>
  <title>Template:Items</title>
  <ns>10</ns>
  <revision><text>{{#switch: {{{1|}}}
|M = [[File:Material.png|22px|link=Customization|Material]]
|LY = [[File:Lemon Yellow.png|25px|link=Customization#Dyes|Lemon Yellow]]
|KS = [[File:Karma Sand.png|19px|link=Shadow Workshop|Karma Sand]]
|MeC = [[File:Memory Crystal.png|19px|link=Shadow Workshop|Memory Crystal]]
}}</text></revision>
</page>
<page>
  <title>Test Laziness</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|name =
|type = Hair
|wardrobe nr = 17
|how to obtain = [[Crafting]]&lt;br /&gt;[[Pavilion of Mystery]]
}}
'''Test Laziness''' is a hair item. It can be obtained through [[Crafting]].

== [[Customization]] ==
* {{IconItem|Test Glow}}: 1 {{Items|LY}}, 3 {{Items|M}}

== [[Crafting]] ==
=== Crafted from: ===
{{Recipe
|source = Store of Starlight
|price = 204
|item1 = Test Khaki
|item1_quantity = 4
|item2 = Test Old Name
|item2_quantity = 3
}}
{{Recipe
|source = Extra Stage Bonus
|stage = ii.1-S2
|stage_type = P
|item1 = Test Scientist
|item1_quantity = 2
}}</text></revision>
</page>
<page>
  <title>Test Glow</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Hair
|wardrobe nr = 18
|how to obtain = [[Customization]]
}}
'''Test Glow''' is a hair item. It can be obtained through the [[Customization]] of [[Test Laziness]].

== Customization ==
* {{IconItem|Test Laziness}}: 1 {{Items|LY}}, 3 {{Items|M}}</text></revision>
</page>
<page>
  <title>Test Khaki</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Hair
|wardrobe nr = 1
|how to obtain = {{Stages|V1: 3-12|2|stage=1}} (Maiden)&lt;br&gt;{{Stages|V1: 9-9|2|stage=1}} (Princess)&lt;br&gt;[[Clothes Store]]
}}
'''Test Khaki''' is a hair item. It can be bought in the [[Clothes Store]] for 18,314 {{Currency|G}}.</text></revision>
</page>
<page>
  <title>Test Old Name</title>
  <ns>0</ns>
  <redirect title="Test Scientist" />
  <revision><text>#REDIRECT [[Test Scientist]]</text></revision>
</page>
<page>
  <title>Test Scientist</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Hair
|wardrobe nr = 2
|how to obtain = Evolution
}}
'''Test Scientist''' is a hair item. It can be obtained through [[Evolution]].

== [[Evolution]] ==
=== Evolved from: ===
* {{IconItem|Test Khaki|quantity=10}}, 7,000 {{Currency|G}}
* {{IconItem|Test Glow}}</text></revision>
</page>
<page>
  <title>Test Dawnblade</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Hair
|wardrobe nr = 3
|how to obtain = {{Stages|15-Side Story 2|2|stage=1}}
}}</text></revision>
</page>
<page>
  <title>Test Feather</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Accessory, Foreground
|wardrobe nr = 1957
|how to obtain = [[Shadow Workshop]]
}}
'''Test Feather''' is a foreground accessory. It can be obtained in the [[Shadow Workshop]] for 1344 {{Items|KS}} and 1085 {{Items|MeC}}.</text></revision>
</page>
<page>
  <title>Test Clown</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Dress
|movable =
|how to obtain = [[Circus Night Event|Circus Night]] event&lt;br&gt;[[Clothes Store]], [[Recharge]]
|wardrobe nr = 51
}}
'''Test Clown''' is a dress item. It could be obtained from the [[Circus Night Event|Circus Night]] event and can now be bought in the [[Clothes Store]] for 236 {{Currency|D}}.</text></revision>
</page>
<page>
  <title>Test Box</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Top
|wardrobe nr = 5
|how to obtain = [[Styling Gift Box]]&lt;br&gt;[[Dreamland - Yvette/Time Magic|Yvette: The Change]]
}}
'''Test Box''' is a top item. It can be obtained from a [[Styling Gift Box]] after completing [[Test Suit]].</text></revision>
</page>
<page>
  <title>Test Misnumbered</title>
  <ns>0</ns>
  <revision><text>{{Clothing
|type = Top
|wardrobe nr = 5
|how to obtain = [[Mailbox Gift]]
}}</text></revision>
</page>
<page>
  <title>Test Suit</title>
  <ns>0</ns>
  <revision><text>{{Suit Infobox
|type = Collection Suit
|theme =
|how to obtain = [[Recharge]]&lt;br /&gt;[[Secret Shop]]
|reward = {{Gift Box|Test Reward;30 {{Currency|D}}}}
}}
==Wardrobe==
{{Suit Part|Test Part (Coat)|type=Coat}}
{{Suit Part|Test Reward|type=Dress}}
{{Suit Part|Test Box|type=Top}}</text></revision>
</page>
</mediawiki>`

func acquisitionCatalogue() AcquisitionCatalogue {
	return AcquisitionCatalogue{
		Names: map[int]string{
			10001: "Test Khaki", 10002: "Test Scientist", 10003: "Test Dawnblade", 10017: "Test Laziness", 10018: "Test Glow",
			20051: "Test Clown", 40005: "Test Box", 81957: "Test Feather",
			30007: "Test Part", 20009: "Test Reward",
		},
		Suits:  map[int]string{30007: "Test Suit", 20009: "Test Suit", 40005: "Test Suit"},
		Stages: map[string]bool{"Story/3-12": true, "Story/II-1-Side 2": true},
	}
}

func TestParseFandomAcquisition(t *testing.T) {
	corrections := &IDCorrections{Drop: map[int]string{40005: "Test Misnumbered"}, Owner: map[int]string{40005: "Test Box"}}
	got, stats, err := ParseFandomAcquisition(strings.NewReader(acquisitionDump), nil, corrections, acquisitionCatalogue())
	if err != nil {
		t.Fatal(err)
	}
	want := `{"version":"t","items":{` +
		`"10001":[{"k":"stage","t":"Story 3-12 (Maiden)","stage":"Story/3-12","level":"Maiden"},` +
		`{"k":"stage","t":"Story 9-9 (Princess)","level":"Princess"},` +
		`{"k":"store","t":"Clothes Store · 18,314 Gold","cost":[[18314,"Gold"]]}],` +
		`"10002":[{"k":"evolve","t":"Evolve: 10× Test Khaki + 7,000 Gold","from":[[10001,10]],"cost":[[7000,"Gold"]]}],` +
		`"10003":[{"k":"stage","t":"Story 15-Side Story 2"}],` +
		`"10017":[{"k":"craft","t":"Craft: 4× Test Khaki, 3× Test Scientist","from":[[10001,4],[10002,3]],"recipe":"Store of Starlight · 204 Starlight Coin"},` +
		`{"k":"craft","t":"Craft: 2× Test Scientist","from":[[10002,2]],"recipe":"Extra Stage Bonus on Story II-1-Side 2 (Princess)","stage":"Story/II-1-Side 2","level":"Princess"},` +
		`{"k":"pavilion","t":"Pavilion of Mystery"}],` +
		`"10018":[{"k":"customize","t":"Customize: Test Laziness + 1 Lemon Yellow + 3 Material","from":[[10017,1]],"cost":[[1,"Lemon Yellow"],[3,"Material"]]}],` +
		`"20051":[{"k":"event","t":"Circus Night event","past":1},{"k":"store","t":"Clothes Store · 236 Diamonds","cost":[[236,"Diamonds"]]},{"k":"recharge","t":"Recharge","past":1}],` +
		`"40005":[{"k":"suit","t":"Styling Gift Box for completing Test Suit"},{"k":"dream","t":"Dream Weaver: Yvette"}],` +
		`"81957":[{"k":"craft","t":"Shadow Workshop · 1,344 Karma Sand + 1,085 Memory Crystal","cost":[[1344,"Karma Sand"],[1085,"Memory Crystal"]]}]}}`
	if out := string(WriteAcquisition("t", got.Items)); out != want {
		t.Errorf("items:\n got %s\nwant %s", out, want)
	}
	wantSuits := `{"version":"t","items":{` +
		`"20009":[{"k":"suit","t":"Styling Gift Box for completing Test Suit"}],` +
		`"30007":[{"k":"recharge","t":"Recharge","past":1},{"k":"store","t":"Secret Shop"}],` +
		`"40005":[{"k":"recharge","t":"Recharge","past":1},{"k":"store","t":"Secret Shop"}]}}`
	if out := string(WriteAcquisition("t", got.Suits)); out != wantSuits {
		t.Errorf("suits:\n got %s\nwant %s", out, wantSuits)
	}
	if stats.Dropped != 1 || stats.Pages != 8 || stats.SuitPages != 1 || stats.Unresolved != 0 || stats.UnknownUnits != 0 {
		t.Errorf("stats = %+v", stats)
	}
}

func TestParseFandomAcquisitionIsDeterministic(t *testing.T) {
	var first []byte
	for range 5 {
		got, _, err := ParseFandomAcquisition(strings.NewReader(acquisitionDump), nil, nil, acquisitionCatalogue())
		if err != nil {
			t.Fatal(err)
		}
		out := WriteAcquisition("t", got.Items)
		out = append(out, WriteAcquisition("t", got.Suits)...)
		if first == nil {
			first = out
		} else if !bytes.Equal(out, first) {
			t.Fatalf("two runs over the same dump differ:\n%s\n%s", first, out)
		}
	}
}

func TestParseFandomAcquisitionSkipsUnknownItems(t *testing.T) {
	got, _, err := ParseFandomAcquisition(strings.NewReader(acquisitionDump), map[int]bool{10001: true}, nil, acquisitionCatalogue())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[10001] == nil {
		t.Errorf("items = %v, want only 10001", got.Items)
	}
}

func TestClassifySource(t *testing.T) {
	for _, c := range []struct {
		in         string
		kind, text string
	}{
		{"[[Clothes Store]]", "store", "Clothes Store"},
		{"Clothing Store", "store", "Clothes Store"},
		{"[[Time Palace|Old Album]] event", "event", "Old Album event"},
		{"Frozen Event", "event", "Frozen event"},
		{"[[Honeymoon Holyland]]", "event", "Honeymoon Holyland event"},
		{"[[Recharge|Zodiac Lucky Pack]]", "recharge", "Zodiac Lucky Pack"},
		{"[[Log-in Event]]", "signin", "Log-in Event"},
		{"[[Monthly Sign-In]]", "signin", "Monthly Sign-In"},
		{"[[Dreamland - Fu Su]]", "dream", "Dream Weaver: Fu Su"},
		{"Complete [[Hip-hop Queen]]", "suit", "Styling Gift Box for completing Hip-hop Queen"},
		{"[[Mailbox Gift]]", "gift", "Mailbox Gift"},
		{"Porch of Misty", "pavilion", "Porch of Misty"},
		{"[[Association Store]]", "association", "Association Store"},
		{"Unlocked at Start", "other", "Available from the start"},
		{"Sent to mailbox for binding [account] to Facebook", "gift", "Mailbox Gift"},
		{"Unknown", "", ""},
		{"Craftin", "craft", "Crafting"},
		{"Pavillion of Glaze", "pavilion", "Pavilion of Glaze"},
		{"Pavillon of Mystery", "pavilion", "Pavilion of Mystery"},
		{"[[Time Palace|Old Album]]event", "event", "Old Album event"},
		{"Extra Stage Bonus Event", "other", "Extra Stage Bonus"},
		{"Time Diary event", "other", "Time Diary"},
		{"[[Friends List#Friend Suit|Obtain 20 friends]]", "achievement", "Reach 20 friends"},
		{"Gain 60 friends", "achievement", "Reach 60 friends"},
		{"[[Friends List]]", "achievement", "Friend count reward"},
		{"Completing the suit [[Sweet Talk]]", "suit", "Styling Gift Box for completing Sweet Talk"},
		{"Complete [[The Gentle Madam]]", "suit", "Styling Gift Box for completing The Gentle Madam"},
	} {
		got := classifySource(c.in)
		if got.kind != c.kind || got.text != c.text {
			t.Errorf("classifySource(%q) = %s %q, want %s %q", c.in, got.kind, got.text, c.kind, c.text)
		}
	}
}

func TestStageNames(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"V1: 3-12", "3-12"}, {"V2: 5-S2", "II-5-Side 2"}, {"ii.2-2", "II-2-2"},
		{"9-S3", "9-Side 3"}, {"V3: 1-7", "III-1-7"},
	} {
		if got, ok := wikiStageName(c.in); !ok || got != c.want {
			t.Errorf("wikiStageName(%q) = %q, %v, want %q", c.in, got, ok, c.want)
		}
	}
	if _, ok := wikiStageName("15-Side Story 2"); ok {
		t.Error("a side story is not a story stage")
	}
	for _, c := range []struct{ in, stage, level string }{
		{"Stage 5-11 (Maiden)", "5-11", "Maiden"},
		{"Stage 3-S2 Princess", "3-Side 2", "Princess"},
		{"[[V2: 2-2 Poem and Rose|Volume 2, Stage 2-2]] (Maiden)", "II-2-2", "Maiden"},
	} {
		got := classifySource(c.in)
		if got.kind != "stage" || got.stage != c.stage || got.level != c.level {
			t.Errorf("classifySource(%q) = %+v, want stage %s %s", c.in, got, c.stage, c.level)
		}
	}
}

func TestWikiUnits(t *testing.T) {
	units := wikiUnits("|D = [[File:Diamond.png|18px|link=Currency|Diamonds]]\n|AC = [[File:AC.png|23px|link=Currency|Association Coin|alt=Association Coin]]\n|D. = {{#if:}}")
	if units["D"] != "Diamonds" || units["AC"] != "Association Coin" || len(units) != 2 {
		t.Errorf("units = %v", units)
	}
}

func dumpPage(title, box, body string) string {
	return "<page>\n  <title>" + title + "</title>\n  <ns>0</ns>\n  <revision><text>" + box + "\n" + body + "</text></revision>\n</page>\n"
}

func clothing(kind string, nr int, obtain string) string {
	return "{{Clothing\n|type = " + kind + "\n|wardrobe nr = " + strconv.Itoa(nr) + "\n|how to obtain = " + obtain + "\n}}"
}

var reviewedDump = `<mediawiki>
<page>
  <title>Template:Currency</title>
  <ns>10</ns>
  <revision><text>{{#switch: {{{1|}}}
|G = [[File:Gold.png|18px|link=Currency#Types of Currency|Gold]]
|DH = [[File:Destiny Hourglass.png|18px|link=Currency#Types of Currency|Destiny Hourglass]]
}}</text></revision>
</page>
<page>
  <title>Template:Items</title>
  <ns>10</ns>
  <revision><text>{{#switch: {{{1|}}}
|M = [[File:Material.png|22px|link=Customization|Material]]
|IB = [[File:Innocent Blue.png|25px|link=Customization#Dyes|Innocent Blue]]
}}</text></revision>
</page>
` +
	dumpPage("Test Fairy Cat", clothing("Accessory, Hairpin", 2077, "[[Recharge]]"),
		"'''Test Fairy Cat''' is a hairpin. It could be obtained from a [[Cumulative Recharge]] and can now be bought in the [[Secret Shop]].") +
	dumpPage("Test Old Stock", clothing("Top", 41, "[[Recharge]]"),
		"'''Test Old Stock''' is a top. It could be obtained from the [[Secret Shop]].") +
	dumpPage("Test Corridor", clothing("Accessory, Background", 673, "Corridor of Clock&lt;br&gt;Pavilion of Time"),
		"'''Test Corridor''' is a background. It can be obtained from a draw in the [[Corridor of Clock]], or can be exchanged for 42 {{Currency|DH}} in the [[Pavilion of Time]].") +
	dumpPage("Test Dream", clothing("Hair", 1010, "[[Clothes Store]]"), "") +
	dumpPage("Test Dream - Rare", clothing("Hair", 1011, "[[Evolution]]"),
		"== Evolution ==\n=== Evolved from: ===\n* {{IconItem|Test Dream|quantity=6}}, 4000 {{Currency|G}}") +
	dumpPage("Test Dream - Epic", clothing("Hair", 1012, "[[Evolution]]"),
		"== Evolution ==\n=== Evolved from: ===\n* {{IconItem|Test Dream - Rare|quantity=4}}, 7000 {{Currency|G}}\n* {{IconItem|Test Dream|quantity=6}}, 4000 {{Currency|G}}") +
	dumpPage("Test Dream - Legend", clothing("Hair", 1013, "[[Evolution]]"),
		"== Evolution ==\n=== Evolved from: ===\n* {{IconItem|Test Dream|quantity=6}}, 4000 {{Currency|G}}\n* {{IconItem|Test Dream - Rare|quantity=4}}, 7000 {{Currency|G}}") +
	dumpPage("Test Clover - Pink", clothing("Dress", 54, "[[Clothes Store]]"),
		"== [[Customization]] ==\n* {{IconItem|Test Clover-Blue}}: 2 {{Items|IB}}, 10 {{Items|M}}") +
	dumpPage("Test Clover-Blue", clothing("Dress", 55, "[[Customization]]"),
		"'''Test Clover-Blue''' is a dress. It can be obtained through the [[Customization]] of [[Test Clover-Blue]].") +
	dumpPage("Test Night", clothing("Coat", 769, "[[Clothes Store]]"), "") +
	dumpPage("Test Gallus", clothing("Top", 713, "[[Customization]]"),
		"'''Test Gallus''' is a top. It can be obtained through the [[Customization]] of [[Test Night]].") +
	dumpPage("Test Crescent Dress", clothing("Dress", 970, "[[Styling Gift Box]]"),
		"'''Test Crescent Dress''' is a dress. It can be obtained from a [[Styling Gift Box]] after completing the [[Pigeon]] suit [[Test Crescent]].") +
	dumpPage("Test Privilege", clothing("Hair", 63, "[[Recharge|V8 Privilege Giftpack]]&lt;br&gt;[[Recharge|Zodiac Lucky Pack]]"), "") +
	dumpPage("Test Glaze", clothing("Dress", 732, "Pavillion of Glaze&lt;br/&gt;Craftin"),
		"'''Test Glaze''' is a dress. It can be exchanged in the [[Pavilion of Glaze]] for 50 {{Currency|G}}.") +
	dumpPage("Test Hunting", clothing("Bottom", 539, "[[Clothes Store]]"), "") +
	dumpPage("Test Hunting - Rare", clothing("Bottom", 1204, "[[Evolution]]"),
		"== Evolution ==\n=== Evolved from: ===\n* {{IconItem|Test Hunting|quantity=6}}, 4000 {{Currency|G}}") +
	dumpPage("Test Hunting - Epic", clothing("Bottom", 541, "[[Evolution]]"),
		"== Evolution ==\n=== Evolved from: ===\n* {{IconItem|Test Hunting - Rare|quantity=4}}, 7000 {{Currency|G}}") +
	`<page>
  <title>Test Taste</title>
  <ns>0</ns>
  <revision><text>{{Suit Infobox
|type = Collection Suit
|how to obtain = [[Recharge]]
}}
==Wardrobe==
{{Suit Part|Test Sweet&amp;Smooth|type=Bracelet (Right)}}
{{Suit Part|Test Situation (Hosiery)|type=Hosiery}}</text></revision>
</page>
</mediawiki>`

func reviewedCatalogue() AcquisitionCatalogue {
	return AcquisitionCatalogue{
		Names: map[int]string{
			82077: "Test Fairy Cat", 40041: "Test Old Stock", 80673: "Test Corridor",
			11010: "Test Dream", 11011: "Test Dream - Rare", 11012: "Test Dream - Epic", 11013: "Test Dream - Legend",
			20054: "Test Clover - Pink", 20055: "Test Clover - Blue", 30769: "Test Night", 40713: "Test Gallus",
			20970: "Test Crescent Dress", 10063: "Test Privilege", 20732: "Test Glaze",
			50539: "Test Hunting", 50540: "Test Hunting · Rare", 50541: "Test Hunting - Epic", 51204: "Test Hunting - Rare",
			182019: "Test Sweet & Smooth", 61364: "Test Situation", 88135: "Test Situation (Earrings)", 90621: "Test Situation (Makeup)",
		},
		Suits: map[int]string{88135: "Test Aries", 90621: "Test Dream"},
	}
}

func TestParseFandomAcquisitionReadsWhatTheWikiMeans(t *testing.T) {
	corrections := &IDCorrections{
		Owner:  map[int]string{51204: "Test Guardian - Glow"},
		Drop:   map[int]string{51204: "Test Hunting - Rare"},
		RealID: map[int]int{51204: 50540},
	}
	got, stats, err := ParseFandomAcquisition(strings.NewReader(reviewedDump), nil, corrections, reviewedCatalogue())
	if err != nil {
		t.Fatal(err)
	}
	want := `{"version":"t","items":{` +
		`"10063":[{"k":"recharge","t":"V8 Privilege Giftpack"},{"k":"recharge","t":"Zodiac Lucky Pack","past":1}],` +
		`"11010":[{"k":"store","t":"Clothes Store"}],` +
		`"11011":[{"k":"evolve","t":"Evolve: 6× Test Dream + 4,000 Gold","from":[[11010,6]],"cost":[[4000,"Gold"]]}],` +
		`"11012":[{"k":"evolve","t":"Evolve: 4× Test Dream - Rare + 7,000 Gold","from":[[11011,4]],"cost":[[7000,"Gold"]]}],` +
		`"11013":[{"k":"evolve","t":"Evolve: 4× Test Dream - Rare + 7,000 Gold","from":[[11011,4]],"cost":[[7000,"Gold"]]}],` +
		`"20054":[{"k":"store","t":"Clothes Store"}],` +
		`"20055":[{"k":"customize","t":"Customize: Test Clover - Pink + 2 Innocent Blue + 10 Material","from":[[20054,1]],"cost":[[2,"Innocent Blue"],[10,"Material"]]}],` +
		`"20732":[{"k":"pavilion","t":"Pavilion of Glaze · 50 Gold","cost":[[50,"Gold"]]},{"k":"craft","t":"Crafting"}],` +
		`"20970":[{"k":"suit","t":"Styling Gift Box for completing Test Crescent"}],` +
		`"30769":[{"k":"store","t":"Clothes Store"}],` +
		`"40041":[{"k":"recharge","t":"Recharge","past":1}],` +
		`"40713":[{"k":"customize","t":"Customization"}],` +
		`"50539":[{"k":"store","t":"Clothes Store"}],` +
		`"50540":[{"k":"evolve","t":"Evolve: 6× Test Hunting + 4,000 Gold","from":[[50539,6]],"cost":[[4000,"Gold"]]}],` +
		`"50541":[{"k":"evolve","t":"Evolve: 4× Test Hunting · Rare + 7,000 Gold","from":[[50540,4]],"cost":[[7000,"Gold"]]}],` +
		`"80673":[{"k":"pavilion","t":"Corridor of Clock"},{"k":"pavilion","t":"Pavilion of Time · 42 Destiny Hourglass","cost":[[42,"Destiny Hourglass"]]}],` +
		`"82077":[{"k":"recharge","t":"Recharge","past":1},{"k":"store","t":"Secret Shop"}]}}`
	if out := string(WriteAcquisition("t", got.Items)); out != want {
		t.Errorf("items:\n got %s\nwant %s", out, want)
	}
	if _, ok := got.Items[51204]; ok {
		t.Error("the misnumbered page still gives its lines to the ID it lands on")
	}
	if stats.Moved != 1 || stats.Dropped != 0 {
		t.Errorf("%d pages read at their own ID and %d left out, want 1 and 0", stats.Moved, stats.Dropped)
	}
	wantSuits := `{"version":"t","items":{"61364":[{"k":"recharge","t":"Recharge","past":1}],"182019":[{"k":"recharge","t":"Recharge","past":1}]}}`
	if out := string(WriteAcquisition("t", got.Suits)); out != wantSuits {
		t.Errorf("suits:\n got %s\nwant %s", out, wantSuits)
	}
	if v := CheckAcquisition(got.Items, reviewedCatalogue(), Coverage{}); len(v) != 0 {
		t.Errorf("the lines fail the bundle check: %v", v)
	}

	delete(corrections.RealID, 51204)
	got, stats, err = ParseFandomAcquisition(strings.NewReader(reviewedDump), nil, corrections, reviewedCatalogue())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Items[51204]; ok || stats.Dropped != 1 || stats.Moved != 0 {
		t.Errorf("a misnumbered page with no ID of its own: lines at 51204 %v, %d left out, %d moved; want none, 1, 0", ok, stats.Dropped, stats.Moved)
	}
	if _, ok := got.Items[50540]; ok {
		t.Error("a misnumbered page with no ID of its own still gives its lines to 50540")
	}
}

func TestPageUnderAnotherGarmentsNameKeepsItsID(t *testing.T) {
	dump := "<mediawiki>\n<page>\n  <title>Template:Currency</title>\n  <ns>10</ns>\n  <revision><text>{{#switch: {{{1|}}}\n" +
		"|G = [[File:Gold.png|18px|link=Currency#Types of Currency|Gold]]\n}}</text></revision>\n</page>\n" +
		dumpPage("Test Memory", clothing("Accessory, Earrings", 9121, "[[Test Festival]] event"), "") +
		dumpPage("Test Depicting", clothing("Accessory, Earrings", 9126, "[[Styling Gift Box]]"), "") +
		dumpPage("Test Chime", clothing("Accessory, Earrings", 9130, "[[Evolution]]"),
			"== Evolution ==\n=== Evolved from: ===\n* {{IconItem|Test Memory|quantity=2}}, 1000 {{Currency|G}}") +
		"</mediawiki>"
	cat := AcquisitionCatalogue{Names: map[int]string{89121: "Test Depicting", 89126: "Test Memory", 89130: "Test Chime"}}
	corrections := &IDCorrections{Name: map[int]NameOverride{
		89121: {ID: 89121, Name: "Test Depicting", Was: "Test Memory"},
		89126: {ID: 89126, Name: "Test Memory", Was: "Test Depicting"},
	}}
	got, _, err := ParseFandomAcquisition(strings.NewReader(dump), nil, corrections, cat)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"version":"t","items":{` +
		`"89121":[{"k":"event","t":"Test Festival event","past":1}],` +
		`"89126":[{"k":"suit","t":"Styling Gift Box"}],` +
		`"89130":[{"k":"evolve","t":"Evolve: 2× Test Memory + 1,000 Gold","from":[[89126,2]],"cost":[[1000,"Gold"]]}]}}`
	if out := string(WriteAcquisition("t", got.Items)); out != want {
		t.Errorf("items:\n got %s\nwant %s", out, want)
	}
	if v := CheckAcquisition(got.Items, cat, Coverage{}); len(v) != 0 {
		t.Errorf("the lines fail the bundle check: %v", v)
	}
}

func TestShownNamesPrintWhileTheSourcesSpellingMatches(t *testing.T) {
	dump := "<mediawiki>\n<page>\n  <title>Template:Currency</title>\n  <ns>10</ns>\n  <revision><text>{{#switch: {{{1|}}}\n" +
		"|G = [[File:Gold.png|18px|link=Currency#Types of Currency|Gold]]\n}}</text></revision>\n</page>\n" +
		dumpPage("Test Chime", clothing("Accessory, Earrings", 9130, "[[Evolution]]"),
			"== Evolution ==\n=== Evolved from: ===\n* {{IconItem|Test Chord|quantity=2}}, 1000 {{Currency|G}}") +
		"</mediawiki>"
	cat := AcquisitionCatalogue{
		Names: map[int]string{89126: "Test Chord", 89130: "Test Chime"},
		Shown: map[int]string{89126: "Test Chord Shown", 89130: "Test Chime (Earrings)"},
	}
	got, stats, err := ParseFandomAcquisition(strings.NewReader(dump), nil, nil, cat)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"version":"t","items":{` +
		`"89130":[{"k":"evolve","t":"Evolve: 2× Test Chord Shown + 1,000 Gold","from":[[89126,2]],"cost":[[1000,"Gold"]]}]}}`
	if out := string(WriteAcquisition("t", got.Items)); out != want || stats.Unresolved != 0 {
		t.Errorf("items (%d unresolved):\n got %s\nwant %s", stats.Unresolved, out, want)
	}
}

func TestPastSource(t *testing.T) {
	for _, c := range []struct {
		kind, text string
		past       bool
	}{
		{"recharge", "Recharge", true},
		{"recharge", "Zodiac Lucky Pack", true},
		{"recharge", "V8 Privilege Giftpack", false},
		{"recharge", "VIP 11", false},
		{"recharge", "VIP Rank 7 Reward", false},
		{"recharge", "First Recharge Giftpack", false},
		{"recharge", "Monthly Card", false},
		{"event", "VIP event", true},
		{"store", "Secret Shop", false},
	} {
		if got := pastSource(c.kind, c.text); got != c.past {
			t.Errorf("pastSource(%s, %q) = %v, want %v", c.kind, c.text, got, c.past)
		}
	}
}

func suitPage(title, obtain string, parts ...string) string {
	body := "{{Suit Infobox\n|type = Collection Suit\n|how to obtain = " + obtain + "\n}}\n==Wardrobe=="
	for _, p := range parts {
		body += "\n{{Suit Part|" + p + "}}"
	}
	return "<page>\n  <title>" + title + "</title>\n  <ns>0</ns>\n  <revision><text>" + body + "</text></revision>\n</page>\n"
}

func TestSuitsFromTheWikiTellSameNamedItemsApart(t *testing.T) {
	dump := "<mediawiki>\n" +
		dumpPage("Twin Chime (Earrings)", "{{Clothing\n|type = Accessory, Earrings\n|wardrobe nr = 9001\n|how to obtain = [[Recharge]]\n|part of suit = [[Beta Suit]]\n}}", "") +
		dumpPage("Alpha Gown", "{{Clothing\n|type = Dress\n|wardrobe nr = 5\n|how to obtain = [[Crafting]]\n|part of suit = [[Alpha Suit]]\n}}",
			"== [[Crafting]] ==\n=== Crafted from: ===\n{{Recipe\n|item1 = Twin Chime\n|item1_quantity = 1\n|item2 = Sand Twin\n|item2_quantity = 1\n}}") +
		dumpPage("Sand Twin (Leglet)", "{{Clothing\n|type = Hosiery, Leglet\n|wardrobe nr = 1\n|how to obtain = [[Recharge]]\n|part of suit = [[Beta Suit]]\n}}", "") +
		suitPage("Alpha Suit", "[[Secret Shop]]", "Alpha Gown", "Twin Chime", "Lone Coat") +
		suitPage("Beta Suit", "[[Recharge]]", "Twin Chime") +
		"</mediawiki>"
	cat := AcquisitionCatalogue{Names: map[int]string{
		89001: "Twin Chime", 89002: "Twin Chime", 20005: "Alpha Gown", 30007: "Lone Coat",
		40001: "Sand Twin", 60001: "Sand Twin",
	}}
	got, stats, err := ParseFandomAcquisition(strings.NewReader(dump), nil, nil, cat)
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]string{89001: "Beta Suit", 89002: "Alpha Suit", 20005: "Alpha Suit", 30007: "Alpha Suit", 60001: "Beta Suit"}
	if !maps.Equal(got.SuitOf, want) {
		t.Errorf("suits %v, want %v", got.SuitOf, want)
	}
	if lines := got.Suits[89002]; len(lines) != 1 || lines[0].Text != "Secret Shop" {
		t.Errorf("89002 from its suit page: %+v, want Alpha Suit's Secret Shop line", lines)
	}
	if lines := got.Suits[89001]; len(lines) != 1 || lines[0].Text != "Recharge" {
		t.Errorf("89001 from its suit page: %+v, want Beta Suit's Recharge line", lines)
	}
	if from := got.Items[20005]; len(from) == 0 || len(from[0].From) != 1 || from[0].From[0].ID != 89002 {
		t.Errorf("Alpha Gown's recipe: %+v, want Twin Chime resolved to 89002 and Sand Twin left as the wiki writes it", from)
	}
	if stats.Unresolved != 1 || stats.UnknownParts != 0 {
		t.Errorf("%d names and %d suit parts unmatched, want only Sand Twin, which no suit places", stats.Unresolved, stats.UnknownParts)
	}
}

func TestUnmatchedNamesAreHeldToTheirCeiling(t *testing.T) {
	stats := WikiAcquisitionStats{Unresolved: 2, UnknownParts: 3}
	if v := CheckAcquisitionNames(stats, Coverage{MaxUnmatchedNames: 5}); len(v) != 0 {
		t.Errorf("at the ceiling: %v", v)
	}
	if v := CheckAcquisitionNames(stats, Coverage{MaxUnmatchedNames: 4}); len(v) != 1 || !strings.Contains(v[0], "5 item names") {
		t.Errorf("above the ceiling: %v", v)
	}
}

func TestTheWikisItemListsNameWhatNothingElseCan(t *testing.T) {
	list := "<page>\n  <title>Hairs/2001-4000</title>\n  <ns>0</ns>\n  <revision><text>" +
		"{{WIL|2913|Twin Crown (Alpha Suit)|Twin Crown}}\n{{WIL|2914|Twin Crown (Beta Suit)|Twin Crown}}\n" +
		"{{WIL|2915|Split Entry}}\n{{WIL|2916|Split Entry}}</text></revision>\n</page>\n"
	dump := "<mediawiki>\n" + list +
		suitPage("Alpha Suit", "[[Secret Shop]]", "Twin Crown (Alpha Suit)", "Split Entry") +
		"</mediawiki>"
	cat := AcquisitionCatalogue{Names: map[int]string{
		12913: "Twin Crown", 12914: "Twin Crown", 12915: "Split Entry", 12916: "Split Entry",
	}}
	got, stats, err := ParseFandomAcquisition(strings.NewReader(dump), nil, nil, cat)
	if err != nil {
		t.Fatal(err)
	}
	if lines := got.Suits[12913]; len(lines) != 1 || lines[0].Text != "Secret Shop" {
		t.Errorf("12913: %+v, want Alpha Suit's Secret Shop line", lines)
	}
	if len(got.Suits) != 1 || got.SuitOf[12913] != "Alpha Suit" {
		t.Errorf("suit lines %v and suits %v, want only 12913 in Alpha Suit", got.Suits, got.SuitOf)
	}
	if stats.UnknownParts != 1 {
		t.Errorf("%d suit parts unmatched, want Split Entry, which the list gives two numbers", stats.UnknownParts)
	}
}

func typedSuitPage(title, kind, chinese string, parts ...string) string {
	page := suitPage(title, "[[Recharge]]", parts...)
	return strings.Replace(page, "|type = Collection Suit\n", "|type = "+kind+"\n|cnwiki = "+chinese+"\n", 1)
}

func TestASuitPageOutranksAPackThatListsTheSameItem(t *testing.T) {
	dump := "<mediawiki>\n" +
		typedSuitPage("Aa Winter Pack", "Pack", "冬礼包", "Snow Hat", "Snow Bell") +
		typedSuitPage("Winter Wish", "Collection Suit", "冬愿", "Snow Hat", "Snow Coat") +
		typedSuitPage("Zz Winter Wish Pack", "Pack", "冬愿", "Snow Coat") +
		"</mediawiki>"
	cat := AcquisitionCatalogue{Names: map[int]string{10001: "Snow Hat", 80002: "Snow Bell", 30003: "Snow Coat"}}
	got, stats, err := ParseFandomAcquisition(strings.NewReader(dump), nil, nil, cat)
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]string{10001: "Winter Wish", 80002: "Aa Winter Pack", 30003: "Winter Wish"}
	if !maps.Equal(got.SuitOf, want) {
		t.Errorf("suits %v, want %v", got.SuitOf, want)
	}
	if stats.SharedParts != 0 {
		t.Errorf("%d items on two or more suit pages, want none: the others that list Snow Hat and Snow Coat are packs", stats.SharedParts)
	}
	if want := map[string]string{"冬礼包": "Aa Winter Pack"}; !maps.Equal(got.ChineseSuits, want) {
		t.Errorf("Chinese names %v, want %v: a name two pages claim names neither", got.ChineseSuits, want)
	}
}

func TestSuitPagesGiveTheirChineseNamesTrimmed(t *testing.T) {
	dump := "<mediawiki>\n" +
		typedSuitPage("Alpha Suit", "Collection Suit", " 甲套 ", "Alpha Gown") +
		suitPage("Beta Suit", "[[Recharge]]", "Beta Gown") +
		"</mediawiki>"
	cat := AcquisitionCatalogue{Names: map[int]string{20001: "Alpha Gown", 20002: "Beta Gown"}}
	got, _, err := ParseFandomAcquisition(strings.NewReader(dump), nil, nil, cat)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"甲套": "Alpha Suit"}; !maps.Equal(got.ChineseSuits, want) {
		t.Errorf("Chinese names %v, want %v", got.ChineseSuits, want)
	}
}

func TestDayAndNightFormsBothJoinTheirSuit(t *testing.T) {
	dump := "<mediawiki>\n" +
		suitPage("Dawn Suit", "[[Recharge]]",
			"Dawn Veil (Day)|icon2=Dawn Veil (Night)|type=Hair", "Dawn Gown (Night)|type=Dress|v=2", "Dusk Pin (Day)") +
		"</mediawiki>"
	cat := AcquisitionCatalogue{Names: map[int]string{
		10001: "Dawn Veil", 10002: "Dawn Veil", 80003: "Dawn Veil",
		20004: "Dawn Gown", 20005: "Dawn Gown", 80006: "Dusk Pin", 80007: "Dusk Pin",
	}}
	got, _, err := ParseFandomAcquisition(strings.NewReader(dump), nil, nil, cat)
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]string{10001: "Dawn Suit", 10002: "Dawn Suit", 20004: "Dawn Suit", 20005: "Dawn Suit"}
	if !maps.Equal(got.SuitOf, want) {
		t.Errorf("suits %v, want %v: both forms in the part's slot, nothing for a part with no type", got.SuitOf, want)
	}
}

func TestASlotInAPartNameKeepsItOffAnotherSlotsItem(t *testing.T) {
	list := "<page>\n  <title>Shoes/2001-4000</title>\n  <ns>0</ns>\n  <revision><text>" +
		"{{WIL|2014|Prism Step (Shoes)|Prism Steps}}</text></revision>\n</page>\n"
	dump := "<mediawiki>\n" + list +
		suitPage("Duel Suit", "[[Recharge]]", "Prism Step (Shoes)|type=Shoes") +
		"</mediawiki>"
	cat := AcquisitionCatalogue{Names: map[int]string{12106: "Prism Step (Hair)", 72014: "Prism Steps"}}
	got, _, err := ParseFandomAcquisition(strings.NewReader(dump), nil, nil, cat)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[int]string{72014: "Duel Suit"}; !maps.Equal(got.SuitOf, want) {
		t.Errorf("suits %v, want %v: a part named for shoes is not the hair", got.SuitOf, want)
	}
}

func TestAnItemPageYieldsToTheSuitPageThatListsIt(t *testing.T) {
	dump := "<mediawiki>\n" +
		dumpPage("March Plume-Memory", "{{Clothing\n|type = Hair\n|wardrobe nr = 1965\n|how to obtain = [[Recharge]]\n|part of suit = [[Revival Suit]]\n}}", "") +
		dumpPage("Lone Heel", "{{Clothing\n|type = Shoes\n|wardrobe nr = 5\n|how to obtain = [[Recharge]]\n|part of suit = [[Revival Suit]]\n}}", "") +
		suitPage("March Suit", "[[Recharge]]", "March Plume-Memory") +
		suitPage("Revival Suit", "[[Recharge]]", "Old Cape") +
		"</mediawiki>"
	cat := AcquisitionCatalogue{Names: map[int]string{11965: "March Plume - Memory", 70005: "Lone Heel", 30001: "Old Cape"}}
	got, _, err := ParseFandomAcquisition(strings.NewReader(dump), nil, nil, cat)
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]string{11965: "March Suit", 70005: "Revival Suit", 30001: "Revival Suit"}
	if !maps.Equal(got.SuitOf, want) {
		t.Errorf("suits %v, want %v", got.SuitOf, want)
	}
}

func TestPacksAreNoSuits(t *testing.T) {
	dump := "<mediawiki>\n" +
		typedSuitPage("Aa Winter Pack", "Pack", "冬礼包", "Snow Hat", "Snow Bell") +
		typedSuitPage("Winter Wish", "Collection Suit", "冬愿", "Snow Hat") +
		"</mediawiki>"
	cat := AcquisitionCatalogue{Names: map[int]string{10001: "Snow Hat", 80002: "Snow Bell", 80003: "Snow Bow"}}
	wiki, _, err := ParseFandomAcquisition(strings.NewReader(dump), nil, nil, cat)
	if err != nil {
		t.Fatal(err)
	}
	packed := map[int]PackedSuit{80003: {Name: "冬礼包"}}
	got, stats := LayerSuits([]int{10001, 80002, 80003}, wiki, packed)
	if want := map[int]string{10001: "Winter Wish"}; !maps.Equal(got, want) {
		t.Errorf("suits %v, want %v", got, want)
	}
	if stats.Packs != 1 || stats.Unnamed != 1 || stats.None != 1 {
		t.Errorf("stats %+v, want Snow Bell counted as only on a pack and Snow Bow's pack name unnamed", stats)
	}
}

func TestAnAmbiguousPartTakesTheItemThePackedTableLeavesFree(t *testing.T) {
	dump := "<mediawiki>\n" +
		typedSuitPage("Lone Suit", "Collection Suit", "孤月", "Glow Veil (Lone Suit)") +
		typedSuitPage("Star Suit", "Collection Suit", "星语", "Glow Veil (Star Suit)") +
		"</mediawiki>"
	cat := AcquisitionCatalogue{Names: map[int]string{10001: "Glow Veil", 10002: "Glow Veil"}}
	wiki, _, err := ParseFandomAcquisition(strings.NewReader(dump), nil, nil, cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(wiki.SuitOf) != 0 || len(wiki.Unplaced) != 2 {
		t.Fatalf("suits %v and unplaced parts %v, want none placed and both parts waiting", wiki.SuitOf, wiki.Unplaced)
	}
	got, stats := LayerSuits([]int{10001, 10002}, wiki, map[int]PackedSuit{10001: {Name: "孤月"}})
	if want := map[int]string{10001: "Lone Suit", 10002: "Star Suit"}; !maps.Equal(got, want) {
		t.Errorf("suits %v, want %v", got, want)
	}
	if stats.Packed != 1 || stats.Late != 1 {
		t.Errorf("stats %+v, want one from the packed table and one placed after it", stats)
	}
	if got, _ := LayerSuits([]int{10001, 10002}, wiki, nil); len(got) != 0 {
		t.Errorf("suits %v, want none while both items are free", got)
	}
}
