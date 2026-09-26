package pipeline

import (
	"os"
	"strings"
	"testing"
)

const packedSourcesSrc = `
var code2suit = ['虎皮喵星人','仲夏星辰'];
var code2src = ['赠送','店·金币','抽·谜','抽·幻','设·图','活动·蔷薇光华','签到·2月','活动·未知'];
var codewardrobe = [
'甲|01|0|||0|1/2|0',
'乙|02|0|||0|@1|0',
'丙|03|0|||0|!2/5|0',
'丁|1J|0|||0|*1|0',
'戊|X1|0|||0|~绫罗1-1|0',
'己|04|0|||0|3-12少/II-1-支2公|0',
'庚|05|0|||0|6|0',
'辛|06|0|||0|故宫-1|0',
'壬|07|0|||0|7|0',
'癸|08|0|||0|~南希2-3|0',
];
`

const acquisitionMapSrc = `{
  "codes": {
    "赠送": {"k": "gift", "t": "Gift", "past": true},
    "店·金币": {"k": "store", "t": "Clothes Store (gold)"},
    "抽·谜": {"k": "pavilion", "t": "Pavilion of Mystery"},
    "活动·蔷薇光华": {"k": "event", "t": "Rose Glow event", "past": true, "basis": "named by the wiki on 46 of the 47 items both sources cover"}
  },
  "signIn": {"k": "signin", "t": "Monthly Sign-In", "past": true},
  "texts": {"故宫-1": {"k": "event", "t": "Limited event", "past": true}},
  "dreamWeavers": {"绫罗": "Lunar", "南希": ""}
}`

func packedCatalogue() AcquisitionCatalogue {
	return AcquisitionCatalogue{
		Names: map[int]string{
			10001: "Test A", 10002: "Test B", 10003: "Test C", 20019: "Test D",
			880001: "Test E", 10004: "Test F", 10005: "Test G", 10006: "Test H", 10008: "Test J",
		},
		Suits:  map[int]string{20019: "Test Suit"},
		Stages: map[string]bool{"Story/3-12": true},
	}
}

