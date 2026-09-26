package pipeline

import (
	"slices"
	"strings"
	"testing"
)

const scopeSrc = `var levelsRaw = {
  '1-1': [1, 1, 1, 1, 1],
  '2-支1': [1, 1, 1, 1, 1],
  '9-6-1': [1, 1, 1, 1, 1],
  'II-6-1': [1, 1, 1, 1, 1],
  'II-6-2': [1, 1, 1, 1, 1],
  'III-3-1': [1, 1, 1, 1, 1],
  'III-3-2': [1, 1, 1, 1, 1],
  'III-3-3': [1, 1, 1, 1, 1],
  'III-3-4': [1, 1, 1, 1, 1],
  'III-3-5': [1, 1, 1, 1, 1],
  'III-3-6': [1, 1, 1, 1, 1],
  'III-3-支1': [1, 1, 1, 1, 1],
  'III-3-支2': [1, 1, 1, 1, 1],
  'III-3-支3': [1, 1, 1, 1, 1],
  'III-4-1': [1, 1, 1, 1, 1],
};
var tasksRaw = {
'协战: 黑卡-海军': [1, 1, 1, 1, 1],
'联盟委托: 20-7': [1, 1, 1, 1, 1],
'联盟委托: 21-1': [1, 1, 1, 1, 1],
};
var competitionsRaw = {
'海边派对的搭配': [1, 1, 1, 1, 1],
};
var extraRaw = {
'倾心回忆-春与花恋': [1, 1, 1, 1, 1],
};
var dreamWeavingRaw = {
'绫罗1-6': [1, 1, 1, 1, 1],
};`

const scopeJSON = `{
  "note": "test",
  "story": [
    {"volume": 1, "chapters": [1, 19]},
    {"volume": 2, "chapters": [1, 15]},
    {"volume": 3, "chapters": [3, 3], "stages": ["1", "2", "3", "4", "5", "支1", "支2"]}
  ],
  "commissionActs": [1, 20],
  "whole": ["Arena", "Co-op"],
  "exclude": [{"stage": "II-6-2", "reason": "a placeholder"}]
}`

func TestApplyScopeKeepsGlobalStagesOnly(t *testing.T) {
	scope, err := ReadStageScope([]byte(scopeJSON))
	if err != nil {
		t.Fatal(err)
	}
	stages, _, err := ParseStages([]byte(scopeSrc))
	if err != nil {
		t.Fatal(err)
	}
	kept, out, err := ApplyScope(stages, scope)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, s := range kept {
		names = append(names, s.Name)
	}
	want := []string{"1-1", "2-支1", "9-6-1", "II-6-1", "III-3-1", "III-3-2", "III-3-3", "III-3-4",
		"III-3-5", "III-3-支1", "III-3-支2",
		"协战: 黑卡-海军", "联盟委托: 20-7", "海边派对的搭配"}
	if !slices.Equal(names, want) {
		t.Errorf("kept %v, want %v", names, want)
	}
	wantOut := map[string]int{"Story": 4, "Commission": 1, "Event": 1, "Dreamweaver": 1}
	for mode, n := range wantOut {
		if out[mode] != n {
			t.Errorf("out of scope %s = %d, want %d (all: %v)", mode, out[mode], n, out)
		}
	}
}

func TestApplyScopeRefusesEntriesMatchingNothing(t *testing.T) {
	stages, _, err := ParseStages([]byte(scopeSrc))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ name, scope, want string }{
		{"an exclusion naming no stage", strings.Replace(scopeJSON, `"II-6-2"`, `"II-6-9"`, 1), "II-6-9"},
		{"a listed stage the source lacks", strings.Replace(scopeJSON, `"支2"]`, `"支2", "7"]`, 1), "III-3-7"},
		{"a volume with no stages", strings.Replace(scopeJSON, `"volume": 2, "chapters": [1, 15]`, `"volume": 2, "chapters": [16, 20]`, 1), "volume 2"},
	} {
		t.Run(c.name, func(t *testing.T) {
			scope, err := ReadStageScope([]byte(c.scope))
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := ApplyScope(stages, scope); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want one naming %q", err, c.want)
			}
		})
	}
}

func TestReadStageScopeRejectsMalformedEntries(t *testing.T) {
	for _, c := range []struct{ name, scope string }{
		{"exclusion without a reason", strings.Replace(scopeJSON, `"reason": "a placeholder"`, `"reason": ""`, 1)},
		{"unknown whole mode", strings.Replace(scopeJSON, `"Co-op"`, `"Coop"`, 1)},
		{"backwards chapters", strings.Replace(scopeJSON, `[1, 19]`, `[19, 1]`, 1)},
		{"volume out of range", strings.Replace(scopeJSON, `"volume": 1,`, `"volume": 4,`, 1)},
		{"backwards acts", strings.Replace(scopeJSON, `"commissionActs": [1, 20]`, `"commissionActs": [20, 1]`, 1)},
	} {
		if _, err := ReadStageScope([]byte(c.scope)); err == nil {
			t.Errorf("%s: accepted", c.name)
		}
	}
}
