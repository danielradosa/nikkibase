package pipeline

import (
	"maps"
	"os"
	"slices"
	"strings"
	"testing"
)

const variantSrc = `var levelsRaw = {
'6-9': [-1.4, -2.87, -4.2, -2.87, 1.4],
'6-10': [-0.87, 2.53, -1.67, 1, -2],
'7-支3': [0.13, -0.13, 0.07, -0.07, 0.07],
'8-1': [1, 1, 1, 1, 1],
};
var levelBonus = {
'6-9': [addBonusInfo('F', 35.5, '民国服饰'), addBonusInfo('F', 31.4, '异域风')],
'6-10': [addBonusInfo('B', 1, "中式现代"), addBonusInfo('B', 1, "冬装")],
'7-支3': [addBonusInfo('SS', 20, "军装")],
};
var addHintInfo = {
'6-10' :[['少女级权重是0.67,1,0.8,0.67,0.8，tag更重'],[''],['']],
'7-支3' :[['少女级权重0.2,0.2,0.13,0.13,0.13，tag减半'],[''],['']],
'8-1' :[['上下装容易F！'],[''],['']],
};`

const variantJSON = `{
  "note": "test",
  "variants": [
    {"stage": "6-9", "difficulty": "maiden",
     "bonus": [{"grade": "F", "multiplier": 31.4, "tag": "旗袍"}, {"grade": "F", "multiplier": 35.5, "tag": "民国服饰"}],
     "basis": "the wiki's Maiden tags"},
    {"stage": "6-10", "difficulty": "maiden", "weights": [-0.67, 1, -0.8, 0.67, -0.8], "keepAwards": true,
     "basis": "Maiden weights, tags heavier", "quote": "少女级权重是0.67,1,0.8,0.67,0.8"},
    {"stage": "7-支3", "difficulty": "maiden", "weights": [0.2, -0.2, 0.13, -0.13, 0.13],
     "bonus": [{"grade": "SS", "multiplier": 10, "tag": "军装"}],
     "basis": "Maiden weights, tag halved", "quote": "少女级权重0.2,0.2,0.13,0.13,0.13，tag减半"}
  ]
}`

func TestApplyVariants(t *testing.T) {
	variants, err := ReadStageVariants([]byte(variantJSON))
	if err != nil {
		t.Fatal(err)
	}
	stages, stats, err := ParseStages([]byte(variantSrc))
	if err != nil {
		t.Fatal(err)
	}
	before := map[string]map[int]int{}
	for _, s := range stages {
		before[s.Name] = maps.Clone(s.Stage.Tags)
	}
	if err := ApplyVariants(stages, variants, []byte(variantSrc), &stats); err != nil {
		t.Fatal(err)
	}
	id := func(name string) int {
		i, _ := TagID(name)
		return i
	}
	by := map[string]Stage{}
	for _, s := range stages {
		by[s.Name] = s
		if !maps.Equal(s.Stage.Tags, before[s.Name]) {
			t.Errorf("%s: its own tags changed to %v", s.Name, s.Stage.Tags)
		}
	}
	for _, c := range []struct {
		stage   string
		weights [5]float64
		tags    map[int]int
	}{
		{"6-9", [5]float64{21, 63, 43, 43, 21}, map[int]int{id("Cheongsam"): 5997, id("Republic of China"): 6781}},
		{"6-10", [5]float64{10, 12, 15, 10, 12}, map[int]int{id("Modern China"): 10563, id("Winter"): 10563}},
		{"7-支3", [5]float64{3, 2, 3, 2, 2}, map[int]int{id("Army"): 20904}},
	} {
		v, ok := by[c.stage].Variants["maiden"]
		if !ok {
			t.Errorf("%s has no Maiden variant", c.stage)
			continue
		}
		if v.Weights != c.weights || !maps.Equal(v.Tags, c.tags) {
			t.Errorf("%s Maiden = %v %v, want %v %v", c.stage, v.Weights, v.Tags, c.weights, c.tags)
		}
		if v.Attrs != by[c.stage].Stage.Attrs {
			t.Errorf("%s Maiden sides %v differ from the stage's %v", c.stage, v.Attrs, by[c.stage].Stage.Attrs)
		}
	}
	if by["8-1"].Variants != nil || stats.Variants != 3 {
		t.Errorf("8-1 variants %v, Variants %d; want none and 3", by["8-1"].Variants, stats.Variants)
	}
}

