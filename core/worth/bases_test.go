package worth

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/danielradosa/nikkibase/core/optimizer"
)

func TestBasesMatchTheEngine(t *testing.T) {
	rng := rand.New(rand.NewPCG(31, 7))
	checked, failing := 0, 0
	for trial := range 120 {
		fx := newFixture(rng, places, 4, 6, trial%2 == 1)
		bases := fx.session().Bases()
		if len(bases) != len(fx.versions) {
			t.Fatalf("trial %d: %d bases for %d stages", trial, len(bases), len(fx.versions))
		}
		for i, v := range fx.versions {
			b := bases[i]
			r := fx.search(fx.owned2(), v)
			unmet := len(optimizer.Unmet(r.Items, v.Require)) > 0
			if b.Key != v.Key || b.Failing != unmet {
				t.Errorf("trial %d %s: got key %s failing %v, want failing %v", trial, v.Key, b.Key, b.Failing, unmet)
				continue
			}
			if unmet {
				failing++
				continue
			}
			if b.Score != r.Score {
				t.Errorf("trial %d %s: base %d, the engine finds %d", trial, v.Key, b.Score, r.Score)
			}
			checked++
		}
	}
	if checked < 300 || failing < 20 {
		t.Errorf("checked %d passing and %d failing stages; the fixtures no longer cover both", checked, failing)
	}
}

func TestBasesCoverOnlyTheStagesRunSoFar(t *testing.T) {
	fx := newFixture(rand.New(rand.NewPCG(3, 9)), places, 3, 5, false)
	s := NewSession(fx.positions, fx.posOf, fx.own, fx.versions, fx.settings)
	if got := s.Bases(); len(got) != 0 {
		t.Errorf("before a run: %+v", got)
	}
	s.Run(2)
	got := s.Bases()
	if len(got) != 2 || got[0].Key != fx.versions[0].Key || got[1].Key != fx.versions[1].Key {
		t.Errorf("after two stages: %+v", got)
	}
}

func excludeSome(rng *rand.Rand, fx *fixture) int {
	var owned []int
	for _, p := range fx.positions {
		for _, it := range p.Items {
			if fx.owned[it.ID] {
				owned = append(owned, it.ID)
			}
		}
	}
	excluded := 0
	for vi := range fx.versions {
		if rng.IntN(2) == 0 || len(owned) == 0 {
			continue
		}
		for range 1 + rng.IntN(4) {
			fx.versions[vi].Exclude = append(fx.versions[vi].Exclude, owned[rng.IntN(len(owned))])
		}
		excluded++
	}
	return excluded
}

func TestGainsMatchTheEngineWithItemsLeftOut(t *testing.T) {
	rng := rand.New(rand.NewPCG(41, 3))
	var n tally
	left := 0
	for trial := range 150 {
		fx := newFixture(rng, places, 4, 6, trial%2 == 1)
		left += excludeSome(rng, fx)
		checkGains(t, trial, fx, fx.session(), &n)
	}
	if left < 300 || n.pairs < 10000 || n.exact*100 < n.pairs*97 {
		t.Errorf("%d versions left items out; checked %d pairs: %d exact", left, n.pairs, n.exact)
	}
}

func TestBasesLeaveOutWhatTheVersionExcludes(t *testing.T) {
	rng := rand.New(rand.NewPCG(8, 13))
	checked := 0
	for trial := range 120 {
		fx := newFixture(rng, places, 4, 6, trial%2 == 1)
		excludeSome(rng, fx)
		for i, b := range fx.session().Bases() {
			v := fx.versions[i]
			has := func(id int) bool { return fx.owned[id] && !slices.Contains(v.Exclude, id) }
			r := fx.search(has, v)
			if len(optimizer.Unmet(r.Items, v.Require)) > 0 {
				continue
			}
			if b.Score != r.Score {
				t.Errorf("trial %d %s leaving out %v: base %d, the engine finds %d", trial, v.Key, v.Exclude, b.Score, r.Score)
			}
			checked++
		}
	}
	if checked < 300 {
		t.Errorf("checked only %d bases", checked)
	}
}
