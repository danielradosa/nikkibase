//go:build dataset

package golden

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/catalogue"
	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
	"github.com/danielradosa/nikkibase/pipeline"
)

var (
	update = flag.Bool("update", false, "rewrite testdata/best-possible.golden from the current bundle")
	data   = flag.String("data", "../web/public/data", "the directory holding index.json and the versioned bundles")
	ideal  = flag.String("ideal", "../web/src/generated/ideal.json", "the precomputed best possible outfits the site ships")
)

type stage struct {
	Name string `json:"name"`
	Mode string `json:"mode"`
	nums
	Variants map[string]nums `json:"variants"`
}

type nums struct {
	Weights []float64      `json:"weights"`
	Attrs   []int8         `json:"attrs"`
	Tags    map[string]int `json:"tags"`
	Rules   *rules         `json:"rules"`
}

type rules struct {
	Require [][]int `json:"require"`
}

type version struct {
	variant string
	label   string
	st      scoring.Stage
	require [][]int
}

func (s stage) versions() []version {
	var require [][]int
	if s.Rules != nil {
		require = s.Rules.Require
	}
	out := []version{{"", s.Name, s.scoring(), require}}
	for _, d := range slices.Sorted(maps.Keys(s.Variants)) {
		label := fmt.Sprintf("%s (%s)", s.Name, strings.ToUpper(d[:1])+d[1:])
		v := version{d, label, s.Variants[d].scoring(), require}
		if s.Variants[d].Rules != nil {
			v.require = s.Variants[d].Rules.Require
		}
		out = append(out, v)
	}
	return out
}

func (s stage) scoring() scoring.Stage { return s.nums.scoring() }

func (s nums) scoring() scoring.Stage {
	var st scoring.Stage
	copy(st.Weights[:], s.Weights)
	copy(st.Attrs[:], s.Attrs)
	if len(s.Tags) > 0 {
		st.Tags = make(map[int]int, len(s.Tags))
		for k, v := range s.Tags {
			id, _ := strconv.Atoi(k)
			st.Tags[id] = v
		}
	}
	return st
}

type bundle struct {
	version   string
	catalogue *catalogue.Catalogue
	stages    []stage
	positions []optimizer.Position
	posOf     map[int]int
	placeOf   map[int]uint16
}

func load(t *testing.T) *bundle {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(*data, "index.json"))
	if err != nil {
		t.Fatalf("no bundle to check (build one with cmd/bundle, see SOURCES.md): %v", err)
	}
	var index struct{ Version string }
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(*data, index.Version)
	b := &bundle{version: index.Version}
	if raw, err = os.ReadFile(filepath.Join(dir, "items.bin")); err != nil {
		t.Fatal(err)
	}
	if b.catalogue, err = catalogue.Read(raw); err != nil {
		t.Fatal(err)
	}
	if raw, err = os.ReadFile(filepath.Join(dir, "stages.json")); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &b.stages); err != nil {
		t.Fatal(err)
	}
	b.positions, b.posOf, b.placeOf = optimizer.FromCatalogue(b.catalogue, nil)
	return b
}

func (b *bundle) space(ver version) optimizer.Space {
	return optimizer.Require(b.positions, b.posOf, ver.require)
}

func (b *bundle) best(ver version) optimizer.Result {
	return b.space(ver).Best(ver.st, nil)
}

func unmet(worn []int, require [][]int) [][]int {
	var out [][]int
	for _, set := range require {
		met := false
		for _, id := range worn {
			if slices.Contains(set, id) {
				met = true
			}
		}
		if !met {
			out = append(out, set)
		}
	}
	return out
}

