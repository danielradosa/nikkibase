package parity

import (
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/catalogue"
	"github.com/danielradosa/nikkibase/core/ideal"
	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
	"github.com/danielradosa/nikkibase/core/wardrobe"
)

var update = flag.Bool("update", false, "rewrite the golden file from the native result")

const goldenPath = "testdata/engine.golden"

const (
	tagOne = 7
	tagTwo = 11
)

const handheld = 1

var (
	stageWeights = [5]float64{2, 4, 0, 1, 3}
	stageAttrs   = [5]int8{scoring.Simple, scoring.Lively, scoring.Cute, scoring.Sexy, scoring.Cool}
	stageTags    = map[int]int{tagOne: 120, tagTwo: 60}
	stageSkills  = scoring.Skills{scoring.Lively: scoring.CharmingSmile}
)

var (
	matches         = [5]int8{scoring.Simple, scoring.Lively, scoring.Cute, scoring.Sexy, scoring.Cool}
	offDeadPair     = [5]int8{scoring.Simple, scoring.Lively, scoring.Mature, scoring.Sexy, scoring.Cool}
	offSexy         = [5]int8{scoring.Simple, scoring.Lively, scoring.Cute, scoring.Pure, scoring.Cool}
	offLively       = [5]int8{scoring.Simple, scoring.Elegant, scoring.Cute, scoring.Sexy, scoring.Cool}
	offGorgeous     = [5]int8{scoring.Gorgeous, scoring.Lively, scoring.Cute, scoring.Sexy, scoring.Cool}
	offFirstAndLast = [5]int8{scoring.Gorgeous, scoring.Lively, scoring.Cute, scoring.Sexy, scoring.Warm}
	offEverything   = [5]int8{scoring.Gorgeous, scoring.Elegant, scoring.Mature, scoring.Pure, scoring.Warm}
)

const (
	betterHair = 10002
	weakDress  = 20002
	rightHand  = 170007
)

var unowned = []int32{betterHair, rightHand}

