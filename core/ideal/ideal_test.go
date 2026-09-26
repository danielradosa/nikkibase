package ideal

import (
	"encoding/json"
	"math/rand/v2"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/catalogue"
	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
	"github.com/danielradosa/nikkibase/pipeline"
)

const (
	tagOne = 7
	tagTwo = 11
)

const handheld = 1

var (
	matches     = [5]int8{scoring.Simple, scoring.Lively, scoring.Cute, scoring.Sexy, scoring.Cool}
	offDeadPair = [5]int8{scoring.Simple, scoring.Lively, scoring.Mature, scoring.Sexy, scoring.Cool}
	offSexy     = [5]int8{scoring.Simple, scoring.Lively, scoring.Cute, scoring.Pure, scoring.Cool}
	offLively   = [5]int8{scoring.Simple, scoring.Elegant, scoring.Cute, scoring.Sexy, scoring.Cool}
	offGorgeous = [5]int8{scoring.Gorgeous, scoring.Lively, scoring.Cute, scoring.Sexy, scoring.Cool}
	offWarm     = [5]int8{scoring.Gorgeous, scoring.Lively, scoring.Cute, scoring.Sexy, scoring.Warm}
	offAll      = [5]int8{scoring.Gorgeous, scoring.Elegant, scoring.Mature, scoring.Pure, scoring.Warm}
)

var fixture = []catalogue.Item{
	item(10001, scoring.Hair, 0, offDeadPair, [5]int32{30, 45, 20, 10, 25}, tagOne),
	item(10002, scoring.Hair, 0, offSexy, [5]int32{40, 90, 60, 40, 30}),
	item(20001, scoring.Dress, 1, offWarm, [5]int32{80, 260, 40, 60, 30}, tagTwo),
	item(20002, scoring.Dress, 1, offAll, [5]int32{90, 90, 90, 90, 90}),
	item(30001, scoring.Coat, 2, offLively, [5]int32{12, 10, 8, 6, 14}),
	item(40001, scoring.Top, 3, offSexy, [5]int32{55, 70, 30, 25, 40}, tagOne),
	item(40002, scoring.Top, 3, offAll, [5]int32{120, 120, 120, 120, 120}),
	item(50001, scoring.Bottom, 4, offDeadPair, [5]int32{50, 65, 35, 20, 45}),
	item(50002, scoring.Bottom, 4, offAll, [5]int32{110, 110, 110, 110, 110}, tagTwo),
	item(60001, scoring.Hosiery, 5, matches, [5]int32{18, 22, 10, 8, 15}, tagTwo),
	item(70001, scoring.Shoes, 6, offGorgeous, [5]int32{20, 28, 12, 10, 18}, tagOne),
	item(80001, scoring.Makeup, 7, matches, [5]int32{6, 8, 4, 3, 5}),
	item(170001, scoring.Accessory, 8, matches, [5]int32{14, 16, 8, 6, 10}, tagOne),
	item(170002, scoring.Accessory, 9, offLively, [5]int32{12, 30, 6, 4, 9}, tagTwo),
	item(170003, scoring.Accessory, 10, matches, [5]int32{4, 5, 3, 2, 4}, tagOne, tagTwo),
	item(170004, scoring.Accessory, 11, offWarm, [5]int32{10, 12, 5, 4, 8}),
	item(170005, scoring.Accessory, 12, offAll, [5]int32{10, 10, 10, 10, 10}),
	item(170006, scoring.Accessory, 12, offGorgeous, [5]int32{5, 5, 5, 5, 5}),
	spirit(880001, 13, matches, [5]int32{9, 11, 5, 4, 7}, 137, tagTwo),
	spirit(880002, 13, offAll, [5]int32{1, 1, 1, 1, 1}, 400),
	held(170007, 14, false, offGorgeous, [5]int32{12, 12, 6, 5, 7}),
	held(170008, 15, false, matches, [5]int32{10, 11, 5, 5, 7}),
	held(170009, 16, true, offSexy, [5]int32{16, 15, 9, 6, 11}),
	held(170010, 16, true, offAll, [5]int32{40, 40, 40, 40, 40}, tagOne),
}

