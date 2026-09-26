package pipeline

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

const ruleSrc = `var levelsRaw = {
'1-1': [1, 1, 1, 1, 1],
'1-2': [1, 1, 1, 1, 1],
'1-3': [1, 1, 1, 1, 1],
};
var addHintInfo = {
'1-1' :[['必做测试发型+测试长裙+测试鞋'],[''],['']],
'1-3' :[['少女级必做测试鞋'],[''],['']],
};`

const ruleJSON = `{
  "note": "test",
  "rules": [
    {"stage": "1-1", "mode": "Story", "require": [[10001], [20001], [70001]],
     "source": "stage source note", "quote": "必做测试发型+测试长裙+测试鞋"},
    {"stage": "1-2", "mode": "Story", "require": [[40001, 20001], [50001, 20001]],
     "source": "wiki quest field", "quote": "[[Test Top]] and [[Test Bottom]], or [[Test Dress]]"},
    {"stage": "1-3", "mode": "Story", "require": [[83001, 83002]],
     "source": "wiki quest field", "quote": "[[Test Hat]] OR [[Test Staff]]"},
    {"stage": "1-3", "mode": "Story", "difficulty": "maiden", "require": [[70001]],
     "source": "stage source note", "quote": "少女级必做测试鞋"}
  ]
}`

func ruleEntries() []Entry {
	a := [5]string{"A", "A", "A", "A", "A"}
	return []Entry{
		graded(10001, scoring.Hair, "Test Hair", "hair", a),
		graded(20001, scoring.Dress, "Test Dress", "dress", a),
		graded(20002, scoring.Dress, "Other Dress", "dress", a),
		graded(40001, scoring.Top, "Test Top", "top", a),
		graded(50001, scoring.Bottom, "Test Bottom", "bottom", a),
		graded(70001, scoring.Shoes, "Test Shoes", "shoes", a),
		graded(83001, scoring.Accessory, "Test Hat", "accessory_headwear", a),
		graded(83002, scoring.Accessory, "Test Staff", "accessory_handheld_both", a),
		graded(83003, scoring.Accessory, "Test Fan", "accessory_handheld_right", a),
		graded(83004, scoring.Accessory, "Test Sword", "accessory_handheld_left", a),
	}
}

func ruleStages(t *testing.T) ([]Stage, StageStats) {
	t.Helper()
	stages, stats, err := ParseStages([]byte(ruleSrc))
	if err != nil {
		t.Fatal(err)
	}
	for i := range stages {
		if stages[i].Name == "1-3" {
			stages[i].Variants = map[string]scoring.Stage{Maiden: stages[i].Stage}
		}
	}
	return stages, stats
}

func TestApplyStageRules(t *testing.T) {
	rules, err := ReadStageRules([]byte(ruleJSON))
	if err != nil {
		t.Fatal(err)
	}
	stages, stats := ruleStages(t)
	if err := ApplyStageRules(stages, rules, ruleEntries(), []byte(ruleSrc), &stats); err != nil {
		t.Fatal(err)
	}
	by := map[string]Stage{}
	for _, s := range stages {
		by[s.Name] = s
	}
	for name, want := range map[string][][]int{
		"1-1": {{10001}, {20001}, {70001}},
		"1-2": {{40001, 20001}, {50001, 20001}},
		"1-3": {{83001, 83002}},
	} {
		if r := by[name].Rules; r == nil || !slices.EqualFunc(r.Require, want, slices.Equal) {
			t.Errorf("%s requires %v, want %v", name, r, want)
		}
	}
	maiden, ok := by["1-3"].VariantRules[Maiden]
	if want := [][]int{{83001, 83002}, {70001}}; !ok || !slices.EqualFunc(maiden.Require, want, slices.Equal) {
		t.Errorf("1-3 Maiden requires %v, want %v", maiden.Require, want)
	}
	if by["1-1"].VariantRules != nil || stats.Required != 3 {
		t.Errorf("1-1 variant rules %v, Required %d; want none and 3", by["1-1"].VariantRules, stats.Required)
	}

	out := string(WriteStages(stages))
	for _, want := range []string{
		`"rules":{"styles":[],"require":[[10001],[20001],[70001]]}`,
		`"rules":{"styles":[],"require":[[40001,20001],[50001,20001]]}`,
		`"variants":{"maiden":{"weights":[15,15,15,15,15],"attrs":[1,3,5,7,9],"rules":{"styles":[],"require":[[83001,83002],[70001]]}}}`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("stages.json lacks %s:\n%s", want, out)
		}
	}
}