func wornIDs(items []scoring.Item) []int {
	ids := make([]int, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	return ids
}

func placedIDs(items [][2]int) []int {
	ids := make([]int, len(items))
	for i, it := range items {
		ids[i] = it[0]
	}
	return ids
}

func (b *bundle) find(t *testing.T, mode, name string) stage {
	t.Helper()
	for _, s := range b.stages {
		if s.Mode == mode && s.Name == name {
			return s
		}
	}
	t.Fatalf("the bundle has no %s stage %s", mode, name)
	return stage{}
}

func TestShippedCatalogueIsWearable(t *testing.T) {
	b := load(t)
	c := b.catalogue
	places := map[scoring.Slot]map[uint16]bool{}
	ids := make(map[int32]bool, len(c.IDs))
	placeSlot := map[uint16]scoring.Slot{}
	placeGroup := map[uint16]uint8{}
	for i, id := range c.IDs {
		slot := scoring.Slot(c.Slots[i])
		if ids[id] {
			t.Errorf("item %d appears twice", id)
		}
		ids[id] = true
		if want := pipeline.SlotOfID(int(id)); slot != want {
			t.Errorf("item %d is scored as %s, and its ID says %s", id, pipeline.SlotName(slot), pipeline.SlotName(want))
		}
		place := c.Positions[i]
		if places[slot] == nil {
			places[slot] = map[uint16]bool{}
		}
		places[slot][place] = true
		if s, seen := placeSlot[place]; seen && s != slot {
			t.Errorf("place %d holds both %s and %s items", place, pipeline.SlotName(s), pipeline.SlotName(slot))
		}
		placeSlot[place] = slot
		if g, seen := placeGroup[place]; seen && g != c.Groups[i] {
			t.Errorf("place %d holds items in two exclusion groups", place)
		}
		placeGroup[place] = c.Groups[i]
	}
	total := 0
	for slot, set := range places {
		total += len(set)
		if len(set) > scoring.SlotLimit(slot) {
			t.Errorf("%s holds %d places, and the game has %d", pipeline.SlotName(slot), len(set), scoring.SlotLimit(slot))
		}
	}
	if total != len(pipeline.SubSlots()) {
		t.Errorf("the catalogue fills %d places, and the game has %d", total, len(pipeline.SubSlots()))
	}
}

func TestEveryBestOutfitIsWearable(t *testing.T) {
	b := load(t)
	subs := pipeline.SubSlots()
	required := 0
	for _, s := range b.stages {
		for _, ver := range s.versions() {
			wearable(t, b, subs, s.Mode+" "+ver.label, ver)
			if len(ver.require) > 0 {
				required++
			}
		}
	}
	t.Logf("%d stage versions require particular items", required)
}

func wearable(t *testing.T, b *bundle, subs []pipeline.SubSlot, name string, ver version) {
	t.Helper()
	got := b.best(ver)
	if missing := unmet(wornIDs(got.Items), ver.require); len(missing) > 0 {
		t.Errorf("%s: wears none of %v, which the stage requires", name, missing)
	}
	bySlot := map[scoring.Slot]int{}
	seen := map[int]bool{}
	for _, it := range got.Items {
		bySlot[it.Slot]++
		if seen[it.ID] {
			t.Errorf("%s: item %d is worn twice", name, it.ID)
		}
		seen[it.ID] = true
	}
	for slot, n := range bySlot {
		if n > scoring.SlotLimit(slot) {
			t.Errorf("%s: %d %s items, and the game allows %d", name, n, pipeline.SlotName(slot), scoring.SlotLimit(slot))
		}
	}
	if bySlot[scoring.Dress] > 0 && bySlot[scoring.Top]+bySlot[scoring.Bottom] > 0 {
		t.Errorf("%s: a dress worn with a top or bottom", name)
	}
	handheld := map[string]bool{}
	for _, it := range got.Items {
		for i, id := range b.catalogue.IDs {
			if int(id) == it.ID {
				handheld[subs[b.catalogue.Positions[i]].Name] = true
				break
			}
		}
	}
	if handheld["accessory_handheld_both"] && (handheld["accessory_handheld_left"] || handheld["accessory_handheld_right"]) {
		t.Errorf("%s: a both-hands item held with a one-hand item", name)
	}
	if want := scoring.Score(got.Items, ver.st, nil); got.Score != want {
		t.Errorf("%s: reported %d for an outfit worth %d", name, got.Score, want)
	}
}

type reference struct {
	Kind      string  `json:"kind"`
	Mode      string  `json:"mode"`
	Stage     string  `json:"stage"`
	Tag       string  `json:"tag"`
	Value     int     `json:"value"`
	Tolerance float64 `json:"tolerance"`
	Source    string  `json:"source"`
}

func references(t *testing.T) []reference {
	t.Helper()
	raw, err := os.ReadFile("testdata/references.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct{ References []reference }
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	return f.References
}

func TestAgainstMeasuredReferences(t *testing.T) {
	b := load(t)
	tags := pipeline.TagNames()
	for _, r := range references(t) {
		s := b.find(t, r.Mode, r.Stage)
		var got int
		switch r.Kind {
		case "same-model":
			got = b.best(s.versions()[0]).Score
		case "tag-award":
			id := slices.Index(tags, r.Tag)
			if id < 0 {
				t.Fatalf("%s %s: no style is named %q", r.Mode, r.Stage, r.Tag)
			}
			got = s.Tags[strconv.Itoa(id)]
		default:
			continue
		}
		off := float64(got-r.Value) / float64(r.Value)
		label := fmt.Sprintf("%s %s %s", r.Mode, r.Stage, r.Tag)
		if math.Abs(off) > r.Tolerance {
			t.Errorf("%s: %d, against a measured %d (%+.1f%%, tolerance %.0f%%)\n  source: %s",
				strings.TrimSpace(label), got, r.Value, off*100, r.Tolerance*100, r.Source)
		} else {
			t.Logf("%s: %d against %d (%+.2f%%)", strings.TrimSpace(label), got, r.Value, off*100)
		}
	}
}

func TestDeviationsAreExplained(t *testing.T) {
	b := load(t)
	doc, err := os.ReadFile("../KNOWN-DEVIATIONS.md")
	if err != nil {
		t.Fatal(err)
	}
	rows := deviationRows(string(doc))
	for _, r := range references(t) {
		if r.Kind != "in-game-ranked" {
			continue
		}
		got := b.best(b.find(t, r.Mode, r.Stage).versions()[0]).Score
		off := float64(got-r.Value) / float64(r.Value)
		entry := fmt.Sprintf("%s %s", r.Mode, r.Stage)
		row, listed := rows[entry]
		if !listed {
			if math.Abs(off) > 0.10 {
				t.Errorf("%s: NikkiBase says %d and the #1 ranked score is %d (%+.0f%%), and KNOWN-DEVIATIONS.md has no row for it",
					entry, got, r.Value, off*100)
			}
			continue
		}
		gap := fmt.Sprintf("%+.0f%%", off*100)
		if row.best != got || row.ranked != r.Value || row.gap != gap {
			t.Errorf("%s: KNOWN-DEVIATIONS.md says %d against %d (%s), and it is now %d against %d (%s)",
				entry, row.best, row.ranked, row.gap, got, r.Value, gap)
		}
	}
}

type deviationRow struct {
	best, ranked int
	gap          string
}

func deviationRows(doc string) map[string]deviationRow {
	rows := map[string]deviationRow{}
	for _, line := range strings.Split(doc, "\n") {
		cells := strings.Split(strings.TrimSpace(line), "|")
		if len(cells) != 6 || cells[0] != "" || cells[5] != "" {
			continue
		}
		best, err1 := strconv.Atoi(strings.ReplaceAll(strings.TrimSpace(cells[2]), ",", ""))
		ranked, err2 := strconv.Atoi(strings.ReplaceAll(strings.TrimSpace(cells[3]), ",", ""))
		if err1 != nil || err2 != nil {
			continue
		}
		rows[strings.TrimSpace(cells[1])] = deviationRow{best, ranked, strings.TrimSpace(cells[4])}
	}
	return rows
}

func TestPinnedBestPossible(t *testing.T) {
	b := load(t)
	var out strings.Builder
	fmt.Fprintf(&out, "# Whole-catalogue best score and outfit size for every stage in bundle %s.\n", b.version)
	fmt.Fprintf(&out, "# Regenerate after a deliberate change: go test -tags dataset ./golden/ -run Pinned -update\n")
	for _, s := range b.stages {
		for _, ver := range s.versions() {
			got := b.best(ver)
			if missing := unmet(wornIDs(got.Items), ver.require); len(missing) > 0 {
				t.Errorf("%s %s: wears none of %v, which the stage requires", s.Mode, ver.label, missing)
			}
			fmt.Fprintf(&out, "%s\t%s\t%d\t%d\n", s.Mode, ver.label, got.Score, len(got.Items))
		}
	}
	const path = "testdata/best-possible.golden"
	if *update {
		if err := os.WriteFile(path, []byte(out.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (create it with -update)", err)
	}
	if out.String() != string(want) {
		gotLines, wantLines := strings.Split(out.String(), "\n"), strings.Split(string(want), "\n")
		shown := 0
		for i := 0; i < max(len(gotLines), len(wantLines)) && shown < 20; i++ {
			var g, w string
			if i < len(gotLines) {
				g = gotLines[i]
			}
			if i < len(wantLines) {
				w = wantLines[i]
			}
			if g != w {
				t.Errorf("line %d:\n  pinned: %s\n  now:    %s", i+1, w, g)
				shown++
			}
		}
	}
}

type shippedOutfit struct {
	Score int          `json:"score"`
	Items [][2]int     `json:"items"`
	Auto  *shippedAuto `json:"auto"`
}

type shippedAuto struct {
	CharmSmile int      `json:"charmSmile"`
	Smile      int      `json:"smile"`
	Score      int      `json:"score"`
	Items      [][2]int `json:"items"`
}

func TestShippedIdealIsCurrent(t *testing.T) {
	b := load(t)
	dataDir, _ := filepath.Abs(*data)
	out, _ := filepath.Abs(*ideal)
	regenerate := fmt.Sprintf("go run ./cmd/ideal -data %s -out %s", dataDir, out)

	raw, err := os.ReadFile(*ideal)
	if err != nil {
		t.Fatalf("no precomputed best possible outfits (generate them from the repository root with: %s): %v", regenerate, err)
	}
	var shipped struct {
		Version string `json:"version"`
		Stages  map[string]struct {
			shippedOutfit
			Variants map[string]shippedOutfit `json:"variants"`
		} `json:"stages"`
	}
	if err := json.Unmarshal(raw, &shipped); err != nil {
		t.Fatalf("%s: %v (regenerate with: %s)", *ideal, err, regenerate)
	}
	if shipped.Version != b.version {
		t.Fatalf("%s was computed from bundle %q, and index.json names %q (regenerate with: %s)",
			*ideal, shipped.Version, b.version, regenerate)
	}

	keys, err := stageKeys(raw)
	if err != nil {
		t.Fatalf("%s: %v", *ideal, err)
	}
	want := make([]string, 0, len(b.stages))
	for _, s := range b.stages {
		want = append(want, s.Mode+"/"+s.Name)
	}
	if !slices.Equal(keys, want) {
		i := 0
		for i < min(len(keys), len(want)) && keys[i] == want[i] {
			i++
		}
		t.Errorf("%s lists %d stages and the bundle has %d, each once and in the bundle's order; entry %d differs",
			*ideal, len(keys), len(want), i)
	}

	stale := 0
	report := func(format string, args ...any) {
		stale++
		if stale <= 20 {
			t.Errorf(format, args...)
		}
	}
	for _, s := range b.stages {
		key := s.Mode + "/" + s.Name
		got, ok := shipped.Stages[key]
		if !ok {
			report("%s: not shipped", key)
			continue
		}
		if have, bundled := slices.Sorted(maps.Keys(got.Variants)), slices.Sorted(maps.Keys(s.Variants)); !slices.Equal(have, bundled) {
			report("%s: shipped variants %v, and the bundle has %v", key, have, bundled)
		}
		for _, ver := range s.versions() {
			o := got.shippedOutfit
			if ver.variant != "" {
				if o, ok = got.Variants[ver.variant]; !ok {
					continue
				}
			}
			space := b.space(ver)
			best := space.Best(ver.st, nil)
			items := b.placed(best)
			if o.Score != best.Score || !slices.Equal(o.Items, items) {
				report("%s %s: shipped %d wearing %v, and the bundle gives %d wearing %v",
					s.Mode, ver.label, o.Score, o.Items, best.Score, items)
			}
			if missing := unmet(placedIDs(o.Items), ver.require); len(missing) > 0 {
				report("%s %s: shipped an outfit that wears none of %v, which the stage requires", s.Mode, ver.label, missing)
			}
			auto, placement := space.BestPlacedFrom(ver.st, best)
			want := shippedAuto{placement.CharmSmile, placement.Smile, auto.Score, b.placed(auto)}
			if o.Auto == nil {
				report("%s %s: shipped without auto skills", s.Mode, ver.label)
			} else if a := *o.Auto; a.CharmSmile != want.CharmSmile || a.Smile != want.Smile ||
				a.Score != want.Score || !slices.Equal(a.Items, want.Items) {
				report("%s %s: shipped auto %+v, and the bundle gives %+v", s.Mode, ver.label, *o.Auto, want)
			} else if missing := unmet(placedIDs(a.Items), ver.require); len(missing) > 0 {
				report("%s %s: shipped an auto outfit that wears none of %v, which the stage requires", s.Mode, ver.label, missing)
			}
		}
	}
	for _, key := range slices.Sorted(maps.Keys(shipped.Stages)) {
		if !slices.Contains(want, key) {
			report("%s: shipped, and the bundle has no such stage", key)
		}
	}
	if stale > 0 {
		t.Errorf("%s is stale (%d findings, the first 20 shown); regenerate with: %s", *ideal, stale, regenerate)
	}
}

func TestBestPossibleItemsSayHowToGetThem(t *testing.T) {
	b := load(t)
	raw, err := os.ReadFile(*ideal)
	if err != nil {
		t.Fatalf("no precomputed best possible outfits to check (generate them with cmd/ideal): %v", err)
	}
	var shipped struct {
		Version string `json:"version"`
		Stages  map[string]struct {
			shippedOutfit
			Variants map[string]shippedOutfit `json:"variants"`
		} `json:"stages"`
	}
	if err := json.Unmarshal(raw, &shipped); err != nil {
		t.Fatal(err)
	}
	if shipped.Version != b.version {
		t.Fatalf("%s was computed from bundle %q, and index.json names %q", *ideal, shipped.Version, b.version)
	}
	worn := map[int]bool{}
	wear := func(o shippedOutfit) {
		for _, it := range o.Items {
			worn[it[0]] = true
		}
		if o.Auto != nil {
			for _, it := range o.Auto.Items {
				worn[it[0]] = true
			}
		}
	}
	for _, s := range shipped.Stages {
		wear(s.shippedOutfit)
		for _, v := range s.Variants {
			wear(v)
		}
	}

	if raw, err = os.ReadFile(filepath.Join(*data, b.version, "acquire.json")); err != nil {
		t.Fatal(err)
	}
	var acq struct {
		Version string                       `json:"version"`
		Items   map[string][]json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(raw, &acq); err != nil {
		t.Fatal(err)
	}
	if acq.Version != b.version {
		t.Errorf("acquire.json says version %q, and the bundle is %q", acq.Version, b.version)
	}
	if raw, err = os.ReadFile("../data/acquisition-gaps.json"); err != nil {
		t.Fatal(err)
	}
	var gaps struct {
		Items []struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Reason string `json:"reason"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &gaps); err != nil {
		t.Fatal(err)
	}
	listed := map[int]bool{}
	for _, g := range gaps.Items {
		switch {
		case g.Reason == "":
			t.Errorf("data/acquisition-gaps.json: %d (%s) gives no reason", g.ID, g.Name)
		case len(acq.Items[strconv.Itoa(g.ID)]) > 0:
			t.Errorf("data/acquisition-gaps.json: %d (%s) now says how to get it; remove it from the list", g.ID, g.Name)
		case !worn[g.ID]:
			t.Errorf("data/acquisition-gaps.json: %d (%s) is no longer in a best possible outfit; remove it from the list", g.ID, g.Name)
		}
		listed[g.ID] = true
	}
	var missing []int
	for id := range worn {
		if len(acq.Items[strconv.Itoa(id)]) == 0 && !listed[id] {
			missing = append(missing, id)
		}
	}
	slices.Sort(missing)
	if len(missing) > 0 {
		t.Errorf("%d items in best possible outfits have no line in acquire.json and are not in data/acquisition-gaps.json: %v",
			len(missing), missing)
	}
	t.Logf("%d items in best possible outfits, %d of them without a line (listed)", len(worn), len(listed))
}

func (b *bundle) placed(r optimizer.Result) [][2]int {
	items := make([][2]int, len(r.Items))
	for i, it := range r.Items {
		items[i] = [2]int{it.ID, int(b.placeOf[it.ID])}
	}
	return items
}

func stageKeys(raw []byte) ([]string, error) {
	var top struct {
		Stages json.RawMessage `json:"stages"`
	}
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(top.Stages))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil, fmt.Errorf("its stages are not an object")
	}
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		keys = append(keys, tok.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return nil, err
		}
	}
	return keys, nil
}
