//go:build dataset

package worth

import (
	"encoding/json"
	"flag"
	"maps"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/danielradosa/nikkibase/core/catalogue"
	"github.com/danielradosa/nikkibase/core/ideal"
	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
	"github.com/danielradosa/nikkibase/core/wardrobe"
)

var (
	dataDir      = flag.String("data", "../../web/public/data", "the directory holding index.json and the versioned bundles")
	wardrobeFile = flag.String("wardrobe", "", "an @SEL selections file used as the owned pool (default: a fixed sample)")
	pairsFlag    = flag.Int("pairs", 300, "stage and item pairs compared with the engine per skill setting")
)

type bundleData struct {
	cat       *catalogue.Catalogue
	positions []optimizer.Position
	posOf     map[int]int
	stages    []ideal.Stage
	have      map[int32]bool
	names     map[int]string
	suits     []Suit
}

func loadBundle(t *testing.T) *bundleData {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(*dataDir, "index.json"))
	if err != nil {
		t.Skipf("no data bundle: %v", err)
	}
	var index struct{ Version string }
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(*dataDir, index.Version)
	b := &bundleData{have: map[int32]bool{}, names: map[int]string{}}
	if raw, err = os.ReadFile(filepath.Join(dir, "items.bin")); err != nil {
		t.Fatal(err)
	}
	if b.cat, err = catalogue.Read(raw); err != nil {
		t.Fatal(err)
	}
	b.positions, b.posOf, _ = optimizer.FromCatalogue(b.cat, nil)
	if raw, err = os.ReadFile(filepath.Join(dir, "stages.json")); err != nil {
		t.Fatal(err)
	}
	if b.stages, err = ideal.ReadStages(raw); err != nil {
		t.Fatal(err)
	}
	if raw, err = os.ReadFile(filepath.Join(dir, "items.json")); err == nil {
		var items struct{ Items [][]any }
		if json.Unmarshal(raw, &items) == nil {
			of := map[string]int{}
			for _, row := range items.Items {
				id, ok := row[0].(float64)
				if !ok {
					continue
				}
				b.names[int(id)], _ = row[1].(string)
				if len(row) < 15 {
					continue
				}
				if suit, _ := row[14].(string); suit != "" {
					k, seen := of[suit]
					if !seen {
						k = len(b.suits)
						of[suit] = k
						b.suits = append(b.suits, Suit{Key: suit})
					}
					b.suits[k].Items = append(b.suits[k].Items, int(id))
				}
			}
		}
	}
	if *wardrobeFile != "" {
		if raw, err = os.ReadFile(*wardrobeFile); err != nil {
			t.Fatal(err)
		}
		w, err := wardrobe.DecodeSelections(raw)
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range w.Items {
			b.have[int32(id)] = true
		}
	} else {
		for i, id := range b.cat.IDs {
			if (uint32(i)*2654435761)>>16%3 == 0 {
				b.have[id] = true
			}
		}
	}
	return b
}

func (b *bundleData) own(id int32) bool { return b.have[id] }

func (b *bundleData) engine(owned func(int32) bool, v Version, set Settings) int {
	positions, posOf := b.positions, b.posOf
	if owned != nil {
		positions, posOf, _ = optimizer.FromCatalogue(b.cat, owned)
	}
	return search(positions, posOf, v, set)
}

func search(positions []optimizer.Position, posOf map[int]int, v Version, set Settings) int {
	space := optimizer.Require(positions, posOf, v.Require)
	if set.Skills && !set.Levels.None() {
		r, _ := space.BestPlacedAt(v.Stage, set.Levels)
		return r.Score
	}
	return space.Best(v.Stage, nil).Score
}

