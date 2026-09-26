package parity

import (
	"encoding/json"
	"maps"
	"slices"
	"strconv"
	"testing"

	"github.com/danielradosa/nikkibase/core/catalogue"
	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
	"github.com/danielradosa/nikkibase/core/worth"
)

var worthOwned = []int{10001, weakDress, 30001, 60001, 70001, 170001, 170003, 170005, 170008, 880001}

type worthStage struct {
	key, mode string
	weights   [5]float64
	attrs     [5]int8
	tags      map[int]int
	require   [][]int
}

var worthStages = []worthStage{
	{"Story/1-1", "Story", stageWeights, stageAttrs, stageTags, nil},
	{"Story/1-1#maiden", "Story", [5]float64{4, 1, 0, 1, 4}, stageAttrs, map[int]int{tagOne: 120}, nil},
	{"Commission/2-3", "Commission", [5]float64{1, 1, 1, 1, 1}, [5]int8{scoring.Gorgeous, scoring.Lively, scoring.Cute, scoring.Sexy, scoring.Warm}, map[int]int{tagTwo: 300}, nil},
	{"Arena/Night \"Gala\"", "Arena", [5]float64{0, 5, 0, 2, 3}, stageAttrs, nil, [][]int{{weakDress}}},
	{"Story/2-1", "Story", stageWeights, stageAttrs, stageTags, [][]int{{betterHair}, {40001, weakDress}}},
	{"Co-op/Duo", "Co-op", [5]float64{3, 3, 0, 0, 4}, stageAttrs, nil, [][]int{{rightHand, 170008}}},
}

var worthSuits = []worth.Suit{
	{Key: "Separates", Items: []int{40001, 50001, 99999}},
	{Key: "Night \"Gala\" set", Items: []int{20001, 170002, 170004, 10001}},
	{Key: "Hands", Items: []int{rightHand, 170009, 170008}},
	{Key: "Better hair", Items: []int{betterHair, 80001, 170006}},
	{Key: "Owned", Items: []int{10001, 30001}},
}

func suitRequests() []map[string]any {
	out := make([]map[string]any, len(worthSuits))
	for i, su := range worthSuits {
		out[i] = map[string]any{"key": su.Key, "items": su.Items}
	}
	return out
}

type worthRunCase struct {
	name     string
	request  any
	settings worth.Settings
}

var worthRuns = []worthRunCase{
	{"no skills", nil, worth.Settings{}},
	{"auto", map[string]bool{"auto": true}, worth.Settings{Skills: true, Levels: scoring.MaxLevels}},
	{"auto at 4/2", map[string]any{"auto": true, "levels": levels(4, 2)}, worth.Settings{Skills: true, Levels: scoring.Levels{Charming: 4, Smile: 2}}},
	{"auto off", map[string]bool{"auto": false}, worth.Settings{}},
}

type worthFilter struct {
	worth.Filter
	places []int
}

var worthFilters = []worthFilter{
	{},
	{Filter: worth.Filter{Modes: []string{"Story"}, Skip: []string{"Story/1-1"}}},
	{Filter: worth.Filter{Slots: []scoring.Slot{scoring.Accessory}}},
	{Filter: worth.Filter{Modes: []string{"Commission", "Arena", "Co-op"}, Slots: []scoring.Slot{scoring.Top, scoring.Bottom}}},
	{places: []int{9, 11}},
	{Filter: worth.Filter{Modes: []string{"Story", "Arena"}}, places: []int{3}},
	{Filter: worth.Filter{Slots: []scoring.Slot{scoring.Accessory}}, places: []int{15, 17, 3}},
	{places: []int{99}},
	{Filter: worth.Filter{Suits: true}},
	{Filter: worth.Filter{Suits: true, Modes: []string{"Story", "Co-op"}, Skip: []string{"Story/1-1#maiden"}}},
	{Filter: worth.Filter{Suits: true, Slots: []scoring.Slot{scoring.Accessory}}},
	{Filter: worth.Filter{Suits: true}, places: []int{4}},
}

func (f worthFilter) native(posOf map[int]int) worth.Filter {
	out := f.Filter
	for _, place := range f.places {
		at := -1
		for _, it := range fixture {
			if int(it.Position) == place {
				at = posOf[int(it.ID)]
				break
			}
		}
		out.Positions = append(out.Positions, at)
	}
	return out
}

const (
	worthChunk = 2
	worthLimit = 8
)

type badStart struct {
	versions any
	settings any
	suits    any
	want     string
}