func TestParsePackedSources(t *testing.T) {
	got, err := ParsePackedSources([]byte(packedSourcesSrc), map[int]bool{10001: true, 10002: true, 10003: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d items, want the 3 known ones", len(got))
	}
	want := []PackedSource{{Kind: "evolve", ID: 10002}, {Kind: "code", Value: "活动·蔷薇光华"}}
	if s := got[10003]; len(s) != 2 || s[0] != want[0] || s[1] != want[1] {
		t.Errorf("10003 = %+v, want %+v", s, want)
	}
	all, err := ParsePackedSources([]byte(packedSourcesSrc), nil)
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[int]PackedSource{
		10002:  {Kind: "customize", ID: 10001},
		20019:  {Kind: "suit", Value: "仲夏星辰"},
		880001: {Kind: "dream", Value: "绫罗1-1"},
		10004:  {Kind: "text", Value: "3-12少"},
		10006:  {Kind: "text", Value: "故宫-1"},
	} {
		if s := all[id]; len(s) == 0 || s[0] != want {
			t.Errorf("%d = %+v, want %+v first", id, s, want)
		}
	}
}

const crossWiredSrc = `
var code2suit = [];
var code2src = [];
var codewardrobe = [
'莹白花冠|81|0|||0||0',
'云舞华章|E2|0|||0||0',
'莹白花冠·珍稀|83|0|||0|!1|0',
'云舞华章·珍稀|D4|0|||0|!2|0',
'云舞华章·暮|D5|0|||0|@3|0',
'莹白花冠·暮|86|0|||0|@4|0',
'束缚绷带|E7|0|||0||0',
'魅惑颈圈|D8|0|||0|!7|0',
'糖果手饰·华丽|F9|0|||0||0',
'糖果手饰·珍稀|HA|0|||0|!9|0',
];
`

func TestParsePackedSourcesDropsCrossWiredBases(t *testing.T) {
	got, err := ParsePackedSources([]byte(crossWiredSrc), nil)
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[int]PackedSource{
		80003: {Kind: "evolve", ID: 80001},
		80004: {Kind: "evolve", ID: 80002},
		80005: {Kind: "customize"},
		80006: {Kind: "customize"},
		80008: {Kind: "evolve", ID: 80007},
		80010: {Kind: "evolve", ID: 80009},
	} {
		if s := got[id]; len(s) != 1 || s[0] != want {
			t.Errorf("%d = %+v, want %+v", id, s, want)
		}
	}
	m, err := ReadAcquisitionMap([]byte(acquisitionMapSrc))
	if err != nil {
		t.Fatal(err)
	}
	cat := AcquisitionCatalogue{Names: map[int]string{80003: "Crown - Rare", 80004: "Cloud - Rare", 80005: "Cloud - Dusk"}}
	lines, err := m.Translate(map[int][]PackedSource{80005: got[80005]}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if out := string(WriteAcquisition("t", lines)); out != `{"version":"t","items":{"80005":[{"k":"customize","t":"Customization","cn":1}]}}` {
		t.Errorf("a cross-wired base still names an ingredient: %s", out)
	}
}

func TestCheckBasisRecountsTheSources(t *testing.T) {
	m := &AcquisitionMap{Codes: map[string]MappedSource{
		"活动·甲": {Kind: "event", Text: "Rose Glow event", Past: true, Basis: "named by the wiki on 3 of the 4 items both sources cover"},
		"赠送·乙": {Kind: "signin", Text: "Log-in Event", Past: true, Basis: "named by the wiki on 2 of the 3 items both sources cover"},
	}}
	src := map[int][]PackedSource{
		1: {{Kind: "code", Value: "活动·甲"}}, 2: {{Kind: "code", Value: "活动·甲"}}, 3: {{Kind: "code", Value: "活动·甲"}},
		4: {{Kind: "code", Value: "活动·甲"}}, 5: {{Kind: "code", Value: "活动·甲"}},
		6: {{Kind: "code", Value: "赠送·乙"}}, 7: {{Kind: "code", Value: "赠送·乙"}}, 8: {{Kind: "code", Value: "赠送·乙"}},
	}
	rose := []Acquisition{{Kind: "event", Text: "Rose Glow event", Past: true}}
	wiki := map[int][]Acquisition{
		1: rose, 2: rose, 3: rose, 4: {{Kind: "event", Text: "Rose Gold event", Past: true}},
		6: {{Kind: "signin", Text: "Log-in Event", Past: true}}, 7: {{Kind: "signin", Text: "Log-in Event", Past: true}},
		8: {{Kind: "gift", Text: "Mailbox Gift", Past: true}},
	}
	if err := m.CheckBasis(src, wiki); err != nil {
		t.Fatalf("true counts were refused: %v", err)
	}
	wiki[3] = []Acquisition{{Kind: "event", Text: "Dreamy Sea Voyage event", Past: true}}
	err := m.CheckBasis(src, wiki)
	if err == nil || !strings.Contains(err.Error(), `named by the wiki on 2 of the 4 items both sources cover`) {
		t.Errorf("a false count passed: %v", err)
	}
	m.Codes["活动·甲"] = MappedSource{Kind: "event", Text: "Rose Glow event", Past: true, Basis: "named by the wiki on 2 of the 4 items both sources cover"}
	if err := m.CheckBasis(src, wiki); err == nil || !strings.Contains(err.Error(), "only 2 of 4") {
		t.Errorf("an event named on half the items passed: %v", err)
	}
	m.Codes["活动·甲"] = MappedSource{Kind: "event", Text: "Rose Glow event", Past: true, Basis: "named by the wiki on most items"}
	if err := m.CheckBasis(src, wiki); err == nil || !strings.Contains(err.Error(), "not a count") {
		t.Errorf("a basis without a count passed: %v", err)
	}
}

func TestShippedAcquisitionMapBasisIsACount(t *testing.T) {
	raw, err := os.ReadFile("../data/acquisition-cn.json")
	if err != nil {
		t.Fatal(err)
	}
	m, err := ReadAcquisitionMap(raw)
	if err != nil {
		t.Fatal(err)
	}
	for code, s := range m.Codes {
		if s.Basis != "" && !basisCount.MatchString(s.Basis) {
			t.Errorf("%s: basis %q is not a count the build can check", code, s.Basis)
		}
	}
	for _, code := range []string{"抽·沉落之海", "抽·云空之境"} {
		if s := m.Codes[code]; s.Text != limitedEvent || s.Basis != "" {
			t.Errorf("%s names %q, and the wiki splits its items between two events", code, s.Text)
		}
	}
}

func TestTranslatePackedSources(t *testing.T) {
	m, err := ReadAcquisitionMap([]byte(acquisitionMapSrc))
	if err != nil {
		t.Fatal(err)
	}
	src, err := ParsePackedSources([]byte(packedSourcesSrc), nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := m.Translate(src, packedCatalogue())
	if err != nil {
		t.Fatal(err)
	}
	want := `{"version":"t","items":{` +
		`"10001":[{"k":"store","t":"Clothes Store (gold)","cn":1},{"k":"pavilion","t":"Pavilion of Mystery","cn":1}],` +
		`"10002":[{"k":"customize","t":"Customize: Test A","from":[[10001,1]],"cn":1}],` +
		`"10003":[{"k":"evolve","t":"Evolve: Test B","from":[[10002,0]],"cn":1},{"k":"event","t":"Rose Glow event","past":1,"cn":1}],` +
		`"10004":[{"k":"stage","t":"Story 3-12 (Maiden)","stage":"Story/3-12","level":"Maiden","cn":1},{"k":"stage","t":"Story II-1-Side 2 (Princess)","level":"Princess","cn":1}],` +
		`"10005":[{"k":"signin","t":"Monthly Sign-In","past":1,"cn":1}],` +
		`"10006":[{"k":"event","t":"Limited event","past":1,"cn":1}],` +
		`"10008":[{"k":"dream","t":"Dream Weaver","cn":1}],` +
		`"20019":[{"k":"suit","t":"Styling Gift Box for completing Test Suit","cn":1}],` +
		`"880001":[{"k":"dream","t":"Dream Weaver: Lunar","cn":1}]}}`
	if out := string(WriteAcquisition("t", got)); out != want {
		t.Errorf("\n got %s\nwant %s", out, want)
	}
}

func TestTranslatePrintsShownNames(t *testing.T) {
	m, err := ReadAcquisitionMap([]byte(acquisitionMapSrc))
	if err != nil {
		t.Fatal(err)
	}
	src, err := ParsePackedSources([]byte(packedSourcesSrc), nil)
	if err != nil {
		t.Fatal(err)
	}
	cat := packedCatalogue()
	cat.Shown = map[int]string{10001: "Test A (Hair)", 10002: "Test Bee"}
	got, err := m.Translate(src, cat)
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[int]Acquisition{
		10002: {Kind: "customize", Text: "Customize: Test A (Hair)", From: []Ingredient{{10001, 1}}, CN: true},
		10003: {Kind: "evolve", Text: "Evolve: Test Bee", From: []Ingredient{{10002, 0}}, CN: true},
	} {
		if len(got[id]) == 0 || !sameAcquisition(got[id][0], want) {
			t.Errorf("%d: %+v, want %+v", id, got[id], want)
		}
	}
}

func TestTranslateRejectsUnknownSources(t *testing.T) {
	m, err := ReadAcquisitionMap([]byte(acquisitionMapSrc))
	if err != nil {
		t.Fatal(err)
	}
	src, err := ParsePackedSources([]byte(packedSourcesSrc), nil)
	if err != nil {
		t.Fatal(err)
	}
	cat := packedCatalogue()
	cat.Names[10007] = "Test I"
	src[10009] = []PackedSource{{Kind: "text", Value: "剧情II-9-9"}, {Kind: "dream", Value: "某人1-1"}}
	cat.Names[10009] = "Test K"
	_, err = m.Translate(src, cat)
	if err == nil {
		t.Fatal("an unmapped code was translated")
	}
	for _, want := range []string{"3 sources", "code 活动·未知", "text 剧情II-9-9", "dream 某人1-1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestReadAcquisitionMapRefuses(t *testing.T) {
	for name, src := range map[string]string{
		"unknown kind":      `{"codes": {"x": {"k": "shop", "t": "Shop"}}, "signIn": {"k": "signin", "t": "Monthly Sign-In"}}`,
		"Chinese line":      `{"codes": {"x": {"k": "store", "t": "服装店"}}, "signIn": {"k": "signin", "t": "Monthly Sign-In"}}`,
		"unreviewed event":  `{"codes": {"x": {"k": "event", "t": "Rose Glow event"}}, "signIn": {"k": "signin", "t": "Monthly Sign-In"}}`,
		"no sign-in line":   `{"codes": {}}`,
		"Chinese character": `{"codes": {}, "signIn": {"k": "signin", "t": "Monthly Sign-In"}, "dreamWeavers": {"绫罗": "绫罗"}}`,
	} {
		if _, err := ReadAcquisitionMap([]byte(src)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestShippedAcquisitionMapReads(t *testing.T) {
	raw, err := os.ReadFile("../data/acquisition-cn.json")
	if err != nil {
		t.Fatal(err)
	}
	m, err := ReadAcquisitionMap(raw)
	if err != nil {
		t.Fatal(err)
	}
	for code, s := range m.Codes {
		if s.Kind == "event" && s.Text != limitedEvent && !strings.HasSuffix(s.Text, " event") {
			t.Errorf("%s: event line %q does not end in \" event\"", code, s.Text)
		}
	}
	if m.Codes["联盟·协力"].Text != "Shadow Workshop" || m.DreamWeavers["绫罗"] != "Lunar" {
		t.Error("the shipped map lost a reviewed line")
	}
}