func (b *bundleData) versions(t *testing.T, set Settings) []Version {
	t.Helper()
	var out []Version
	add := func(key, mode string, st scoring.Stage, require [][]int) {
		v := Version{Key: key, Mode: mode, Stage: st, Require: require}
		v.Ideal = b.engine(nil, v, set)
		out = append(out, v)
	}
	for _, s := range b.stages {
		add(s.Key(), s.Mode, s.Scoring, s.Require)
		for _, name := range slices.Sorted(maps.Keys(s.Variants)) {
			add(s.Key()+"#"+name, s.Mode, s.Variants[name], s.VariantRequire[name])
		}
	}
	return out
}

var settingsUnderTest = []struct {
	name string
	set  Settings
}{
	{"none", Settings{}},
	{"auto", Settings{Skills: true, Levels: scoring.MaxLevels}},
	{"levels 4/6", Settings{Skills: true, Levels: scoring.Levels{Charming: 4, Smile: 6}}},
}

func TestBundleTiming(t *testing.T) {
	b := loadBundle(t)
	for _, run := range settingsUnderTest {
		versions := b.versions(t, run.set)
		runtime.GC()
		start := time.Now()
		s := NewSession(b.positions, b.posOf, b.own, versions, run.set)
		s.Run(len(versions))
		took := time.Since(start)
		start = time.Now()
		s.Rank(Filter{}, 50)
		all := time.Since(start)
		start = time.Now()
		s.Rank(Filter{Modes: []string{"Story"}}, 50)
		t.Logf("%s: run %v, rank all %v, rank story %v", run.name, took, all, time.Since(start))
	}
}

func TestBundleTimingWithRequiredItems(t *testing.T) {
	b := loadBundle(t)
	have := maps.Clone(b.have)
	for _, s := range b.stages {
		sets := slices.Clone(s.Require)
		for _, name := range slices.Sorted(maps.Keys(s.VariantRequire)) {
			sets = append(sets, s.VariantRequire[name]...)
		}
		for _, set := range sets {
			for _, id := range set {
				if _, ok := b.posOf[id]; ok {
					have[int32(id)] = true
					break
				}
			}
		}
	}
	b.have = have
	for _, run := range settingsUnderTest[:2] {
		versions := b.versions(t, run.set)
		runtime.GC()
		start := time.Now()
		s := NewSession(b.positions, b.posOf, b.own, versions, run.set)
		s.Run(len(versions))
		took := time.Since(start)
		failing, members := 0, 0
		for _, st := range s.stages {
			if st.failing {
				failing++
			}
			members += len(st.members)
		}
		start = time.Now()
		s.Rank(Filter{}, 50)
		t.Logf("%s: run %v, rank all %v; %d failing, %d required items to price with the engine", run.name, took, time.Since(start), failing, members)
	}
}

func TestBundleValuesAreContribution(t *testing.T) {
	b := loadBundle(t)
	l := newLayout(b.positions, b.posOf, b.own)
	checked := 0
	for _, s := range b.stages {
		for _, sk := range []scoring.Skills{nil, scoring.Placement{CharmSmile: int(s.Scoring.Attrs[0]), Smile: int(s.Scoring.Attrs[3])}.Skills()} {
			c := l.coefOf(s.Scoring, sk)
			if !c.exact {
				t.Fatalf("%s: the packed values are not used", s.Key())
			}
			for i := range int32(len(l.items)) {
				gs, gf := l.value(c, i)
				ws, wf := scoring.Contribution(l.items[i], s.Scoring, sk)
				if gs != ws || gf != wf {
					t.Fatalf("%s item %d: packed %v %v, Contribution %v %v", s.Key(), l.items[i].ID, gs, gf, ws, wf)
				}
				checked++
			}
		}
	}
	t.Logf("%d item values identical to scoring.Contribution", checked)
}

