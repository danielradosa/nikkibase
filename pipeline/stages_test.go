package pipeline

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
)

func TestStyleFromStageTag(t *testing.T) {
	for _, c := range []struct {
		in, want string
		ok       bool
	}{
		{"Lolita(洛丽塔)", "Lolita", true},
		{"洛丽塔", "Lolita", true},
		{"Sun Care(Sun Care(防晒))", "Sun Care", true},
		{"防晒", "Sun Care", true},
		{"OL", "Chic", true},
		{"Lolita", "Lolita", true},
		{"Chinese Classical(中式古典)", "Chinese Classical", true},
		{"中式现代", "Modern China", true},
		{"不存在的标签", "", false},
	} {
		got, ok := StyleFromStageTag(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("StyleFromStageTag(%q) = %q,%v; want %q,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestRenamedStylesResolveToTheirCurrentName(t *testing.T) {
	for _, c := range []struct {
		current string
		stage   []string
		code    string
	}{
		{"Animal", []string{"Pet", "Pet(小动物)", "小动物", "动物系"}, "Pet"},
		{"Musician", []string{"Rock", "Rock(摇滚风)", "摇滚风", "乐队风"}, "Roc"},
		{"Chic", []string{"Office", "OL", "轻熟风"}, "Off"},
		{"Street", []string{"Harajuku", "Harajuku(原宿系)", "原宿系", "潮酷风"}, "Har"},
		{"Sailor", []string{"Navy", "Navy(海军风)", "海军风", "航海风"}, "Nav"},
		{"Multicultural", []string{"Hindu", "Hindu(印度服饰)", "印度服饰", "异域风"}, "Hin"},
		{"Workwear", []string{"Denim", "Denim(牛仔布)", "牛仔布", "工装风"}, "Den"},
	} {
		for _, in := range c.stage {
			if got, ok := StyleFromStageTag(in); !ok || got != c.current {
				t.Errorf("StyleFromStageTag(%q) = %q,%v; want %q", in, got, ok, c.current)
			}
		}
		if got, ok := StyleFromCode(c.code); !ok || got != c.current {
			t.Errorf("StyleFromCode(%q) = %q,%v; want %q", c.code, got, ok, c.current)
		}
	}
}

func TestItemAndStageTagsAgree(t *testing.T) {
	fromItem, ok := StyleFromCode("Lol")
	if !ok {
		t.Fatal("wiki shortcode Lol did not resolve")
	}
	fromStage, ok := StyleFromStageTag("洛丽塔")
	if !ok {
		t.Fatal("stage tag 洛丽塔 did not resolve")
	}
	if fromItem != fromStage {
		t.Fatalf("item says %q, stage says %q", fromItem, fromStage)
	}
	a, _ := TagID(fromItem)
	b, _ := TagID(fromStage)
	if a != b {
		t.Errorf("tag IDs differ: item %d, stage %d", a, b)
	}
}

const bonusSrc = `var levelsRaw = {
  '1-1': [1, 2, 3, 2, 1],
  '1-2': [1, 2, 3, 2, 1],
};
var levelBonus = {
  "1-1": [addBonusInfo('B', 0.25, "Lolita(洛丽塔)")],
  "1-2": [],
  "9-9": [addBonusInfo('A', 1, "睡衣")],
};`

func TestParseStagesBonus(t *testing.T) {
	stages, stats, err := ParseStages([]byte(bonusSrc))
	if err != nil {
		t.Fatal(err)
	}
	if len(stages) != 2 {
		t.Fatalf("got %d stages, want 2", len(stages))
	}
	if stages[0].Mode != "Story" {
		t.Errorf("mode = %q, want Story", stages[0].Mode)
	}
	id, _ := TagID("Lolita")
	if got := stages[0].Stage.Tags[id]; got != 2946 {
		t.Errorf("award = %d, want 2946", got)
	}
	if stages[1].Stage.Tags != nil {
		t.Errorf("stage with an empty bonus list got tags %v", stages[1].Stage.Tags)
	}
	if stats.OrphanBonus != 1 {
		t.Errorf("OrphanBonus = %d, want 1", stats.OrphanBonus)
	}
	if stats.UnknownTag != 0 || stats.UnknownGrade != 0 {
		t.Errorf("unexpected unknowns: %+v", stats)
	}
	if stats.BonusCalls != 2 || stats.Dropped != 0 || stats.Awards != 1 {
		t.Errorf("BonusCalls %d, Dropped %d, Awards %d; want 2, 0, 1", stats.BonusCalls, stats.Dropped, stats.Awards)
	}
}

const valuesBase = `var levelsRaw = {
  '1-1': [1, 2, 3, 2, 1],
  '1-2': [1, 2, 3, 2, 2],
  '1-3': [1, 2, 3, 2, 1],
};
var tasksRaw = {
'协战: 黑卡-海军': [1, 1, 1, 1, 2],
'联盟委托: 12-3': [-1, -0.73, -1.07, -0.67, -0.33],
};
var competitionsRaw = {
'海边派对的搭配': [2, 2, 2, 2, 2],
};
var levelBonus = {
  '1-1': [addBonusInfo('B', 0.25, "洛丽塔")],
  '1-2': [addBonusInfo('A', 1, "动物系")],
  '联盟委托: 12-3': [addBonusInfo('S', 0.5, "欧式古典")],
  '协战: 黑卡-海军': [addBonusInfo('S', 1, "航海风")],
};`

const valuesExact = `var competitionsRaw = {
'海边派对的搭配':[2,2,2,2,2.0667],
};
var tasksRaw = {
'联盟委托: 12-3': [-1, -0.7333, -1.0667, -0.6667, -0.3333],
'联盟委托: 21-1': [1, 1, 1, 1, 1],
};
var levelsRaw = {
'1-1':[1,2,3,2,1],
'1-2':[1,2,3,2,2],
'1-3':[1,-2,3,2,1],
};
var levelBonus = {
'1-1':[addBonusInfo('F',20,'洛丽塔')],
'联盟委托: 12-3':[addBonusInfo('F',72.8,'欧式古典'),addBonusInfo('F',33.2,'晚礼服')],
'海边派对的搭配':[addBonusInfo('F',30,'泳装')],
'联盟委托: 21-1':[addBonusInfo('F',10,'泳装')],
};`

func TestApplyStageValues(t *testing.T) {
	stages, stats, err := ParseStages([]byte(valuesBase))
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyStageValues(stages, []byte(valuesExact), nil, &stats); err != nil {
		t.Fatal(err)
	}
	by := map[string]scoring.Stage{}
	for _, s := range stages {
		by[s.Name] = s.Stage
	}
	id := func(name string) int {
		t.Helper()
		i, ok := TagID(name)
		if !ok {
			t.Fatalf("no tag %q", name)
		}
		return i
	}
	for _, c := range []struct {
		stage   string
		weights [5]float64
		tags    map[int]int
	}{
		{"1-1", [5]float64{15, 45, 30, 30, 15}, map[int]int{id("Lolita"): 2700}},
		{"1-2", [5]float64{15, 45, 30, 30, 30}, nil},
		{"1-3", [5]float64{15, 45, 30, 30, 15}, map[int]int{}},
		{"联盟委托: 12-3", [5]float64{15, 16, 11, 10, 5}, map[int]int{id("European"): 4150, id("Evening Gown"): 1892}},
		{"协战: 黑卡-海军", [5]float64{15, 15, 15, 15, 30}, map[int]int{id("Sailor"): 12537}},
		{"海边派对的搭配", [5]float64{30, 30, 30, 30, 31}, map[int]int{id("Swimsuit"): 4530}},
	} {
		got := by[c.stage]
		if got.Weights != c.weights {
			t.Errorf("%s weights = %v, want %v", c.stage, got.Weights, c.weights)
		}
		if len(got.Tags) != len(c.tags) || (len(c.tags) > 0 && !maps.Equal(got.Tags, c.tags)) {
			t.Errorf("%s tags = %v, want %v", c.stage, got.Tags, c.tags)
		}
	}
	if stats.Valued != 4 || stats.SideConflicts != 1 || stats.Beyond != 1 {
		t.Errorf("Valued %d, SideConflicts %d, Beyond %d; want 4, 1, 1", stats.Valued, stats.SideConflicts, stats.Beyond)
	}
	if stats.Awards != 5 || stats.WithBonus != 4 || stats.Dropped != 0 || stats.UnknownTag != 0 {
		t.Errorf("Awards %d, WithBonus %d, Dropped %d, UnknownTag %d; want 5, 4, 0, 0",
			stats.Awards, stats.WithBonus, stats.Dropped, stats.UnknownTag)
	}
}

func TestApplyStageValuesRefusesACorrectedStage(t *testing.T) {
	corrections := map[string]StageCorrection{
		"1-1": {Stage: "1-1", Weights: [5]float64{1, 2, 3, 2, 1}, Basis: "test", BonusDivisor: 1},
	}
	stages, stats, err := ParseStagesWith([]byte(valuesBase), corrections)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyStageValues(stages, []byte(valuesExact), corrections, &stats); err == nil || !strings.Contains(err.Error(), "1-1") {
		t.Errorf("err = %v, want a refusal naming 1-1", err)
	}
}

func TestWeightsAreWholeGameUnits(t *testing.T) {
	stages, _, err := ParseStages([]byte(`var levelsRaw = {
  '1-1': [0.07, 0.13, 2.93, -1.33, 0.2],
  '1-2': [1, 2, 2.9333, 2, -1.3333],
};`))
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range [][5]float64{{1, 44, 2, 20, 3}, {15, 44, 30, 30, 20}} {
		if got := stages[i].Stage.Weights; got != want {
			t.Errorf("%s weights = %v, want %v", stages[i].Name, got, want)
		}
	}
}

func TestFactorAwardsNeedNoGrade(t *testing.T) {
	stages, stats, err := ParseStages([]byte(`var tasksRaw = {
'联盟委托: 12-3': [-1, -0.7333, -1.0667, -0.6667, -0.3333],
};
var levelBonus = {
'联盟委托: 12-3':[addBonusInfo('F',72.8,'欧式古典'),addBonusInfo('F',33.2,'晚礼服')],
};`))
	if err != nil {
		t.Fatal(err)
	}
	european, _ := TagID("European")
	gown, _ := TagID("Evening Gown")
	want := map[int]int{european: 4150, gown: 1892}
	if got := stages[0].Stage.Tags; !maps.Equal(got, want) || stats.UnknownGrade != 0 {
		t.Errorf("tags = %v (unknown grades %d), want %v", got, stats.UnknownGrade, want)
	}
}

func TestTagAwardChangesTheChoice(t *testing.T) {
	lolita, _ := TagID("Lolita")
	plain := scoring.Item{ID: 1, Slot: scoring.Dress}
	plain.Attrs[0], plain.Stats[0] = scoring.Simple, 100
	tagged := scoring.Item{ID: 2, Slot: scoring.Dress, Tags: []int{lolita}}
	tagged.Attrs[0], tagged.Stats[0] = scoring.Simple, 90

	positions := []optimizer.Position{{Items: []scoring.Item{plain, tagged}}}
	st := scoring.Stage{}
	st.Attrs[0], st.Weights[0] = scoring.Simple, 1

	if got := optimizer.Best(positions, st, nil).Items[0].ID; got != plain.ID {
		t.Fatalf("without a tag award the higher stat should win, got item %d", got)
	}

	st.Tags = map[int]int{lolita: 196}
	if got := optimizer.Best(positions, st, nil).Items[0].ID; got != tagged.ID {
		t.Errorf("with a tag award the tagged item should win, got item %d", got)
	}
}

const rulesSrc = `var levelsRaw = {
  '1-1': [1, 2, 3, 2, 1],
  '1-2': [1, 2, 3, 2, 1],
  '1-3': [1, 2, 3, 2, 1],
  '1-4': [1, 2, 3, 2, 1],
};
var levelFilters = {
  '1-3': normalFilter("Chinese Classical(中式古典)/Modern China(中式现代)"),
  // '1-4': normalFilter("Unisex(中性风)"),
};
var addHintInfo = {
'1-1' :[[''],[''],['游园·紫']],
'1-2' :[[''],['浅空'],['']],
'1-3' :[['连衣裙要求中式风格的tag。'],[''],['']],
'9-9' :[['nothing here exists'],[''],['']],
};`

func TestRuleStylesUseCurrentNames(t *testing.T) {
	src := `var levelsRaw = {
  '3-2': [1, 2, 3, 2, 1],
};
var levelFilters = {
  '3-2': normalFilter("Rock(摇滚风)/Swordsman(侠客联盟)/Mystery(神秘)"),
};`
	stages, _, err := ParseStages([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if r := stages[0].Rules; r == nil || !slices.Equal(r.Styles, []string{"Musician", "Mystery", "Swordsman"}) {
		t.Errorf("3-2 rules = %+v, want Musician, Mystery, Swordsman", r)
	}
}

func TestParseStagesRules(t *testing.T) {
	stages, stats, err := ParseStages([]byte(rulesSrc))
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]*StageRules{}
	for _, s := range stages {
		by[s.Name] = s.Rules
	}
	if by["1-1"] == nil {
		t.Error("1-1 lists an item that scores F, and is not flagged")
	}
	if by["1-2"] != nil {
		t.Error("1-2 only recommends items, and is flagged as having rules")
	}
	if r := by["1-3"]; r == nil || !slices.Equal(r.Styles, []string{"Chinese Classical", "Modern China"}) {
		t.Errorf("1-3 rules = %+v, want the two styles its filter names", r)
	}
	if by["1-4"] != nil {
		t.Error("1-4's filter is commented out, and it is flagged anyway")
	}
	if stats.Ruled != 2 || stats.RuleEntries != 3 {
		t.Errorf("Ruled = %d, RuleEntries = %d, want 2 and 3", stats.Ruled, stats.RuleEntries)
	}
}

func TestApplyStageNames(t *testing.T) {
	stages, _, err := ParseStages([]byte(`var levelsRaw = {
  '1-1': [1, 1, 1, 1, 1],
};
var tasksRaw = {
'协战: 黑卡-女仆': [1, 1, 1, 1, 1],
'协战: 洁洁云-办公室': [1, 1, 1, 1, 1],
'协战: 拂苏-简约清凉的云端': [1, 1, 1, 1, 1],
};
var competitionsRaw = {
'海边派对的搭配': [1, 1, 1, 1, 1],
'无名的竞技场': [1, 1, 1, 1, 1],
};`))
	if err != nil {
		t.Fatal(err)
	}
	missing, err := ApplyStageNames(stages, []byte(`var tasksRaw = {
  '协战:Neve-Maiden(黑卡-女仆)': [1, 1, 1, 1, 1],
  '协战:Yvette-Ofiice Lady(洁洁云-办公室)': [1, 1, 1, 1, 1],
  '协战:Fu Su-Simple&Cool Cloud(拂苏-简约清凉云端)': [1, 1, 1, 1, 1],
};
var competitionsRaw = {
  'Beach Party(海边派对的搭配)': [1, 1, 1, 1, 1],
};`))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, s := range stages {
		got = append(got, DisplayName(s))
	}
	want := []string{"1-1", "Neva - Maiden", "Yvette - Office Lady", "Fu Su - Simple & Cool Cloud", "Beach Party", "无名的竞技场"}
	if !slices.Equal(got, want) {
		t.Errorf("names %q, want %q", got, want)
	}
	if !slices.Equal(missing, []string{"无名的竞技场"}) {
		t.Errorf("missing = %q, want the one arena stage with no English name", missing)
	}
}

func TestRecountDescribesTheFinalStages(t *testing.T) {
	stages, stats, err := ParseStages([]byte(valuesBase))
	if err != nil {
		t.Fatal(err)
	}
	stats.Recount(stages[:2])
	if stats.Stages != 2 || stats.WithBonus != 2 || stats.Awards != 2 {
		t.Errorf("Stages %d, WithBonus %d, Awards %d; want 2, 2, 2", stats.Stages, stats.WithBonus, stats.Awards)
	}
}