func worthBadStarts(versions []map[string]any) []badStart {
	with := func(field string, value any) []map[string]any {
		v := maps.Clone(versions[0])
		v[field] = value
		return []map[string]any{v}
	}
	return []badStart{
		{"nope", nil, nil, "the stages must be a list"},
		{[]any{5}, nil, nil, "stage 1: not an object"},
		{with("weights", []int{1, 2, 3}), nil, nil, "stage 1: weights and attrs must be five numbers each"},
		{with("attrs", []int{0, 2, 4, 6, 12}), nil, nil, "stage 1: attrs must be attribute codes from 0 to 9"},
		{with("attrs", []any{0, 2, 4, 6, "8"}), nil, nil, "stage 1: attrs must be attribute codes from 0 to 9"},
		{with("ideal", 0), nil, nil, "stage 1: ideal must be the best possible score, a whole number above 0"},
		{with("ideal", 1.5), nil, nil, "stage 1: ideal must be the best possible score, a whole number above 0"},
		{with("tags", map[string]any{"x": 5}), nil, nil, "stage 1: tags must map tag numbers to points"},
		{with("tags", map[string]any{"7": 1.5}), nil, nil, "stage 1: tags must map tag numbers to points"},
		{with("require", []any{[]any{1, "2"}}), nil, nil, "stage 1: require must be a list of item lists"},
		{with("require", []any{[]any{}}), nil, nil, "stage 1: require must be a list of item lists"},
		{with("key", 5), nil, nil, "stage 1: the key and the mode must be text"},
		{versions, 5, nil, "skills must be null or {auto: true}"},
		{versions, map[string]any{"auto": true, "levels": levels(0, 10)}, nil, "skill levels run from 0 to 9"},
		{versions, map[string]any{"auto": true, "levels": 5}, nil, "skill levels run from 0 to 9"},
		{versions, map[string]any{"auto": true, "levels": map[string]any{"smile": 2.5}}, nil, "skill levels run from 0 to 9"},
		{versions, nil, "nope", "suits must be a list"},
		{versions, nil, []any{5}, "suit 1: not an object"},
		{versions, nil, []any{map[string]any{"key": 5, "items": []int{}}}, "suit 1: the key must be text"},
		{versions, nil, []any{map[string]any{"key": "a", "items": []int{1}}, map[string]any{"key": "b", "items": "1"}}, "suit 2: items must be a list of item numbers"},
		{versions, nil, []any{map[string]any{"key": "a", "items": []any{1.5}}}, "suit 1: items must be a list of item numbers"},
		{versions, nil, []any{map[string]any{"key": "a"}}, "suit 1: items must be a list of item numbers"},
	}
}

var worthBadRuns = []struct {
	offset int
	count  any
	want   string
}{
	{0, -1, "the count must be a whole number"},
	{0, 1.5, "the count must be a whole number"},
	{0, "3", "the count must be a whole number"},
	{1, 2, "no session"},
	{-1, 2, "no session"},
}

var worthBadRanks = []struct {
	offset int
	filter any
	limit  any
	want   string
}{
	{0, "x", 5, "the filter must be an object"},
	{0, map[string]any{"modes": "Story"}, 5, "modes must be a list of mode names"},
	{0, map[string]any{"slots": []int{10}}, 5, "slots must be a list of slot numbers from 0 to 9"},
	{0, map[string]any{"slots": []string{"8"}}, 5, "slots must be a list of slot numbers from 0 to 9"},
	{0, map[string]any{"skip": []int{1}}, 5, "skip must be a list of stage keys"},
	{0, map[string]any{"places": "9"}, 5, "places must be a list of place numbers"},
	{0, map[string]any{"places": []int{-1}}, 5, "places must be a list of place numbers"},
	{0, map[string]any{"places": []any{1.5}}, 5, "places must be a list of place numbers"},
	{0, map[string]any{"places": []int{65536}}, 5, "places must be a list of place numbers"},
	{0, map[string]any{}, -1, "the limit runs from 0 to 1000"},
	{0, map[string]any{}, 1001, "the limit runs from 0 to 1000"},
	{0, map[string]any{}, 2.5, "the limit runs from 0 to 1000"},
	{0, map[string]any{"suits": "yes"}, 5, "suits must be true or false"},
	{0, map[string]any{"suits": 1}, 5, "suits must be true or false"},
	{7, map[string]any{}, 5, "no session"},
}