func TestBundleGainsMatchTheEngine(t *testing.T) {
	b := loadBundle(t)
	for _, run := range settingsUnderTest {
		t.Run(run.name, func(t *testing.T) {
			versions := b.versions(t, run.set)
			start := time.Now()
			s := NewSession(b.positions, b.posOf, b.own, versions, run.set)
			s.Run(len(versions))
			t.Logf("session for %d versions: %v", len(versions), time.Since(start))

			bases := map[int]int{}
			failing, off := 0, 0
			mine, mineOf, _ := optimizer.FromCatalogue(b.cat, b.own)
			for vi, v := range versions {
				st := s.stages[vi]
				if want := len(optimizer.Unmet(optimizer.Require(mine, mineOf, v.Require).Best(v.Stage, nil).Items, v.Require)) > 0; want != st.failing {
					t.Errorf("%s: failing %v, and the engine says %v", v.Key, st.failing, want)
				}
				if st.failing {
					failing++
					continue
				}
				bases[vi] = search(mine, mineOf, v, run.set)
				if st.base != bases[vi] {
					off++
					t.Logf("%s: session base %d, engine %d", v.Key, st.base, bases[vi])
				}
			}
			if off > 0 {
				t.Errorf("%d of %d stage versions start from a different score", off, len(bases))
			}

			rng := rand.New(rand.NewPCG(7, uint64(len(run.name))))
			var same, one, more, gained int
			for range *pairsFlag {
				vi := rng.IntN(len(versions))
				st := s.stages[vi]
				if st.failing {
					continue
				}
				var i int32
				if len(st.live) > 0 && rng.IntN(3) > 0 {
					i = st.live[rng.IntN(len(st.live))]
				} else {
					for {
						i = int32(rng.IntN(len(s.l.items)))
						if !s.l.owned[i] {
							break
						}
					}
				}
				id := s.l.items[i].ID
				want := b.engine(func(x int32) bool { return b.have[x] || int(x) == id }, versions[vi], run.set) - bases[vi]
				got := int(st.points(i))
				switch d := got - want; {
				case d == 0:
					same++
				case d == 1 || d == -1:
					one++
				default:
					more++
					t.Errorf("%s item %d: gains %d, and the engine gains %d", versions[vi].Key, id, got, want)
				}
				if want > 0 {
					gained++
				}
			}
			t.Logf("%d failing stage versions; %d pairs identical, %d off by one point, %d off by more; %d with a gain",
				failing, same, one, more, gained)

			for _, view := range []struct {
				name string
				f    Filter
			}{
				{"all", Filter{}},
				{"story", Filter{Modes: []string{"Story"}}},
				{"commission", Filter{Modes: []string{"Commission"}}},
				{"co-op", Filter{Modes: []string{"Co-op"}}},
				{"arena", Filter{Modes: []string{"Arena"}}},
				{"accessories", Filter{Slots: []scoring.Slot{scoring.Accessory}}},
			} {
				start := time.Now()
				r := s.Rank(view.f, 50)
				took := time.Since(start)
				var names []string
				for _, row := range r.Rows[:min(10, len(r.Rows))] {
					var parts []string
					for _, id := range row.Items {
						parts = append(parts, b.names[id])
					}
					names = append(names, strings.Join(parts, " + "))
				}
				t.Logf("rank %s: %d rows, %d needed, %v; first: %q", view.name, len(r.Rows), len(r.Needed), took, names)
			}
		})
	}
}

