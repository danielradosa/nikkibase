package pipeline

import (
	"strings"
	"testing"
)

const displaySrc = `var levelsRaw = {
  '1-1': [1, 1, 1, 1, 1],
};
var dreamWeavingRaw = {
'绫罗1-6': [1, 1, 1, 1, 1],
'奥兰多2-4': [1, 1, 1, 1, 1],
};`

func displayNames(t *testing.T, entries string) []StageDisplayEntry {
	t.Helper()
	names, err := ReadStageDisplayNames([]byte(`{"note": "test", "names": [` + entries + `]}`))
	if err != nil {
		t.Fatal(err)
	}
	return names
}

func displayStages(t *testing.T) []Stage {
	t.Helper()
	stages, _, err := ParseStages([]byte(displaySrc))
	if err != nil {
		t.Fatal(err)
	}
	return stages
}

func TestApplyStageDisplayNamesNamesEveryStageOfANamedMode(t *testing.T) {
	stages := displayStages(t)
	names := displayNames(t, `
		{"mode": "Dreamweaver", "stage": "Lunar 1-6", "name": "Lunar - Millennium Dream 6", "basis": "wiki"},
		{"mode": "Dreamweaver", "stage": "Orlando 2-4", "name": "Orlando - Officer & Wine 4", "basis": "wiki"}`)
	if err := ApplyStageDisplayNames(stages, names); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, s := range stages {
		got[s.Mode+"/"+DisplayName(s)] = s.Name
	}
	for _, want := range []string{"Dreamweaver/Lunar - Millennium Dream 6", "Dreamweaver/Orlando - Officer & Wine 4", "Story/1-1"} {
		if _, ok := got[want]; !ok {
			t.Errorf("no stage shows as %s; have %v", want, got)
		}
	}
}

func TestApplyStageDisplayNamesRefusesAGap(t *testing.T) {
	stages := displayStages(t)
	names := displayNames(t, `{"mode": "Dreamweaver", "stage": "Lunar 1-6", "name": "Lunar - Millennium Dream 6", "basis": "wiki"}`)
	err := ApplyStageDisplayNames(stages, names)
	if err == nil || !strings.Contains(err.Error(), "Orlando 2-4") {
		t.Errorf("got %v, want an error naming the unnamed Orlando 2-4", err)
	}
}

func TestApplyStageDisplayNamesRefusesAnEntryThatMatchesNothing(t *testing.T) {
	stages := displayStages(t)
	names := displayNames(t, `
		{"mode": "Dreamweaver", "stage": "Lunar 1-6", "name": "Lunar - Millennium Dream 6", "basis": "wiki"},
		{"mode": "Dreamweaver", "stage": "Orlando 2-4", "name": "Orlando - Officer & Wine 4", "basis": "wiki"},
		{"mode": "Dreamweaver", "stage": "Lunar 9-9", "name": "Lunar - Nowhere 9", "basis": "wiki"}`)
	err := ApplyStageDisplayNames(stages, names)
	if err == nil || !strings.Contains(err.Error(), "Lunar 9-9") {
		t.Errorf("got %v, want an error naming Lunar 9-9", err)
	}
}

func TestReadStageDisplayNamesRefusesBadEntries(t *testing.T) {
	for _, tc := range []struct{ name, entries string }{
		{"no basis", `{"mode": "Dreamweaver", "stage": "Lunar 1-6", "name": "Lunar - Millennium Dream 6", "basis": ""}`},
		{"no name", `{"mode": "Dreamweaver", "stage": "Lunar 1-6", "name": " ", "basis": "wiki"}`},
		{"no mode", `{"mode": "", "stage": "Lunar 1-6", "name": "Lunar - Millennium Dream 6", "basis": "wiki"}`},
		{"a stage twice", `{"mode": "Dreamweaver", "stage": "Lunar 1-6", "name": "A", "basis": "wiki"},
			{"mode": "Dreamweaver", "stage": "Lunar 1-6", "name": "B", "basis": "wiki"}`},
		{"a name twice", `{"mode": "Dreamweaver", "stage": "Lunar 1-6", "name": "A", "basis": "wiki"},
			{"mode": "Dreamweaver", "stage": "Lunar 2-2", "name": "A", "basis": "wiki"}`},
	} {
		if _, err := ReadStageDisplayNames([]byte(`{"names": [` + tc.entries + `]}`)); err == nil {
			t.Errorf("%s: read without an error", tc.name)
		}
	}
}