var worthBigInts = []string{
	"the stages must be a list",
	"stage 1: weights and attrs must be five numbers each",
	"stage 1: the key and the mode must be text",
	"skills must be null or {auto: true}",
	"skill levels run from 0 to 9",
	"no session",
	"the filter must be an object",
	"modes must be a list of mode names",
	"places must be a list of place numbers",
	"the limit runs from 0 to 1000",
	"suits must be a list",
	"suit 1: the key must be text",
	"suit 1: items must be a list of item numbers",
	"suits must be true or false",
}

func worthCatalogue(t *testing.T, packed []byte) ([]optimizer.Position, map[int]int) {
	t.Helper()
	c, err := catalogue.Read(packed)
	if err != nil {
		t.Fatalf("reading the packed catalogue: %v", err)
	}
	positions, posOf, _ := optimizer.FromCatalogue(c, nil)
	return positions, posOf
}

func worthVersions(positions []optimizer.Position, posOf map[int]int) ([]worth.Version, []map[string]any) {
	var native []worth.Version
	var requests []map[string]any
	for _, s := range worthStages {
		st := scoring.Stage{Weights: s.weights, Attrs: s.attrs, Tags: s.tags}
		ideal := optimizer.Require(positions, posOf, s.require).Best(st, nil).Score
		native = append(native, worth.Version{Key: s.key, Mode: s.mode, Stage: st, Require: s.require, Ideal: ideal})
		request := map[string]any{"key": s.key, "mode": s.mode, "weights": s.weights, "attrs": s.attrs, "ideal": ideal}
		if s.tags != nil {
			tags := map[string]int{}
			for id, award := range s.tags {
				tags[strconv.Itoa(id)] = award
			}
			request["tags"] = tags
		}
		if s.require != nil {
			request["require"] = s.require
		}
		requests = append(requests, request)
	}
	return native, requests
}

func filterRequest(f worthFilter) map[string]any {
	out := map[string]any{}
	if f.Modes != nil {
		out["modes"] = f.Modes
	}
	if f.Skip != nil {
		out["skip"] = f.Skip
	}
	if f.Slots != nil {
		slots := make([]int, len(f.Slots))
		for i, slot := range f.Slots {
			slots[i] = int(slot)
		}
		out["slots"] = slots
	}
	if f.places != nil {
		out["places"] = f.places
	}
	if f.Suits {
		out["suits"] = true
	}
	return out
}

func worthRequests(packed []byte) map[string]any {
	c, err := catalogue.Read(packed)
	if err != nil {
		return nil
	}
	positions, posOf, _ := optimizer.FromCatalogue(c, nil)
	_, versions := worthVersions(positions, posOf)
	runs := make([]any, len(worthRuns))
	for i, run := range worthRuns {
		runs[i] = run.request
	}
	filters := make([]any, len(worthFilters))
	for i, f := range worthFilters {
		filters[i] = filterRequest(f)
	}
	var starts, runsBad, ranksBad []any
	for _, b := range worthBadStarts(versions) {
		starts = append(starts, []any{b.versions, b.settings, b.suits})
	}
	for _, b := range worthBadRuns {
		runsBad = append(runsBad, []any{b.offset, b.count})
	}
	for _, b := range worthBadRanks {
		ranksBad = append(ranksBad, []any{b.offset, b.filter, b.limit})
	}
	return map[string]any{
		"wardrobe": string(selectionsFile(worthOwned)),
		"versions": versions,
		"suits":    suitRequests(),
		"runs":     runs,
		"filters":  filters,
		"chunk":    worthChunk,
		"limit":    worthLimit,
		"bad":      map[string]any{"start": starts, "run": runsBad, "rank": ranksBad},
	}
}

type worthExample struct {
	Key    string  `json:"key"`
	Points int     `json:"points"`
	Pct    float64 `json:"pct"`
}

type worthRow struct {
	Suit     string         `json:"suit,omitempty"`
	Items    []int          `json:"items"`
	Places   []int          `json:"places"`
	Worth    float64        `json:"worth"`
	Stages   int            `json:"stages"`
	Best     worthExample   `json:"best"`
	Examples []worthExample `json:"examples"`
}

type worthNeeded struct {
	Key     string  `json:"key"`
	Missing [][]int `json:"missing"`
}

type worthRanking struct {
	Rows   []worthRow    `json:"rows"`
	Needed []worthNeeded `json:"needed"`
}

type worthProgress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

type worthBrowserRun struct {
	Total    int             `json:"total"`
	Session  float64         `json:"session"`
	Progress []worthProgress `json:"progress"`
	Ranks    []worthRanking  `json:"ranks"`
}