func TestBundleGainsMatchTheEngineAfterPicks(t *testing.T) {
	b := loadBundle(t)
	for _, run := range settingsUnderTest {
		t.Run(run.name, func(t *testing.T) {
			versions := b.versions(t, run.set)
			s := NewSession(b.positions, b.posOf, b.own, versions, run.set)
			s.Run(len(versions))
			r := s.rankerFor(Filter{})
			var picks []int
			for range 20 {
				row, ok := r.next()
				if !ok {
					break
				}
				picks = append(picks, row.Items...)
			}
			has := func(extra int) func(int32) bool {
				return func(x int32) bool { return b.have[x] || int(x) == extra || slices.Contains(picks, int(x)) }
			}
			bases := map[int]int{}
			rng := rand.New(rand.NewPCG(11, uint64(len(run.name))))
			same, one := 0, 0
			for range *pairsFlag / 2 {
				j := rng.IntN(len(r.vs))
				vi, st := r.vs[j], r.cur[j]
				var i int32
				if len(st.live) > 0 && rng.IntN(3) > 0 {
					i = st.live[rng.IntN(len(st.live))]
				} else {
					i = int32(rng.IntN(len(s.l.items)))
				}
				id := s.l.items[i].ID
				if s.l.owned[i] || r.picked[i] {
					continue
				}
				base, ok := bases[vi]
				if !ok {
					base = b.engine(has(0), versions[vi], run.set)
					bases[vi] = base
					if st.base != base {
						t.Errorf("%s after %d picks: the ranker starts from %d, the engine from %d", versions[vi].Key, len(picks), st.base, base)
					}
				}
				want := b.engine(has(id), versions[vi], run.set) - base
				switch d := int(st.points(i)) - want; {
				case d == 0:
					same++
				case d == 1 || d == -1:
					one++
				default:
					t.Errorf("%s item %d after %d picks: gains %d, and the engine gains %d", versions[vi].Key, id, len(picks), st.points(i), want)
				}
			}
			t.Logf("after %d picks: %d pairs identical, %d off by one point", len(picks), same, one)
		})
	}
}

func TestBundleSuitGainsMatchTheEngine(t *testing.T) {
	b := loadBundle(t)
	if len(b.suits) < 1000 {
		t.Fatalf("items.json names %d suits", len(b.suits))
	}
	for _, run := range settingsUnderTest {
		t.Run(run.name, func(t *testing.T) {
			versions := b.versions(t, run.set)
			set := run.set
			set.Suits = b.suits
			s := NewSession(b.positions, b.posOf, b.own, versions, set)
			s.Run(len(versions))
			start := time.Now()
			rows := s.Rank(Filter{Suits: true}, 20).Rows
			took := time.Since(start)
			var names []string
			for _, row := range rows[:min(10, len(rows))] {
				names = append(names, row.Suit)
			}
			t.Logf("20 suit rows in %v; first: %q", took, names)

			r := s.suitRankerFor(Filter{Suits: true})
			var picks []int
			suitsTaken := 0
			rng := rand.New(rand.NewPCG(13, uint64(len(run.name))))
			for round := range 2 {
				bases := map[int]int{}
				same, one, gained := 0, 0, 0
				for range *pairsFlag / 2 {
					j := rng.IntN(len(r.vs))
					vi := r.vs[j]
					var u int32
					if list := r.list[j]; len(list) > 0 && rng.IntN(3) > 0 {
						u = list[rng.IntN(len(list))].u
					} else {
						u = int32(rng.IntN(len(s.units)))
					}
					if r.picked[u] {
						continue
					}
					var pieces []int
					for _, i := range s.units[u].pieces {
						if !r.taken[i] {
							pieces = append(pieces, s.l.items[i].ID)
						}
					}
					has := func(extra []int) func(int32) bool {
						return func(x int32) bool {
							return b.have[x] || slices.Contains(extra, int(x)) || slices.Contains(picks, int(x))
						}
					}
					base, ok := bases[vi]
					if !ok {
						base = b.engine(has(nil), versions[vi], run.set)
						bases[vi] = base
					}
					want := b.engine(has(pieces), versions[vi], run.set) - base
					switch d := int(r.pointsOf(j, u)) - want; {
					case d == 0:
						same++
					case d == 1 || d == -1:
						one++
					default:
						t.Errorf("%s after %d suits: %s gains %d, and the engine gains %d", versions[vi].Key, suitsTaken, s.units[u].key, r.pointsOf(j, u), want)
					}
					if want != 0 {
						gained++
					}
				}
				t.Logf("after %d suits (%d pieces): %d pairs identical, %d off by one point, %d with a gain", suitsTaken, len(picks), same, one, gained)
				if round == 0 {
					for range 10 {
						row, ok := r.next()
						if !ok {
							break
						}
						picks = append(picks, row.Items...)
						suitsTaken++
					}
				}
			}
		})
	}
}