func TestApplyStageRulesRefusesWhatCannotBeWorn(t *testing.T) {
	for _, c := range []struct{ name, json, src, want string }{
		{"a stage the bundle lacks",
			strings.Replace(ruleJSON, `"stage": "1-2"`, `"stage": "1-9"`, 1), ruleSrc, "1-9"},
		{"a quote the stage source no longer has",
			ruleJSON, strings.Replace(ruleSrc, "测试鞋'],", "测试靴'],", 1), "必做测试发型"},
		{"an item the catalogue lacks",
			strings.Replace(ruleJSON, "[[10001]", "[[99999]", 1), ruleSrc, "99999"},
		{"a choice between a hair and shoes",
			strings.Replace(ruleJSON, "[[10001], [20001]", "[[10001, 70001], [20001]", 1), ruleSrc, "one choice"},
		{"two dresses",
			strings.Replace(ruleJSON, "[[10001], [20001]", "[[20002], [20001]", 1), ruleSrc, "never be worn together"},
		{"a dress and a top",
			strings.Replace(ruleJSON, "[[10001], [20001]", "[[40001], [20001]", 1), ruleSrc, "never be worn together"},
		{"a two-handed item and a right-hand one",
			strings.Replace(ruleJSON, "[[83001, 83002]]", "[[83002], [83003]]", 1), ruleSrc, "never be worn together"},
		{"a Maiden rule on a stage with no Maiden numbers",
			strings.Replace(ruleJSON, `"stage": "1-3", "mode": "Story", "difficulty"`, `"stage": "1-1", "mode": "Story", "difficulty"`, 1),
			ruleSrc, "1-1"},
	} {
		t.Run(c.name, func(t *testing.T) {
			rules, err := ReadStageRules([]byte(c.json))
			if err != nil {
				t.Fatal(err)
			}
			stages, stats := ruleStages(t)
			if err := ApplyStageRules(stages, rules, ruleEntries(), []byte(c.src), &stats); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want one naming %s", err, c.want)
			}
		})
	}
}

func TestApplyStageRulesAcceptsBothHands(t *testing.T) {
	rules, err := ReadStageRules([]byte(strings.Replace(ruleJSON, "[[83001, 83002]]", "[[83003], [83004]]", 1)))
	if err != nil {
		t.Fatal(err)
	}
	stages, stats := ruleStages(t)
	if err := ApplyStageRules(stages, rules, ruleEntries(), []byte(ruleSrc), &stats); err != nil {
		t.Errorf("a right-hand and a left-hand item were refused: %v", err)
	}
}

func TestReadStageRulesRejectsMalformedEntries(t *testing.T) {
	for _, c := range []struct{ name, json string }{
		{"no mode", strings.Replace(ruleJSON, `"stage": "1-1", "mode": "Story"`, `"stage": "1-1", "mode": ""`, 1)},
		{"an unknown source", strings.Replace(ruleJSON, `"source": "wiki quest field"`, `"source": "memory"`, 1)},
		{"no quote", strings.Replace(ruleJSON, `"quote": "必做测试发型+测试长裙+测试鞋"`, `"quote": ""`, 1)},
		{"nothing required", strings.Replace(ruleJSON, `[[10001], [20001], [70001]]`, `[]`, 1)},
		{"an empty set", strings.Replace(ruleJSON, `[[10001], [20001], [70001]]`, `[[10001], []]`, 1)},
		{"an item twice in a set", strings.Replace(ruleJSON, `[[83001, 83002]]`, `[[83001, 83001]]`, 1)},
		{"an unknown difficulty", strings.Replace(ruleJSON, `"difficulty": "maiden"`, `"difficulty": "hard"`, 1)},
		{"a stage listed twice", strings.Replace(ruleJSON, `"stage": "1-2"`, `"stage": "1-1"`, 1)},
	} {
		if _, err := ReadStageRules([]byte(c.json)); err == nil {
			t.Errorf("%s: accepted", c.name)
		}
	}
}

func TestCommittedStageRulesParse(t *testing.T) {
	raw, err := os.ReadFile("../data/stage-rules.json")
	if err != nil {
		t.Fatal(err)
	}
	rules, err := ReadStageRules(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) == 0 {
		t.Error("the committed stage rules are empty")
	}
}