func TestCommittedMaidenAwardsOn69(t *testing.T) {
	raw, err := os.ReadFile("../data/stage-difficulty.json")
	if err != nil {
		t.Fatal(err)
	}
	variants, err := ReadStageVariants(raw)
	if err != nil {
		t.Fatal(err)
	}
	at := slices.IndexFunc(variants, func(v StageVariant) bool { return v.Stage == "6-9" })
	if at < 0 {
		t.Fatal("the committed file has no 6-9 variant")
	}
	src := `var levelsRaw = {
'6-9': [-1.4, -2.87, -4.2, -2.87, 1.4],
};
var levelBonus = {
'6-9': [addBonusInfo('C', 0.5, "旗袍"), addBonusInfo('C', 0.5, "民国服饰")],
};`
	stages, stats, err := ParseStages([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyVariants(stages, variants[at:at+1], []byte(src), &stats); err != nil {
		t.Fatal(err)
	}
	cheongsam, _ := TagID("Cheongsam")
	republic, _ := TagID("Republic of China")
	want := map[int]int{cheongsam: 5997, republic: 6781}
	if got := stages[0].Variants[Maiden].Tags; !maps.Equal(got, want) {
		t.Errorf("6-9 Maiden pays %v, want %v", got, want)
	}
}

func TestApplyVariantsHoldsToTheSource(t *testing.T) {
	stages, stats, err := ParseStages([]byte(variantSrc))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ name, json, src, want string }{
		{"a hinted stage with no variant",
			strings.Replace(variantJSON, `"stage": "6-10"`, `"stage": "8-1"`, 1), variantSrc, "6-10"},
		{"a quote the source no longer has",
			variantJSON, strings.Replace(variantSrc, "tag减半", "tag不变", 1), "7-支3"},
		{"a variant that moves a weight to the other side",
			strings.Replace(variantJSON, `[0.2, -0.2, 0.13`, `[0.2, 0.2, 0.13`, 1), variantSrc, "7-支3"},
		{"a hinted stage with no variant, among others",
			variantJSON, strings.Replace(variantSrc, `'8-1' :[['上下装容易F！']`, `'8-1' :[['少女级tag是加分Ax1']`, 1), "8-1"},
		{"a variant for a stage the bundle lacks",
			strings.Replace(variantJSON, `"stage": "6-9"`, `"stage": "6-99"`, 1), variantSrc, "6-99"},
	} {
		t.Run(c.name, func(t *testing.T) {
			variants, err := ReadStageVariants([]byte(c.json))
			if err != nil {
				t.Fatal(err)
			}
			if err := ApplyVariants(stages, variants, []byte(c.src), &stats); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want one naming %s", err, c.want)
			}
		})
	}
}

func TestReadStageVariantsRejectsMalformedEntries(t *testing.T) {
	for _, c := range []struct{ name, json string }{
		{"no basis", strings.Replace(variantJSON, `"basis": "the wiki's Maiden tags"`, `"basis": ""`, 1)},
		{"unknown difficulty", strings.Replace(variantJSON, `"difficulty": "maiden",
     "bonus"`, `"difficulty": "hard",
     "bonus"`, 1)},
		{"both new awards and kept ones", strings.Replace(variantJSON, `"keepAwards": true,`, `"keepAwards": true, "bonus": [],`, 1)},
		{"neither new awards nor kept ones", strings.Replace(variantJSON, `, "keepAwards": true`, ``, 1)},
		{"an unknown tag", strings.Replace(variantJSON, `"旗袍"`, `"不存在"`, 1)},
		{"an unknown grade", strings.Replace(variantJSON, `"grade": "SS"`, `"grade": "Q"`, 1)},
		{"zero weights", strings.Replace(variantJSON, `[-0.67, 1, -0.8, 0.67, -0.8]`, `[0, 0, 0, 0, 0]`, 1)},
		{"a stage listed twice", strings.Replace(variantJSON, `"stage": "6-10"`, `"stage": "6-9"`, 1)},
	} {
		if _, err := ReadStageVariants([]byte(c.json)); err == nil {
			t.Errorf("%s: accepted", c.name)
		}
	}
}

func TestApplyVariantsIgnoresNotesOnStagesNotCarried(t *testing.T) {
	variants, err := ReadStageVariants([]byte(variantJSON))
	if err != nil {
		t.Fatal(err)
	}
	src := strings.Replace(variantSrc, `'8-1' :[['上下装容易F！'],[''],['']],`,
		`'8-1' :[['上下装容易F！'],[''],['']],
'III-9-1' :[['少女级tag是加分Ax1'],[''],['']],`, 1)
	stages, stats, err := ParseStages([]byte(variantSrc))
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyVariants(stages, variants, []byte(src), &stats); err != nil {
		t.Errorf("a note on a stage not carried failed the build: %v", err)
	}
}

func TestWriteStagesWritesVariants(t *testing.T) {
	variants, err := ReadStageVariants([]byte(variantJSON))
	if err != nil {
		t.Fatal(err)
	}
	stages, stats, err := ParseStages([]byte(variantSrc))
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyVariants(stages, variants, []byte(variantSrc), &stats); err != nil {
		t.Fatal(err)
	}
	out := string(WriteStages(stages))
	want := `"variants":{"maiden":{"weights":[3,2,3,2,2],"attrs":[1,3,4,6,9],"tags":{"2":20904}}}`
	if !strings.Contains(out, want) {
		t.Errorf("stages.json lacks %s:\n%s", want, out)
	}
	if strings.Count(out, `"variants"`) != 3 {
		t.Errorf("want variants on exactly 3 stages:\n%s", out)
	}
}