type worthBrowser struct {
	Runs     []worthBrowserRun `json:"runs"`
	Rejected []string          `json:"rejected"`
	Kept     string            `json:"kept"`
	Dropped  []string          `json:"dropped"`
}

func rounded(x float64) float64 {
	v, _ := strconv.ParseFloat(strconv.FormatFloat(x, 'f', 3, 64), 64)
	return v
}

func example(ex worth.Example) worthExample {
	return worthExample{ex.Key, ex.Points, rounded(ex.Pct)}
}

func fixturePlace(id int) int {
	for _, it := range fixture {
		if int(it.ID) == id {
			return int(it.Position)
		}
	}
	return -1
}

func nativeRanking(r worth.Ranking) worthRanking {
	out := worthRanking{Rows: []worthRow{}, Needed: []worthNeeded{}}
	for _, row := range r.Rows {
		w := worthRow{Suit: row.Suit, Items: row.Items, Places: []int{}, Worth: rounded(row.Worth), Stages: row.Stages, Best: example(row.Best), Examples: []worthExample{}}
		for _, id := range row.Items {
			w.Places = append(w.Places, fixturePlace(id))
		}
		for _, ex := range row.Examples {
			w.Examples = append(w.Examples, example(ex))
		}
		out.Rows = append(out.Rows, w)
	}
	for _, n := range r.Needed {
		out.Needed = append(out.Needed, worthNeeded{n.Key, n.Missing})
	}
	return out
}

func TestWorthParity(t *testing.T) {
	packed := catalogue.Write(fixture)
	var r struct {
		Worth worthBrowser `json:"worth"`
	}
	if err := json.Unmarshal(browserOutput(t, packed, selectionsFile(ownedIDs())), &r); err != nil {
		t.Fatalf("parsing the browser build's result: %v", err)
	}
	if len(r.Worth.Runs) != len(worthRuns) {
		t.Fatalf("the driver ran %d worth sessions, want %d", len(r.Worth.Runs), len(worthRuns))
	}
	positions, posOf := worthCatalogue(t, packed)
	versions, _ := worthVersions(positions, posOf)
	owns := func(id int32) bool { return slices.Contains(worthOwned, int(id)) }

	rankings := map[string][]worthRanking{}
	for i, run := range worthRuns {
		got := r.Worth.Runs[i]
		if got.Session <= 0 || got.Session != float64(int64(got.Session)) {
			t.Errorf("%s: session %v is not a positive whole number", run.name, got.Session)
		}
		settings := run.settings
		settings.Suits = worthSuits
		s := worth.NewSession(positions, posOf, owns, versions, settings)
		var progress []worthProgress
		for {
			done, total := s.Run(worthChunk)
			progress = append(progress, worthProgress{done, total})
			if done == total {
				break
			}
		}
		if got.Total != len(versions) || !slices.Equal(got.Progress, progress) {
			t.Errorf("%s: the browser ran %d stages as %v, native %d as %v", run.name, got.Total, got.Progress, len(versions), progress)
		}
		if len(got.Ranks) != len(worthFilters) {
			t.Fatalf("%s: the driver ranked %d filters, want %d", run.name, len(got.Ranks), len(worthFilters))
		}
		for k, f := range worthFilters {
			want := nativeRanking(s.Rank(f.native(posOf), worthLimit))
			if browser, native := mustJSON(t, got.Ranks[k]), mustJSON(t, want); browser != native {
				t.Errorf("%s, filter %+v: the browser build and the native build disagree.\n--- wasm ---\n%s\n--- native ---\n%s", run.name, f, browser, native)
			}
			rankings[run.name] = append(rankings[run.name], want)
		}
	}

	all := rankings["no skills"][0]
	if !slices.EqualFunc(all.Needed, []worthNeeded{{"Story/2-1", [][]int{{betterHair}}}}, func(a, b worthNeeded) bool {
		return a.Key == b.Key && slices.EqualFunc(a.Missing, b.Missing, slices.Equal)
	}) {
		t.Errorf("needed %+v, want Story/2-1 missing the better hair", all.Needed)
	}
	if !slices.ContainsFunc(all.Rows, func(row worthRow) bool { return slices.Equal(row.Items, []int{40001, 50001}) }) {
		t.Errorf("no row pairs the top with the bottom, so pairs are not checked: %+v", all.Rows)
	}
	if !slices.ContainsFunc(all.Rows, func(row worthRow) bool {
		return slices.ContainsFunc(row.Examples, func(ex worthExample) bool { return ex.Key == "Co-op/Duo" })
	}) {
		t.Errorf("nothing improves Co-op/Duo, so the one-of rule is not checked: %+v", all.Rows)
	}
	accessories := map[int]bool{}
	for _, row := range rankings["no skills"][2].Rows {
		for _, place := range row.Places {
			accessories[place] = true
		}
	}
	if len(accessories) < 2 {
		t.Errorf("the accessory rows sit in places %v, so nothing tells an item's place from its slot", slices.Sorted(maps.Keys(accessories)))
	}
	for k, want := range map[int][][]int{4: {{9}, {11}}, 5: {{3, 4}}, 6: {{15}, {17}}, 7: nil} {
		var got [][]int
		for _, row := range rankings["no skills"][k].Rows {
			got = append(got, row.Places)
		}
		slices.SortFunc(got, slices.Compare)
		if !slices.EqualFunc(got, want, slices.Equal) {
			t.Errorf("filter %+v ranks items in places %v, want %v", worthFilters[k], got, want)
		}
	}
	suits := rankings["no skills"][8]
	named := map[string][]int{}
	for _, row := range suits.Rows {
		named[row.Suit] = row.Items
		if len(row.Places) != len(row.Items) {
			t.Errorf("%s lists %d pieces and %d places", row.Suit, len(row.Items), len(row.Places))
		}
	}
	for key, want := range map[string][]int{"Separates": {40001, 50001}, "Night \"Gala\" set": {20001, 170002, 170004}, "Hands": {rightHand, 170009}} {
		if got, ok := named[key]; !ok || !slices.Equal(got, want) {
			t.Errorf("suit %q lists %v, want %v, in %+v", key, got, want, suits.Rows)
		}
	}
	if _, ok := named["Owned"]; ok {
		t.Error("a suit you own whole is listed")
	}
	if !slices.ContainsFunc(suits.Rows, func(row worthRow) bool {
		return row.Suit == "Hands" && slices.ContainsFunc(row.Examples, func(ex worthExample) bool { return ex.Key == "Co-op/Duo" })
	}) {
		t.Errorf("the suit holding the required right hand does not improve Co-op/Duo, so the engine fallback is not checked: %+v", suits.Rows)
	}
	for _, row := range rankings["no skills"][10].Rows {
		if !slices.ContainsFunc(row.Places, func(place int) bool { return place >= 8 && place != 14 }) {
			t.Errorf("the accessory filter lists %s, which has no accessory: %+v", row.Suit, row.Places)
		}
	}
	for _, row := range rankings["no skills"][0].Rows {
		if row.Suit != "" {
			t.Errorf("an item row carries the suit %q", row.Suit)
		}
	}
	if mustJSON(t, rankings["no skills"]) == mustJSON(t, rankings["auto"]) || mustJSON(t, rankings["auto"]) == mustJSON(t, rankings["auto at 4/2"]) {
		t.Error("skills change nothing in the rankings, so nothing checks they reach the session")
	}
	if mustJSON(t, rankings["no skills"]) != mustJSON(t, rankings["auto off"]) {
		t.Error("{auto: false} ranks differently from no skills")
	}
}