var fixture = []catalogue.Item{
	item(10001, scoring.Hair, 0, offDeadPair, [5]int32{30, 45, 20, 10, 25}, tagOne),
	item(betterHair, scoring.Hair, 0, offSexy, [5]int32{40, 90, 60, 40, 30}),
	item(20001, scoring.Dress, 1, offFirstAndLast, [5]int32{80, 260, 40, 60, 30}, tagTwo),
	item(weakDress, scoring.Dress, 1, matches, [5]int32{20, 30, 10, 10, 20}),
	item(30001, scoring.Coat, 2, offLively, [5]int32{12, 10, 8, 6, 14}),
	item(40001, scoring.Top, 3, offSexy, [5]int32{55, 70, 30, 25, 40}, tagOne),
	item(50001, scoring.Bottom, 4, offDeadPair, [5]int32{50, 65, 35, 20, 45}),
	item(60001, scoring.Hosiery, 5, matches, [5]int32{18, 22, 10, 8, 15}, tagTwo),
	item(70001, scoring.Shoes, 6, offGorgeous, [5]int32{20, 28, 12, 10, 18}, tagOne),
	item(80001, scoring.Makeup, 7, matches, [5]int32{6, 8, 4, 3, 5}),
	item(170001, scoring.Accessory, 8, matches, [5]int32{14, 16, 8, 6, 10}, tagOne),
	item(170002, scoring.Accessory, 9, offLively, [5]int32{12, 30, 6, 4, 9}, tagTwo),
	item(170003, scoring.Accessory, 10, matches, [5]int32{4, 5, 3, 2, 4}, tagOne, tagTwo),
	item(170004, scoring.Accessory, 11, offFirstAndLast, [5]int32{10, 12, 5, 4, 8}),
	item(170005, scoring.Accessory, 12, offEverything, [5]int32{10, 10, 10, 10, 10}),
	item(170006, scoring.Accessory, 13, offEverything, [5]int32{5, 5, 5, 5, 5}),
	spirit(880001, 14, matches, [5]int32{9, 11, 5, 4, 7}, 137, tagTwo),
	held(rightHand, 15, false, offGorgeous, [5]int32{12, 12, 6, 5, 7}),
	held(170008, 16, false, matches, [5]int32{10, 11, 5, 5, 7}),
	held(170009, 17, true, offSexy, [5]int32{16, 15, 9, 6, 11}),
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

func TestEngineParity(t *testing.T) {
	packed := catalogue.Write(fixture)
	file := selectionsFile(ownedIDs())
	native := render(nativeRun(t, packed, file))

	if *update {
		if err := os.WriteFile(goldenPath, []byte(native), 0o644); err != nil {
			t.Fatalf("writing %s: %v", goldenPath, err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading %s (regenerate with -update): %v", goldenPath, err)
	}

	t.Run("native", func(t *testing.T) {
		if native != string(want) {
			t.Errorf("the native build no longer matches %s.\n--- got ---\n%s--- want ---\n%s",
				goldenPath, native, want)
		}
	})

	t.Run("wasm", func(t *testing.T) {
		browser := render(browserRun(t, packed, file))
		if browser != native {
			t.Errorf("the browser build and the native build disagree -- they share core/, so one of the two wrappers has drifted.\n--- wasm ---\n%s--- native ---\n%s",
				browser, native)
		}
		if browser != string(want) {
			t.Errorf("the browser build no longer matches %s.\n--- got ---\n%s--- want ---\n%s",
				goldenPath, browser, want)
		}
	})
}

func TestPlacesOfEveryItem(t *testing.T) {
	var r struct {
		Early  string `json:"early"`
		Places struct {
			IDs    []int `json:"ids"`
			Places []int `json:"places"`
		} `json:"places"`
	}
	if err := json.Unmarshal(browserOutput(t, catalogue.Write(fixture), selectionsFile(ownedIDs())), &r); err != nil {
		t.Fatalf("parsing the browser build's result: %v", err)
	}
	if r.Early != "catalogue not loaded" {
		t.Errorf("places before the catalogue: %q", r.Early)
	}
	want := map[int]int{}
	for _, it := range fixture {
		want[int(it.ID)] = int(it.Position)
	}
	got := map[int]int{}
	for i, id := range r.Places.IDs {
		if i < len(r.Places.Places) {
			got[id] = r.Places.Places[i]
		}
	}
	if len(r.Places.IDs) != len(r.Places.Places) || !maps.Equal(got, want) {
		t.Errorf("places %v for ids %v, want %v", r.Places.Places, r.Places.IDs, want)
	}
}

func TestOwnedPlaces(t *testing.T) {
	ids := ownedIDs()
	var r placesRun
	if err := json.Unmarshal(browserOutput(t, catalogue.Write(fixture), selectionsFile(ids)), &r); err != nil {
		t.Fatalf("parsing the browser build's result: %v", err)
	}

	everything := make([]int, 0, len(fixture))
	for _, it := range fixture {
		everything = append(everything, int(it.ID))
	}
	owned, all := placesOf(ids), placesOf(everything)
	if slices.Equal(owned, all) {
		t.Fatalf("every place the catalogue fills is also in the wardrobe, so this cannot tell the player's places from the catalogue's")
	}
	wears := func(o placesOutfit, place int) bool {
		return slices.ContainsFunc(o.Items, func(it placedItem) bool { return it.Pos == place })
	}
	for _, id := range []int32{50001, 170005, 170006, 170008} {
		if place := placeOf(id); wears(r.Owned, place) {
			t.Errorf("the owned outfit now wears something at place %d, so nothing checks an owned place it leaves empty", place)
		}
	}
	if left := placeOf(170008); !wears(r.All, left) {
		t.Errorf("the best possible outfit no longer wears a left hand, so place %d is no longer a row that must read \"not worn\"", left)
	}
	if right := placeOf(rightHand); !wears(r.All, right) || slices.Contains(owned, right) {
		t.Errorf("place %d is no longer one the best possible outfit fills and the player owns nothing for", right)
	}

	for _, run := range []struct {
		name string
		got  placesOutfit
		want []int
	}{{"owned", r.Owned, owned}, {"all", r.All, all}} {
		if !slices.Equal(run.got.OwnedPlaces, run.want) {
			t.Errorf("%s: ownedPlaces = %v, want %v", run.name, run.got.OwnedPlaces, run.want)
			continue
		}
		for _, it := range run.got.Items {
			if !slices.Contains(run.got.OwnedPlaces, it.Pos) {
				t.Errorf("%s: the outfit wears an item at place %d, which ownedPlaces leaves out", run.name, it.Pos)
			}
		}
	}
}

func TestPrecomputedIdealMatchesBrowser(t *testing.T) {
	packed := catalogue.Write(fixture)
	var r struct {
		All   outfit `json:"all"`
		Ideal struct {
			Score int        `json:"score"`
			Items []idealRow `json:"items"`
		} `json:"ideal"`
		Placed []struct {
			All placedOutfit `json:"all"`
		} `json:"placed"`
	}
	if err := json.Unmarshal(browserOutput(t, packed, selectionsFile(ownedIDs())), &r); err != nil {
		t.Fatalf("parsing the browser build's result: %v", err)
	}
	if r.Ideal.Score == r.All.Score {
		t.Fatalf("the skill-free best possible scores %d, the same as with skills, so this cannot tell whether the driver passed no skills", r.All.Score)
	}

	c, err := catalogue.Read(packed)
	if err != nil {
		t.Fatalf("reading the packed catalogue: %v", err)
	}
	positions, posOf, placeOf := optimizer.FromCatalogue(c, nil)
	want := ideal.Of(positions, posOf, placeOf, scoring.Stage{Attrs: stageAttrs, Weights: stageWeights, Tags: stageTags}, nil)

	got := make([][2]int, 0, len(r.Ideal.Items))
	for _, it := range r.Ideal.Items {
		got = append(got, [2]int{it.ID, it.Pos})
	}
	if r.Ideal.Score != want.Score || !slices.Equal(got, want.Items) {
		t.Errorf("the browser's best possible outfit without skills and the precomputed one disagree.\n--- wasm ---\nscore=%d items=%v\n--- core/ideal ---\nscore=%d items=%v",
			r.Ideal.Score, got, want.Score, want.Items)
	}

	if len(r.Placed) != len(skillRuns) {
		t.Fatalf("the driver ran %d skill placements, want %d", len(r.Placed), len(skillRuns))
	}
	auto := r.Placed[slices.IndexFunc(skillRuns, func(run skillRun) bool { return run.name == "auto" })].All
	if auto.Skills == nil {
		t.Fatal("the browser's auto result names no skills")
	}
	gotAuto := make([][2]int, 0, len(auto.Items))
	for _, it := range auto.Items {
		gotAuto = append(gotAuto, [2]int{it.ID, it.Pos})
	}
	if placed := (scoring.Placement{CharmSmile: auto.Skills.CharmSmile, Smile: auto.Skills.Smile}); placed != want.Auto.Placement ||
		auto.Score != want.Auto.Score || !slices.Equal(gotAuto, want.Auto.Items) {
		t.Errorf("the browser's best possible outfit with auto skills and the precomputed one disagree.\n--- wasm ---\n%+v score=%d items=%v\n--- core/ideal ---\n%+v score=%d items=%v",
			placed, auto.Score, gotAuto, want.Auto.Placement, want.Auto.Score, want.Auto.Items)
	}
	if want.Auto.Score <= want.Score {
		t.Errorf("auto skills score %d, no more than %d without, so the check above cannot tell them apart", want.Auto.Score, want.Score)
	}
}

type skillRun struct {
	name    string
	request any
	levels  scoring.Levels
	native  func(positions []optimizer.Position, st scoring.Stage) (optimizer.Result, *scoring.Placement)
}

func named(p scoring.Placement, l scoring.Levels) func([]optimizer.Position, scoring.Stage) (optimizer.Result, *scoring.Placement) {
	return func(positions []optimizer.Position, st scoring.Stage) (optimizer.Result, *scoring.Placement) {
		return optimizer.Best(positions, st, p.SkillsAt(l)), &p
	}
}

func autoAt(l scoring.Levels) func([]optimizer.Position, scoring.Stage) (optimizer.Result, *scoring.Placement) {
	return func(positions []optimizer.Position, st scoring.Stage) (optimizer.Result, *scoring.Placement) {
		r, p := optimizer.BestPlacedAt(positions, st, l)
		if l.Smile == 0 {
			p.Smile = -1
		}
		return r, &p
	}
}

func untaken(positions []optimizer.Position, st scoring.Stage) (optimizer.Result, *scoring.Placement) {
	return optimizer.Best(positions, st, nil), nil
}

func levels(charming, smile int) map[string]int {
	return map[string]int{"charming": charming, "smile": smile}
}

var maxLevels = scoring.MaxLevels

var skillRuns = []skillRun{
	{"two attributes", map[string]int{"charmSmile": scoring.Cool, "smile": scoring.Simple}, maxLevels,
		named(scoring.Placement{CharmSmile: scoring.Cool, Smile: scoring.Simple}, maxLevels)},
	{"one attribute named twice", map[string]int{"charmSmile": scoring.Lively, "smile": scoring.Lively}, maxLevels,
		named(scoring.Placement{CharmSmile: scoring.Lively, Smile: -1}, maxLevels)},
	{"auto", map[string]bool{"auto": true}, maxLevels, autoAt(maxLevels)},
	{"auto at max levels named", map[string]any{"auto": true, "levels": levels(9, 9)}, maxLevels, autoAt(maxLevels)},
	{"auto with Charming locked", map[string]any{"auto": true, "levels": levels(0, 6)},
		scoring.Levels{Charming: 0, Smile: 6}, autoAt(scoring.Levels{Charming: 0, Smile: 6})},
	{"auto without Smile", map[string]any{"auto": true, "levels": levels(9, 0)},
		scoring.Levels{Charming: 9, Smile: 0}, autoAt(scoring.Levels{Charming: 9, Smile: 0})},
	{"auto with only Smile given", map[string]any{"auto": true, "levels": map[string]int{"smile": 3}},
		scoring.Levels{Charming: 9, Smile: 3}, autoAt(scoring.Levels{Charming: 9, Smile: 3})},
	{"two attributes at low levels", map[string]any{"charmSmile": scoring.Cool, "smile": scoring.Simple, "levels": levels(4, 2)},
		scoring.Levels{Charming: 4, Smile: 2}, named(scoring.Placement{CharmSmile: scoring.Cool, Smile: scoring.Simple}, scoring.Levels{Charming: 4, Smile: 2})},
	{"two attributes without Smile", map[string]any{"charmSmile": scoring.Cool, "smile": scoring.Simple, "levels": levels(7, 0)},
		scoring.Levels{Charming: 7, Smile: 0}, named(scoring.Placement{CharmSmile: scoring.Cool, Smile: -1}, scoring.Levels{Charming: 7, Smile: 0})},
	{"nothing taken", map[string]any{"auto": true, "levels": levels(0, 0)}, scoring.Levels{}, untaken},
}

var badLevels = []any{
	map[string]any{"auto": true, "levels": levels(10, 9)},
	map[string]any{"auto": true, "levels": levels(9, -1)},
	map[string]any{"auto": true, "levels": map[string]any{"smile": 2.5}},
	map[string]any{"auto": true, "levels": map[string]any{"smile": "3"}},
	map[string]any{"auto": true, "levels": 5},
	map[string]any{"charmSmile": scoring.Cool, "smile": scoring.Simple, "levels": levels(99, 1)},
}

func skillRequests() []any {
	requests := make([]any, len(skillRuns))
	for i, run := range skillRuns {
		requests[i] = run.request
	}
	return requests
}

type levelsEcho struct {
	Charming int `json:"charming"`
	Smile    int `json:"smile"`
}

type skillsEcho struct {
	CharmSmile int         `json:"charmSmile"`
	Smile      int         `json:"smile"`
	Levels     *levelsEcho `json:"levels,omitempty"`
}

type altRow struct {
	ID    int `json:"id"`
	Delta int `json:"delta"`
}

type pricedItem struct {
	ID       int      `json:"id"`
	Pos      int      `json:"pos"`
	Alts     []altRow `json:"alts"`
	MoreAlts bool     `json:"moreAlts"`
}

type placedOutfit struct {
	Score  int          `json:"score"`
	Skills *skillsEcho  `json:"skills"`
	Items  []pricedItem `json:"items"`
}

type placedRuns struct {
	Placed []struct {
		Owned placedOutfit `json:"owned"`
		All   placedOutfit `json:"all"`
	} `json:"placed"`
}

func TestSkillsParity(t *testing.T) {
	packed := catalogue.Write(fixture)
	var r placedRuns
	if err := json.Unmarshal(browserOutput(t, packed, selectionsFile(ownedIDs())), &r); err != nil {
		t.Fatalf("parsing the browser build's result: %v", err)
	}
	if len(r.Placed) != len(skillRuns) {
		t.Fatalf("the driver ran %d skill placements, want %d", len(r.Placed), len(skillRuns))
	}
	c, err := catalogue.Read(packed)
	if err != nil {
		t.Fatalf("reading the packed catalogue: %v", err)
	}
	st := scoring.Stage{Attrs: stageAttrs, Weights: stageWeights, Tags: stageTags}

	unpriced := false
	scores := map[string]int{}
	for i, run := range skillRuns {
		for _, scope := range []struct {
			name      string
			ownedOnly bool
			got       placedOutfit
		}{{"owned", true, r.Placed[i].Owned}, {"all", false, r.Placed[i].All}} {
			label := run.name + " " + scope.name
			positions := positionsFrom(c, ownedIDs(), scope.ownedOnly)
			res, placement := run.native(positions, st)
			var sk scoring.Skills
			if placement != nil {
				sk = placement.SkillsAt(run.levels)
			}
			want := pricedOutfit(positions, st, res, sk, nil)
			if placement != nil {
				want.Skills = &skillsEcho{CharmSmile: placement.CharmSmile, Smile: placement.Smile}
				if run.levels != scoring.MaxLevels {
					want.Skills.Levels = &levelsEcho{run.levels.Charming, run.levels.Smile}
				}
			}
			if got, _ := json.Marshal(scope.got); string(got) != mustJSON(t, want) {
				t.Errorf("%s: the browser build and the native build disagree.\n--- wasm ---\n%s\n--- native ---\n%s",
					label, got, mustJSON(t, want))
			}
			if placement != nil && mustJSON(t, pricedOutfit(positions, st, res, nil, nil)) != mustJSON(t, pricedOutfit(positions, st, res, sk, nil)) {
				unpriced = true
			}
			scores[label] = res.Score
		}
	}
	if !unpriced {
		t.Error("pricing the alternatives without skills gives the same deltas, so nothing checks they are priced with the skills used")
	}

	all := positionsFrom(c, ownedIDs(), false)
	if smiled := optimizer.Best(all, st, scoring.Skills{scoring.Lively: scoring.SmileOnly}).Score; smiled == scores["one attribute named twice all"] {
		t.Errorf("Smile alone on Lively also scores %d, so nothing checks that naming it twice keeps Charming", smiled)
	}
	if scores["auto all"] == scores["two attributes all"] {
		t.Errorf("auto and the named pair both score %d, so nothing tells them apart", scores["auto all"])
	}
	if unskilled := optimizer.Best(all, st, nil).Score; scores["auto all"] <= unskilled {
		t.Errorf("auto scores %d, no more than %d without skills", scores["auto all"], unskilled)
	}
	seen := map[int]string{}
	for _, name := range []string{"auto", "auto with Charming locked", "auto with only Smile given", "nothing taken"} {
		if other, ok := seen[scores[name+" all"]]; ok {
			t.Errorf("%q and %q both score %d, so nothing checks the levels reach the engine", name, other, scores[name+" all"])
		}
		seen[scores[name+" all"]] = name
	}
	if scores["two attributes at low levels all"] == scores["two attributes all"] || scores["two attributes without Smile all"] == scores["two attributes at low levels all"] {
		t.Error("the named pair scores the same at different levels")
	}
}

func TestSkillLevelsAreChecked(t *testing.T) {
	var r struct {
		Rejected []string `json:"rejected"`
	}
	if err := json.Unmarshal(browserOutput(t, catalogue.Write(fixture), selectionsFile(ownedIDs())), &r); err != nil {
		t.Fatalf("parsing the browser build's result: %v", err)
	}
	if len(r.Rejected) != len(badLevels) {
		t.Fatalf("the driver tried %d bad levels, want %d", len(r.Rejected), len(badLevels))
	}
	for i, got := range r.Rejected {
		if got != "skill levels run from 0 to 9" {
			t.Errorf("levels %v: got %q", badLevels[i], got)
		}
	}
}

func pricedOutfit(positions []optimizer.Position, st scoring.Stage, res optimizer.Result, sk scoring.Skills, met [][]int) placedOutfit {
	worn := 0
	for _, it := range res.Items {
		if it.Slot == scoring.Accessory {
			worn++
		}
	}
	ratio := scoring.AccessoryPenalty(worn)
	rate := func(it scoring.Item) float64 {
		scaled, fixed := scoring.Contribution(it, st, sk)
		if it.Slot == scoring.Accessory {
			scaled *= ratio
		}
		return scaled + fixed
	}

	out := placedOutfit{Score: res.Score, Items: []pricedItem{}}
	for _, it := range res.Items {
		row := pricedItem{ID: it.ID, Pos: placeOf(int32(it.ID)), Alts: []altRow{}}
		for _, p := range positions {
			if !slices.ContainsFunc(p.Items, func(o scoring.Item) bool { return o.ID == it.ID }) {
				continue
			}
			for _, alt := range optimizer.Ranked(p.Items, st, sk) {
				if alt.ID == it.ID || breaksRule(res.Items, it, alt, met) {
					continue
				}
				if len(row.Alts) == 5 {
					row.MoreAlts = true
					break
				}
				row.Alts = append(row.Alts, altRow{alt.ID, int(rate(alt) - rate(it))})
			}
			break
		}
		out.Items = append(out.Items, row)
	}
	return out
}

func breaksRule(worn []scoring.Item, out, in scoring.Item, met [][]int) bool {
	for _, set := range met {
		if !slices.Contains(set, out.ID) || slices.Contains(set, in.ID) {
			continue
		}
		kept := false
		for _, w := range worn {
			if w.ID != out.ID && slices.Contains(set, w.ID) {
				kept = true
			}
		}
		if !kept {
			return true
		}
	}
	return false
}

type requiredRun struct {
	name    string
	skills  any
	require [][]int
}

var requiredRuns = []requiredRun{
	{"weak dress", nil, [][]int{{weakDress}}},
	{"accessory at a loss", map[string]int{"charmSmile": scoring.Lively}, [][]int{{170005}}},
	{"one of two places", map[string]bool{"auto": true}, [][]int{{170006, 170009}}},
	{"missing hair", nil, [][]int{{betterHair}, {40001}}},
}

func requiredRequests() []any {
	requests := make([]any, len(requiredRuns))
	for i, run := range requiredRuns {
		requests[i] = map[string]any{"skills": run.skills, "require": run.require}
	}
	return requests
}

type requiredOutfit struct {
	placedOutfit
	Missing [][]int `json:"missing"`
}

func nativeRequired(positions []optimizer.Position, st scoring.Stage, run requiredRun) (optimizer.Result, requiredOutfit) {
	posOf := map[int]int{}
	for i, p := range positions {
		for _, it := range p.Items {
			posOf[it.ID] = i
		}
	}
	space := optimizer.Require(positions, posOf, run.require)
	var res optimizer.Result
	var placement *scoring.Placement
	sk := scoring.Skills{}
	switch skills := run.skills.(type) {
	case map[string]bool:
		var p scoring.Placement
		res, p = space.BestPlaced(st)
		placement, sk = &p, p.Skills()
	case map[string]int:
		p := scoring.Placement{CharmSmile: skills["charmSmile"], Smile: -1}
		placement, sk = &p, p.Skills()
		res = space.Best(st, sk)
	default:
		res = space.Best(st, sk)
	}
	var met, missing [][]int
	for _, set := range run.require {
		if slices.ContainsFunc(res.Items, func(it scoring.Item) bool { return slices.Contains(set, it.ID) }) {
			met = append(met, set)
		} else {
			missing = append(missing, set)
		}
	}
	out := requiredOutfit{placedOutfit: pricedOutfit(positions, st, res, sk, met), Missing: missing}
	if placement != nil {
		out.Skills = &skillsEcho{CharmSmile: placement.CharmSmile, Smile: placement.Smile}
	}
	return res, out
}

func TestRequiredParity(t *testing.T) {
	packed := catalogue.Write(fixture)
	var r struct {
		Required []struct {
			Owned requiredOutfit `json:"owned"`
			All   requiredOutfit `json:"all"`
		} `json:"required"`
	}
	if err := json.Unmarshal(browserOutput(t, packed, selectionsFile(ownedIDs())), &r); err != nil {
		t.Fatalf("parsing the browser build's result: %v", err)
	}
	if len(r.Required) != len(requiredRuns) {
		t.Fatalf("the driver ran %d requirement sets, want %d", len(r.Required), len(requiredRuns))
	}
	c, err := catalogue.Read(packed)
	if err != nil {
		t.Fatalf("reading the packed catalogue: %v", err)
	}
	st := scoring.Stage{Attrs: stageAttrs, Weights: stageWeights, Tags: stageTags}
	wears := func(res optimizer.Result, id int) bool {
		return slices.ContainsFunc(res.Items, func(it scoring.Item) bool { return it.ID == id })
	}

	results := map[string]optimizer.Result{}
	for i, run := range requiredRuns {
		for _, scope := range []struct {
			name      string
			ownedOnly bool
			got       requiredOutfit
		}{{"owned", true, r.Required[i].Owned}, {"all", false, r.Required[i].All}} {
			label := run.name + " " + scope.name
			res, want := nativeRequired(positionsFrom(c, ownedIDs(), scope.ownedOnly), st, run)
			if got := mustJSON(t, scope.got); got != mustJSON(t, want) {
				t.Errorf("%s: the browser build and the native build disagree.\n--- wasm ---\n%s\n--- native ---\n%s",
					label, got, mustJSON(t, want))
			}
			results[label] = res
		}
	}

	all := positionsFrom(c, ownedIDs(), false)
	for _, scope := range []string{"owned", "all"} {
		res := results["weak dress "+scope]
		if !wears(res, weakDress) || wears(res, 40001) || wears(res, 50001) {
			t.Errorf("weak dress %s: wears %v, want the required dress and no separates", scope, worn(res))
		}
	}
	if free := optimizer.Best(optimizer.Exclude(all, 20001), st, nil); !wears(free, 40001) || !wears(free, 50001) {
		t.Errorf("without the strong dress the best outfit wears %v, so the separates never had to make way for the weak dress", worn(free))
	}
	for _, row := range r.Required[0].All.Items {
		if row.ID == weakDress && len(row.Alts) != 0 {
			t.Errorf("the required dress offers %v as alternatives, which would fail the stage", row.Alts)
		}
	}

	sk := (&scoring.Placement{CharmSmile: scoring.Lively, Smile: -1}).Skills()
	for _, scope := range []struct {
		name      string
		ownedOnly bool
	}{{"owned", true}, {"all", false}} {
		res := results["accessory at a loss "+scope.name]
		free := optimizer.Best(positionsFrom(c, ownedIDs(), scope.ownedOnly), st, sk)
		if !wears(res, 170005) || wears(free, 170005) || res.Score >= free.Score {
			t.Errorf("accessory at a loss %s: wears %v for %d, and without the rule %v for %d; want the required accessory to cost points",
				scope.name, worn(res), res.Score, worn(free), free.Score)
		}
	}

	twoPlaces := results["one of two places all"]
	if !wears(twoPlaces, 170006) && !wears(twoPlaces, 170009) {
		t.Errorf("one of two places: wears %v, neither 170006 nor 170009", worn(twoPlaces))
	}
	if wears(twoPlaces, 170009) && (wears(twoPlaces, rightHand) || wears(twoPlaces, 170008)) {
		t.Errorf("one of two places: wears %v, a both-hands item with a one-hand item", worn(twoPlaces))
	}
	if free, _ := optimizer.BestPlaced(all, st); wears(free, 170006) || wears(free, 170009) {
		t.Errorf("without the rule the best outfit already wears %v, so the rule proves nothing", worn(free))
	}

	if got := r.Required[3].Owned.Missing; !slices.EqualFunc(got, [][]int{{betterHair}}, slices.Equal) {
		t.Errorf("missing hair owned: reported missing %v, want [[%d]]", got, betterHair)
	}
	if got := r.Required[3].All.Missing; got != nil {
		t.Errorf("missing hair all: reported missing %v, and the catalogue has everything", got)
	}
	if res := results["missing hair owned"]; !wears(res, 40001) || wears(res, 20001) {
		t.Errorf("missing hair owned: wears %v, want the required top in place of the dress despite the missing hair", worn(res))
	}
}

func worn(res optimizer.Result) []int {
	ids := make([]int, len(res.Items))
	for i, it := range res.Items {
		ids[i] = it.ID
	}
	return ids
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

type idealRow struct {
	ID  int `json:"id"`
	Pos int `json:"pos"`
}

type placesRun struct {
	Owned placesOutfit `json:"owned"`
	All   placesOutfit `json:"all"`
}

type placesOutfit struct {
	Items       []placedItem `json:"items"`
	OwnedPlaces []int        `json:"ownedPlaces"`
}

type placedItem struct {
	Pos int `json:"pos"`
}

func placesOf(ids []int) []int {
	var places []int
	for _, it := range fixture {
		if slices.Contains(ids, int(it.ID)) && !slices.Contains(places, int(it.Position)) {
			places = append(places, int(it.Position))
		}
	}
	slices.Sort(places)
	return places
}

func placeOf(id int32) int {
	for _, it := range fixture {
		if it.ID == id {
			return int(it.Position)
		}
	}
	panic(fmt.Sprintf("no item %d in the fixture", id))
}

type chosen struct {
	ID   int `json:"id"`
	Slot int `json:"slot"`
}

type outfit struct {
	Score int      `json:"score"`
	Items []chosen `json:"items"`
}

type decoded struct {
	Items      int `json:"items"`
	Unresolved int `json:"unresolved"`
	Known      int `json:"known"`
}

type engineRun struct {
	Decoded decoded `json:"decoded"`
	Owned   outfit  `json:"owned"`
	All     outfit  `json:"all"`
}

const goldenHeader = `# Both builds of the engine must render exactly this for the fixture in
# parity_test.go. Regenerate with: go test ./wasm/parity -update
`

func render(r engineRun) string {
	var b strings.Builder
	b.WriteString(goldenHeader)
	fmt.Fprintf(&b, "\nwardrobe items=%d unresolved=%d known=%d\n",
		r.Decoded.Items, r.Decoded.Unresolved, r.Decoded.Known)
	for _, section := range []struct {
		name string
		o    outfit
	}{{"owned", r.Owned}, {"all", r.All}} {
		fmt.Fprintf(&b, "\n%s score=%d\n", section.name, section.o.Score)
		for _, it := range section.o.Items {
			fmt.Fprintf(&b, "  item=%d slot=%d\n", it.ID, it.Slot)
		}
	}
	return b.String()
}

func nativeRun(t *testing.T, packed, file []byte) engineRun {
	t.Helper()
	c, err := catalogue.Read(packed)
	if err != nil {
		t.Fatalf("reading the packed catalogue: %v", err)
	}
	w, err := wardrobe.DecodeSelections(file)
	if err != nil {
		t.Fatalf("reading the selections file: %v", err)
	}
	st := scoring.Stage{Attrs: stageAttrs, Weights: stageWeights, Tags: stageTags}
	return engineRun{
		Decoded: decoded{Items: len(w.Items), Unresolved: w.Unresolved, Known: known(c, w.Items)},
		Owned:   nativeBest(c, w.Items, true, st),
		All:     nativeBest(c, w.Items, false, st),
	}
}

func nativeBest(c *catalogue.Catalogue, owned []int, ownedOnly bool, st scoring.Stage) outfit {
	best := optimizer.Best(positionsFrom(c, owned, ownedOnly), st, stageSkills)
	out := outfit{Score: best.Score}
	for _, it := range best.Items {
		out.Items = append(out.Items, chosen{ID: it.ID, Slot: int(it.Slot)})
	}
	return out
}

func positionsFrom(c *catalogue.Catalogue, owned []int, ownedOnly bool) []optimizer.Position {
	have := make(map[int32]bool, len(owned))
	for _, id := range owned {
		have[int32(id)] = true
	}
	byPosition := map[uint16][]scoring.Item{}
	groupOf := map[uint16]uint8{}
	var order []uint16
	for i, id := range c.IDs {
		if ownedOnly && !have[id] {
			continue
		}
		it := scoring.Item{
			ID: int(id), Slot: scoring.Slot(c.Slots[i]),
			FlatBonus: int(c.FlatBonus[i]),
		}
		for p := range 5 {
			it.Attrs[p] = c.Attrs[i*5+p]
			it.Stats[p] = int(c.Stats[i*5+p])
		}
		for _, tag := range c.Tags[c.TagOffset[i]:c.TagOffset[i+1]] {
			it.Tags = append(it.Tags, int(tag))
		}
		pos := c.Positions[i]
		if _, seen := byPosition[pos]; !seen {
			order = append(order, pos)
			groupOf[pos] = c.Groups[i]
		}
		byPosition[pos] = append(byPosition[pos], it)
	}
	positions := make([]optimizer.Position, 0, len(order))
	for _, pos := range order {
		group, whole := catalogue.SplitGroup(groupOf[pos])
		positions = append(positions, optimizer.Position{
			Items: byPosition[pos], Group: group, Exclusive: whole,
		})
	}
	return positions
}

func known(c *catalogue.Catalogue, owned []int) int {
	have := make(map[int32]bool, len(owned))
	for _, id := range owned {
		have[int32(id)] = true
	}
	n := 0
	for _, id := range c.IDs {
		if have[id] {
			n++
		}
	}
	return n
}

func browserRun(t *testing.T, packed, file []byte) engineRun {
	t.Helper()
	var r engineRun
	if err := json.Unmarshal(browserOutput(t, packed, file), &r); err != nil {
		t.Fatalf("parsing the browser build's result: %v", err)
	}
	return r
}

func browserOutput(t *testing.T, packed, file []byte) []byte {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not on PATH, so the browser build cannot be run here")
	}
	shim := wasmExec(t)

	dir := t.TempDir()
	module := filepath.Join(dir, "nikkibase.wasm")
	build := exec.Command("go", "build", "-o", module, "github.com/danielradosa/nikkibase/wasm")
	build.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("the browser build does not compile (%v):\n%s", err, out)
	}
	watchBrowserBuild(t)

	in := filepath.Join(dir, "input.json")
	encoded, err := json.Marshal(map[string]any{
		"catalogue": packed,
		"wardrobe":  string(file),
		"weights":   stageWeights,
		"attrs":     stageAttrs,
		"tags":      stageTags,
		"skills":    map[string]int{"charmSmile": scoring.Lively},
		"placed":    skillRequests(),
		"bad":       badLevels,
		"required":  requiredRequests(),
		"worth":     worthRequests(packed),
	})
	if err != nil {
		t.Fatalf("encoding the driver's input: %v", err)
	}
	if err := os.WriteFile(in, encoded, 0o644); err != nil {
		t.Fatalf("writing the driver's input: %v", err)
	}

	out := filepath.Join(dir, "output.json")
	drive := exec.Command(node, "--no-wasm-dynamic-tiering", "testdata/driver.js", shim, module, in, out)
	if log, err := drive.CombinedOutput(); err != nil {
		t.Fatalf("running the browser build under node (%v):\n%s", err, log)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("the driver wrote no result: %v", err)
	}
	return raw
}

func TestBrowserBuildLeavesOutFmtAndOS(t *testing.T) {
	list := exec.Command("go", "list", "-deps", "github.com/danielradosa/nikkibase/wasm", "github.com/danielradosa/nikkibase/core/worth")
	list.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	out, err := list.Output()
	if err != nil {
		t.Fatalf("listing the browser build's packages: %v", err)
	}
	for pkg := range strings.Lines(string(out)) {
		if pkg = strings.TrimSpace(pkg); pkg == "fmt" || pkg == "os" {
			t.Errorf("the browser build links %s, which adds 50 KB or more to the brotli download; build errors in core/ with errors.New and strconv instead", pkg)
		}
	}
}

func watchBrowserBuild(t *testing.T) {
	t.Helper()
	list := exec.Command("go", "list", "-deps",
		"-f", "{{if .Module}}{{if .Module.Main}}{{.Dir}}{{end}}{{end}}",
		"github.com/danielradosa/nikkibase/wasm")
	list.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	out, err := list.Output()
	if err != nil {
		t.Fatalf("listing the browser build's packages: %v", err)
	}
	for dir := range strings.Lines(string(out)) {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		if _, err := os.ReadDir(dir); err != nil {
			t.Fatalf("reading %s: %v", dir, err)
		}
	}
	if _, err := os.Stat("testdata/driver.js"); err != nil {
		t.Fatalf("the driver is missing: %v", err)
	}
}

func wasmExec(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		t.Skipf("cannot locate GOROOT: %v", err)
	}
	root := strings.TrimSpace(string(out))
	for _, rel := range []string{"lib/wasm", "misc/wasm"} {
		path := filepath.Join(root, rel, "wasm_exec.js")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	t.Skipf("no wasm_exec.js under %s", root)
	return ""
}

func selectionsFile(ids []int) []byte {
	var b strings.Builder
	b.WriteString("@SELVER1=")
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%d_%d", i%1000, id)
	}
	return []byte(b.String())
}

func ownedIDs() []int {
	ids := make([]int, 0, len(fixture))
	for _, it := range fixture {
		if !slices.Contains(unowned, it.ID) {
			ids = append(ids, int(it.ID))
		}
	}
	return ids
}