func item(id int32, slot scoring.Slot, pos uint16, attrs [5]int8, stats [5]int32, tags ...int32) catalogue.Item {
	return catalogue.Item{ID: id, Slot: uint8(slot), Position: pos, Attrs: attrs, Stats: stats, Tags: tags}
}

func spirit(id int32, pos uint16, attrs [5]int8, stats [5]int32, flat int32, tags ...int32) catalogue.Item {
	it := item(id, scoring.Spirit, pos, attrs, stats, tags...)
	it.FlatBonus = flat
	return it
}

func held(id int32, pos uint16, bothHands bool, attrs [5]int8, stats [5]int32, tags ...int32) catalogue.Item {
	it := item(id, scoring.Accessory, pos, attrs, stats, tags...)
	it.Group = handheld
	if bothHands {
		it.Group |= catalogue.WholeGroup
	}
	return it
}

func load(t *testing.T) ([]optimizer.Position, map[int]int, map[int]uint16) {
	t.Helper()
	c, err := catalogue.Read(catalogue.Write(fixture))
	if err != nil {
		t.Fatal(err)
	}
	return optimizer.FromCatalogue(c, nil)
}

func placeInFixture(t *testing.T, id int) int {
	t.Helper()
	for _, it := range fixture {
		if int(it.ID) == id {
			return int(it.Position)
		}
	}
	t.Fatalf("no item %d in the fixture", id)
	return -1
}

func randomStage(r *rand.Rand) scoring.Stage {
	var st scoring.Stage
	for p := range 5 {
		st.Attrs[p] = int8(p*2 + r.IntN(2))
		st.Weights[p] = float64(r.IntN(9)) + float64(r.IntN(8))/8
	}
	if r.IntN(3) > 0 {
		st.Tags = map[int]int{}
		for _, tag := range []int{tagOne, tagTwo} {
			if r.IntN(2) == 0 {
				st.Tags[tag] = 100 * (1 + r.IntN(40))
			}
		}
	}
	return st
}

func randomStages(n int) []Stage {
	r := rand.New(rand.NewPCG(20260923, 605))
	stages := make([]Stage, n)
	for i := range stages {
		stages[i] = Stage{Mode: []string{"Story", "Arena", "Co-op"}[i%3], Name: strings.Repeat("x", i+1), Scoring: randomStage(r)}
		if i%4 == 0 {
			stages[i].Variants = map[string]scoring.Stage{"maiden": randomStage(r)}
		}
		if i%8 == 0 {
			stages[i].Variants["hard"] = randomStage(r)
		}
	}
	return stages
}

func TestOfMatchesBest(t *testing.T) {
	positions, posOf, placeOf := load(t)
	var dresses, separates, bothHands, oneHand, spirits int
	placements := map[scoring.Placement]bool{}
	for i, s := range randomStages(60) {
		got := Of(positions, posOf, placeOf, s.Scoring, nil)
		want := optimizer.Best(positions, s.Scoring, nil)
		if got.Score != want.Score {
			t.Errorf("stage %d: score %d, and Best says %d", i, got.Score, want.Score)
		}
		if withSkills := optimizer.Best(positions, s.Scoring, scoring.Skills{}); withSkills.Score != want.Score {
			t.Errorf("stage %d: no skills scores %d and an empty skill set %d", i, want.Score, withSkills.Score)
		}
		auto, placement := optimizer.BestPlaced(positions, s.Scoring)
		if got.Auto.Placement != placement || got.Auto.Score != auto.Score || len(got.Auto.Items) != len(auto.Items) {
			t.Errorf("stage %d: auto %+v scores %d wearing %d, and BestPlaced %+v scores %d wearing %d",
				i, got.Auto.Placement, got.Auto.Score, len(got.Auto.Items), placement, auto.Score, len(auto.Items))
		} else {
			for j, it := range auto.Items {
				if got.Auto.Items[j] != [2]int{it.ID, placeInFixture(t, it.ID)} {
					t.Errorf("stage %d auto item %d: %v, want [%d %d]", i, j, got.Auto.Items[j], it.ID, placeInFixture(t, it.ID))
				}
			}
		}
		if got.Auto.Score < got.Score {
			t.Errorf("stage %d: auto scores %d, below %d without skills", i, got.Auto.Score, got.Score)
		}
		placements[got.Auto.Placement] = true
		if len(got.Items) != len(want.Items) {
			t.Errorf("stage %d: %d items, and Best wears %d", i, len(got.Items), len(want.Items))
			continue
		}
		for j, it := range want.Items {
			if got.Items[j] != [2]int{it.ID, placeInFixture(t, it.ID)} {
				t.Errorf("stage %d item %d: %v, want [%d %d]", i, j, got.Items[j], it.ID, placeInFixture(t, it.ID))
			}
			switch {
			case it.Slot == scoring.Dress:
				dresses++
			case it.Slot == scoring.Top:
				separates++
			case it.Slot == scoring.Spirit:
				spirits++
			case it.ID == 170009 || it.ID == 170010:
				bothHands++
			case it.ID == 170007 || it.ID == 170008:
				oneHand++
			}
		}
	}
	if dresses == 0 || separates == 0 || bothHands == 0 || oneHand == 0 || spirits == 0 {
		t.Errorf("the stages no longer exercise every choice: %d dresses, %d separates, %d both-hands, %d one-hand, %d spirits",
			dresses, separates, bothHands, oneHand, spirits)
	}
	if len(placements) < 8 {
		t.Errorf("the stages placed the skills only %d ways", len(placements))
	}
}