func TestWorthRejectsBadInput(t *testing.T) {
	packed := catalogue.Write(fixture)
	var r struct {
		Worth worthBrowser `json:"worth"`
	}
	if err := json.Unmarshal(browserOutput(t, packed, selectionsFile(ownedIDs())), &r); err != nil {
		t.Fatalf("parsing the browser build's result: %v", err)
	}
	positions, posOf := worthCatalogue(t, packed)
	_, versions := worthVersions(positions, posOf)
	var want []string
	for _, b := range worthBadStarts(versions) {
		want = append(want, b.want)
	}
	for _, b := range worthBadRuns {
		want = append(want, b.want)
	}
	for _, b := range worthBadRanks {
		want = append(want, b.want)
	}
	want = append(want, worthBigInts...)
	if len(r.Worth.Rejected) != len(want) {
		t.Fatalf("the driver tried %d bad calls, want %d", len(r.Worth.Rejected), len(want))
	}
	for i, got := range r.Worth.Rejected {
		if got != want[i] {
			t.Errorf("bad call %d: got %q, want %q", i, got, want[i])
		}
	}
	if r.Worth.Kept != "accepted" {
		t.Errorf("the live session was refused before the wardrobe changed: %q", r.Worth.Kept)
	}
	for _, got := range r.Worth.Dropped {
		if got != "no session" {
			t.Errorf("after the wardrobe changed the session still answers: %q", got)
		}
	}
}