func TestOfWearsNothingFromAnEmptyCatalogue(t *testing.T) {
	got := Of(nil, nil, nil, randomStage(rand.New(rand.NewPCG(1, 1))), [][]int{{10001}})
	if got.Score != 0 || len(got.Items) != 0 || got.Auto.Score != 0 || len(got.Auto.Items) != 0 {
		t.Errorf("an empty catalogue gave %+v", got)
	}
	if enc := string(Encode("v", []Ideal{{Key: "Story/1-1", Outfit: got}})); !strings.Contains(enc, `"items":[]`) {
		t.Errorf("an empty outfit encodes as %s, want an empty items list", enc)
	}
}

func clonePositions(positions []optimizer.Position) []optimizer.Position {
	out := slices.Clone(positions)
	for i := range out {
		out[i].Items = slices.Clone(out[i].Items)
		for j := range out[i].Items {
			out[i].Items[j].Tags = slices.Clone(out[i].Items[j].Tags)
		}
	}
	return out
}

func TestParallelMatchesSerial(t *testing.T) {
	positions, posOf, placeOf := load(t)
	before := clonePositions(positions)
	stages := randomStages(120)

	serial, err := All(positions, posOf, placeOf, stages, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := Encode("test", serial)
	for _, workers := range []int{0, 2, 8, 64, 500} {
		got, err := All(positions, posOf, placeOf, stages, workers)
		if err != nil {
			t.Fatal(err)
		}
		if enc := Encode("test", got); string(enc) != string(want) {
			t.Errorf("%d workers encode differently from one", workers)
		}
	}
	if !reflect.DeepEqual(positions, before) {
		t.Error("computing the outfits changed the shared positions")
	}
	if !json.Valid(want) {
		t.Errorf("the encoding is not JSON: %s", want)
	}
}

func TestVariants(t *testing.T) {
	positions, posOf, placeOf := load(t)
	stages := randomStages(3)
	stages[0].Variants = map[string]scoring.Stage{"maiden": stages[1].Scoring, "hard": stages[2].Scoring}
	stages[1].Variants = nil

	ideals, err := All(positions, posOf, placeOf, stages, 4)
	if err != nil {
		t.Fatal(err)
	}
	if n := Versions(ideals); n != 5 {
		t.Errorf("%d stage versions, want 5", n)
	}
	first := ideals[0]
	if first.Key != "Story/x" || len(first.Variants) != 2 {
		t.Fatalf("first stage = %+v", first)
	}
	for name, st := range stages[0].Variants {
		if want := Of(positions, posOf, placeOf, st, nil); !reflect.DeepEqual(first.Variants[name], want) {
			t.Errorf("%s: %+v, want %+v", name, first.Variants[name], want)
		}
	}
	if !reflect.DeepEqual(first.Variants["maiden"], ideals[1].Outfit) {
		t.Error("the maiden variant and the stage with the same numbers disagree")
	}
	if ideals[1].Variants != nil {
		t.Errorf("a stage without variants got %v", ideals[1].Variants)
	}

	var decoded struct {
		Stages map[string]map[string]json.RawMessage
	}
	enc := Encode("test", ideals)
	if err := json.Unmarshal(enc, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded.Stages["Arena/xx"]["variants"]; ok {
		t.Error("a stage without variants encodes a variants key")
	}

	type auto struct {
		CharmSmile *int     `json:"charmSmile"`
		Smile      *int     `json:"smile"`
		Score      *int     `json:"score"`
		Items      [][2]int `json:"items"`
	}
	var shapes struct {
		Stages map[string]struct {
			Auto     *auto `json:"auto"`
			Variants map[string]struct {
				Auto *auto `json:"auto"`
			} `json:"variants"`
		}
	}
	if err := json.Unmarshal(enc, &shapes); err != nil {
		t.Fatal(err)
	}
	check := func(label string, got *auto, want Outfit) {
		t.Helper()
		if got == nil || got.CharmSmile == nil || got.Smile == nil || got.Score == nil {
			t.Errorf("%s: no complete auto entry in %s", label, enc)
			return
		}
		if placed := (scoring.Placement{CharmSmile: *got.CharmSmile, Smile: *got.Smile}); placed != want.Auto.Placement ||
			*got.Score != want.Auto.Score || !reflect.DeepEqual(got.Items, want.Auto.Items) {
			t.Errorf("%s: auto encodes as %+v %d %v, want %+v", label, placed, *got.Score, got.Items, want.Auto)
		}
	}
	for i, id := range ideals {
		check(id.Key, shapes.Stages[id.Key].Auto, id.Outfit)
		for name, v := range id.Variants {
			check(id.Key+" "+name, shapes.Stages[id.Key].Variants[name].Auto, v)
		}
		if i == 0 && len(shapes.Stages[id.Key].Variants) != 2 {
			t.Errorf("%s: %d variants decoded", id.Key, len(shapes.Stages[id.Key].Variants))
		}
	}
	if i, j := strings.Index(string(enc), `"hard":`), strings.Index(string(enc), `"maiden":`); i < 0 || j < 0 || i > j {
		t.Errorf("variants are not in name order: %s", enc)
	}
}

func TestEncode(t *testing.T) {
	got := string(Encode("2026-09-23", []Ideal{
		{Key: "Story/1-1", Outfit: Outfit{119836, [][2]int{{10123, 0}, {20456, 1}},
			Auto{scoring.Placement{CharmSmile: scoring.Lively, Smile: scoring.Cute}, 161665, [][2]int{{10124, 0}, {20456, 1}}}},
			Variants: map[string]Outfit{
				"maiden": {1, [][2]int{{1, 2}}, Auto{scoring.Placement{CharmSmile: scoring.Warm, Smile: scoring.Simple}, 3, [][2]int{{1, 2}}}},
				"hard":   {Score: 2}}},
		{Key: `Arena/Say "Hi" & <Bye>`, Outfit: Outfit{Score: 5, Auto: Auto{Placement: scoring.Placement{CharmSmile: -1, Smile: -1}, Score: 5}}},
		{Key: "Co-op/Neva - Maiden", Outfit: Outfit{Score: 7, Items: [][2]int{{3, 4}}}, Variants: map[string]Outfit{}},
	}))
	want := `{"version":"2026-09-23","stages":{` +
		`"Story/1-1":{"score":119836,"items":[[10123,0],[20456,1]],"auto":{"charmSmile":3,"smile":5,"score":161665,"items":[[10124,0],[20456,1]]},` +
		`"variants":{"hard":{"score":2,"items":[],"auto":{"charmSmile":0,"smile":0,"score":0,"items":[]}},` +
		`"maiden":{"score":1,"items":[[1,2]],"auto":{"charmSmile":8,"smile":1,"score":3,"items":[[1,2]]}}}},` +
		`"Arena/Say \"Hi\" & <Bye>":{"score":5,"items":[],"auto":{"charmSmile":-1,"smile":-1,"score":5,"items":[]}},` +
		`"Co-op/Neva - Maiden":{"score":7,"items":[[3,4]],"auto":{"charmSmile":0,"smile":0,"score":0,"items":[]}}}}`
	if got != want {
		t.Errorf("Encode =\n%s\nwant\n%s", got, want)
	}
	if empty := string(Encode("v", nil)); empty != `{"version":"v","stages":{}}` {
		t.Errorf("no stages encode as %s", empty)
	}
}

func TestDuplicateStageKey(t *testing.T) {
	positions, posOf, placeOf := load(t)
	stages := randomStages(4)
	stages[3].Mode, stages[3].Name = stages[0].Mode, stages[0].Name
	_, err := All(positions, posOf, placeOf, stages, 2)
	if err == nil || !strings.Contains(err.Error(), `"Story/x"`) {
		t.Errorf("err = %v, want one naming Story/x", err)
	}

	stages[3].Mode = "Arena"
	if _, err := All(positions, posOf, placeOf, stages, 2); err != nil {
		t.Errorf("the same name in two modes was refused: %v", err)
	}
}

const shippedShape = `[` +
	`{"name":"Beach Party","mode":"Arena","weights":[10,15,20,20,20],"attrs":[1,3,5,6,9]},` +
	`{"name":"Cloud Lady","mode":"Arena","weights":[20,20,15,20,10],"attrs":[0,2,4,7,8],"tags":{"26":10005}},` +
	`{"name":"2-Side 2","mode":"Story","weights":[50.625,25,15,8,8.5],"attrs":[1,2,4,7,9],"tags":{"47":8002,"3":12},"rules":{"styles":["Unisex"]},` +
	`"variants":{"maiden":{"weights":[15,25,15,8,8],"attrs":[1,2,4,7,9],"tags":{"47":16003}}}}]`

func TestReadStages(t *testing.T) {
	got, err := ReadStages([]byte(shippedShape))
	if err != nil {
		t.Fatal(err)
	}
	want := []Stage{
		{Mode: "Arena", Name: "Beach Party", Scoring: scoring.Stage{
			Weights: [5]float64{10, 15, 20, 20, 20}, Attrs: [5]int8{1, 3, 5, 6, 9}}},
		{Mode: "Arena", Name: "Cloud Lady", Scoring: scoring.Stage{
			Weights: [5]float64{20, 20, 15, 20, 10}, Attrs: [5]int8{0, 2, 4, 7, 8}, Tags: map[int]int{26: 10005}}},
		{Mode: "Story", Name: "2-Side 2", Scoring: scoring.Stage{
			Weights: [5]float64{50.625, 25, 15, 8, 8.5}, Attrs: [5]int8{1, 2, 4, 7, 9}, Tags: map[int]int{47: 8002, 3: 12}},
			Variants: map[string]scoring.Stage{"maiden": {
				Weights: [5]float64{15, 25, 15, 8, 8}, Attrs: [5]int8{1, 2, 4, 7, 9}, Tags: map[int]int{47: 16003}}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ReadStages =\n%+v\nwant\n%+v", got, want)
	}
}

func TestReadStagesReadsRequiredItems(t *testing.T) {
	raw := `[{"name":"2-7","mode":"Story","weights":[1,1,1,1,1],"attrs":[0,2,4,6,8],"rules":{"styles":[],"require":[[40080],[50091]]},` +
		`"variants":{"maiden":{"weights":[2,1,1,1,1],"attrs":[0,2,4,6,8]}}},` +
		`{"name":"II-2-2","mode":"Story","weights":[1,1,1,1,1],"attrs":[0,2,4,6,8],"rules":{"styles":[],"require":[[83283,83287]]},` +
		`"variants":{"maiden":{"weights":[2,1,1,1,1],"attrs":[0,2,4,6,8],"rules":{"styles":["Unisex"]}}}},` +
		`{"name":"2-Side 2","mode":"Story","weights":[1,1,1,1,1],"attrs":[0,2,4,6,8],"rules":{"styles":["Unisex"]},` +
		`"variants":{"maiden":{"weights":[2,1,1,1,1],"attrs":[0,2,4,6,8],"rules":{"styles":["Unisex"],"require":[[20165]]}}}}]`
	got, err := ReadStages([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []struct {
		require [][]int
		maiden  [][]int
	}{
		{[][]int{{40080}, {50091}}, [][]int{{40080}, {50091}}},
		{[][]int{{83283, 83287}}, nil},
		{nil, [][]int{{20165}}},
	} {
		if !reflect.DeepEqual(got[i].Require, want.require) || !reflect.DeepEqual(got[i].VariantRequire["maiden"], want.maiden) {
			t.Errorf("%s requires %v and on Maiden %v, want %v and %v",
				got[i].Key(), got[i].Require, got[i].VariantRequire["maiden"], want.require, want.maiden)
		}
	}
	if _, ok := got[1].VariantRequire["maiden"]; ok {
		t.Errorf("a variant whose own rules require nothing still requires %v", got[1].VariantRequire["maiden"])
	}
}

func TestOfWearsWhatTheStageRequires(t *testing.T) {
	positions, posOf, placeOf := load(t)
	worn := func(items [][2]int, id int) bool {
		return slices.ContainsFunc(items, func(it [2]int) bool { return it[0] == id })
	}
	var displaced, costly int
	for i, s := range randomStages(40) {
		free := Of(positions, posOf, placeOf, s.Scoring, nil)
		for _, require := range [][][]int{{{20002}}, {{40001}, {50001}}, {{170006}}, {{170005, 170008}}, {{10002}, {999999}}} {
			got := Of(positions, posOf, placeOf, s.Scoring, require)
			for _, o := range []struct {
				label string
				items [][2]int
			}{{"no skills", got.Items}, {"auto", got.Auto.Items}} {
				for _, set := range require {
					if set[0] != 999999 && !slices.ContainsFunc(set, func(id int) bool { return worn(o.items, id) }) {
						t.Errorf("stage %d %s: wore %v, which meets none of %v", i, o.label, o.items, set)
					}
				}
				if worn(o.items, 20002) && (worn(o.items, 40001) || worn(o.items, 50001)) {
					t.Errorf("stage %d %s: wore %v, a dress with separates", i, o.label, o.items)
				}
			}
			if got.Score > free.Score {
				t.Errorf("stage %d: %v scores %d, above %d without the rule", i, require, got.Score, free.Score)
			}
			if got.Score < free.Score {
				costly++
			}
			if require[0][0] == 40001 && worn(free.Items, 20001) {
				displaced++
			}
		}
	}
	if displaced == 0 || costly == 0 {
		t.Errorf("the rules never displaced a dress (%d) or cost points (%d)", displaced, costly)
	}
}

func TestAllAppliesEachVersionsRules(t *testing.T) {
	positions, posOf, placeOf := load(t)
	stages := randomStages(3)
	stages[0].Require = [][]int{{40001}}
	stages[0].VariantRequire = map[string][][]int{"maiden": {{170006}}}
	stages[1].Require = [][]int{{20002}}
	ideals, err := All(positions, posOf, placeOf, stages, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		label   string
		got     Outfit
		st      scoring.Stage
		require [][]int
	}{
		{"stage 0", ideals[0].Outfit, stages[0].Scoring, stages[0].Require},
		{"stage 0 maiden", ideals[0].Variants["maiden"], stages[0].Variants["maiden"], [][]int{{170006}}},
		{"stage 0 hard", ideals[0].Variants["hard"], stages[0].Variants["hard"], nil},
		{"stage 1", ideals[1].Outfit, stages[1].Scoring, [][]int{{20002}}},
		{"stage 2", ideals[2].Outfit, stages[2].Scoring, nil},
	} {
		if want := Of(positions, posOf, placeOf, c.st, c.require); !reflect.DeepEqual(c.got, want) {
			t.Errorf("%s: %+v, want %+v", c.label, c.got, want)
		}
	}
}

func TestReadStagesReadsWhatThePipelineWrites(t *testing.T) {
	written := []pipeline.Stage{
		{Name: "1-1", Display: "1-1", Mode: "Story",
			Stage: scoring.Stage{Weights: [5]float64{1.5, 0.25, 50.625, 3, 4}, Attrs: [5]int8{0, 3, 4, 7, 8}, Tags: map[int]int{5: 900, 12: 1800}},
			Rules: &pipeline.StageRules{Styles: []string{"Unisex"}},
			Variants: map[string]scoring.Stage{"maiden": {
				Weights: [5]float64{2, 2, 2, 2, 2}, Attrs: [5]int8{1, 2, 5, 6, 9}, Tags: map[int]int{5: 1800}}}},
		{Name: "Simple & Cool Cloud", Display: `Fu Su - "Simple" & Cool Cloud`, Mode: "Co-op",
			Stage: scoring.Stage{Weights: [5]float64{10, 20, 30, 40, 50}, Attrs: [5]int8{1, 3, 5, 7, 9}}},
		{Name: "11-3", Display: "11-3", Mode: "Story",
			Stage: scoring.Stage{Weights: [5]float64{1, 1, 1, 1, 1}, Attrs: [5]int8{0, 2, 4, 6, 8}},
			Rules: &pipeline.StageRules{Require: [][]int{{40413, 20617}, {50380, 20617}}},
			Variants: map[string]scoring.Stage{
				"maiden": {Weights: [5]float64{2, 2, 2, 2, 2}, Attrs: [5]int8{0, 2, 4, 6, 8}},
				"hard":   {Weights: [5]float64{3, 3, 3, 3, 3}, Attrs: [5]int8{0, 2, 4, 6, 8}}},
			VariantRules: map[string]pipeline.StageRules{"maiden": {Require: [][]int{{40413, 20617}, {50380, 20617}, {170001}}}}},
	}
	got, err := ReadStages(pipeline.WriteStages(written))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(written) {
		t.Fatalf("read %d stages from %d", len(got), len(written))
	}
	required := map[int]Stage{2: {
		Require: [][]int{{40413, 20617}, {50380, 20617}},
		VariantRequire: map[string][][]int{
			"maiden": {{40413, 20617}, {50380, 20617}, {170001}},
			"hard":   {{40413, 20617}, {50380, 20617}}}}}
	for i, w := range written {
		want := Stage{Mode: w.Mode, Name: pipeline.DisplayName(w), Scoring: w.Stage, Variants: w.Variants,
			Require: required[i].Require, VariantRequire: required[i].VariantRequire}
		if !reflect.DeepEqual(got[i], want) {
			t.Errorf("stage %d =\n%+v\nwant\n%+v", i, got[i], want)
		}
	}
	if key := got[1].Key(); key != `Co-op/Fu Su - "Simple" & Cool Cloud` {
		t.Errorf("key = %q", key)
	}
}

func TestReadStagesRefusesMalformedStages(t *testing.T) {
	for _, raw := range []string{
		`{"name":"not a list"}`,
		`[{"name":"1-1","mode":"Story","weights":[1,2,3,4],"attrs":[0,2,4,6,8]}]`,
		`[{"name":"1-1","mode":"Story","weights":[1,2,3,4,5],"attrs":[0,2,4,6]}]`,
		`[{"name":"1-1","mode":"Story","weights":[1,2,3,4,5],"attrs":[0,2,4,6,300]}]`,
		`[{"name":"1-1","mode":"Story","weights":[1,2,3,4,5],"attrs":[0,2,4,6,8],"tags":{"cute":10}}]`,
		`[{"name":"1-1","mode":"Story","weights":[1,2,3,4,5],"attrs":[0,2,4,6,8],"variants":{"maiden":{"weights":[1],"attrs":[0,2,4,6,8]}}}]`,
		`[{"name":"1-1","mode":"Story","weights":[1,2,3,4,5],"attrs":[0,2,4,6,8],"rules":{"styles":[],"require":[[10001],[]]}}]`,
		`[{"name":"1-1","mode":"Story","weights":[1,2,3,4,5],"attrs":[0,2,4,6,8],"rules":{"require":[["10001"]]}}]`,
		`[{"name":"1-1","mode":"Story","weights":[1,2,3,4,5],"attrs":[0,2,4,6,8],"variants":{"maiden":{"weights":[1,2,3,4,5],"attrs":[0,2,4,6,8],"rules":{"require":[[]]}}}}]`,
	} {
		if _, err := ReadStages([]byte(raw)); err == nil {
			t.Errorf("ReadStages accepted %s", raw)
		}
	}
}
