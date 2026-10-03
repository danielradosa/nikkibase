package themes

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
)

func TestGroupsFindTheThemesOfAStage(t *testing.T) {
	keys := []string{
		"Story/9-9-2", "Story/9-8", "Story/9-9-1", "Story/9-6-1", "Commission/9-9", "Story/9-6-2",
		"Story/9-9-3", "Story/II-4-1-1", "Story/II-4-1-2", "Story/10-9-1", "Arena/9-9-1", "Story/9-9-1#maiden",
	}
	want := [][]int{{2, 0, 6}, {3, 5}, {7, 8}}
	if got := Groups(keys); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := Groups([]string{"Story/9-9-1#maiden", "Story/9-9-2#maiden", "Story/9-9-1"}); !reflect.DeepEqual(got, [][]int{{0, 1}}) {
		t.Errorf("a difficulty's themes group on their own: got %v", got)
	}
}

type fixture struct {
	positions []optimizer.Position
	stages    []scoring.Stage
}

func newFixture(rng *rand.Rand, themes int) fixture {
	var fx fixture
	slots := []scoring.Slot{scoring.Hair, scoring.Dress, scoring.Coat, scoring.Shoes, scoring.Accessory, scoring.Accessory}
	for at, slot := range slots {
		var items []scoring.Item
		for n := range 2 {
			it := scoring.Item{ID: (at+1)*10 + n, Slot: slot}
			for p := range 5 {
				it.Attrs[p] = int8(2*p + rng.IntN(2))
				it.Stats[p] = rng.IntN(120)
			}
			items = append(items, it)
		}
		fx.positions = append(fx.positions, optimizer.Position{Items: items})
	}
	for range themes {
		var sliders [5]int
		for p := range sliders {
			sliders[p] = rng.IntN(201) - 100
		}
		fx.stages = append(fx.stages, scoring.CustomStage(sliders, nil))
	}
	return fx
}

func (fx fixture) solve(t int, exclude map[int]bool) optimizer.Result {
	return optimizer.Best(Without(fx.positions, exclude), fx.stages[t], nil)
}

func (fx fixture) alone() []int {
	refs := make([]int, len(fx.stages))
	for t := range fx.stages {
		refs[t] = fx.solve(t, nil).Score
	}
	return refs
}

func weakest(results []optimizer.Result, refs []int) float64 {
	m := 2.0
	for t, r := range results {
		m = min(m, float64(r.Score)/float64(refs[t]))
	}
	return m
}

func ids(r optimizer.Result) []int {
	out := make([]int, len(r.Items))
	for i, it := range r.Items {
		out[i] = it.ID
	}
	return out
}

func TestSplitNeverWearsAnItemTwiceAndKeepsTheBestOrder(t *testing.T) {
	rng := rand.New(rand.NewPCG(11, 4))
	for trial := range 200 {
		fx := newFixture(rng, 2+trial%2)
		refs := fx.alone()
		got := Split(len(fx.stages), refs, fx.solve)
		seen := map[int]int{}
		for th, r := range got {
			for _, id := range ids(r) {
				if other, ok := seen[id]; ok {
					t.Fatalf("trial %d: item %d is worn in themes %d and %d", trial, id, other, th)
				}
				seen[id] = th
			}
		}
		for _, order := range orders(len(fx.stages)) {
			results := make([]optimizer.Result, len(fx.stages))
			used := map[int]bool{}
			for _, th := range order {
				results[th] = fx.solve(th, used)
				for _, id := range ids(results[th]) {
					used[id] = true
				}
			}
			if weakest(results, refs) > weakest(got, refs) {
				t.Fatalf("trial %d: order %v does better than the split", trial, order)
			}
		}
		for th, others := range Others(got) {
			want := []int{}
			for o, r := range got {
				if o != th {
					want = append(want, ids(r)...)
				}
			}
			slices.Sort(want)
			if !slices.Equal(others, want) {
				t.Fatalf("trial %d theme %d: others %v, want %v", trial, th, others, want)
			}
		}
	}
}

func TestSplitIsCloseToTheBestPossibleAssignment(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 23))
	exact, trials := 0, 120
	worst := 1.0
	for range trials {
		fx := newFixture(rng, 2)
		refs := fx.alone()
		got := weakest(Split(2, refs, fx.solve), refs)
		var all []int
		for _, p := range fx.positions {
			for _, it := range p.Items {
				all = append(all, it.ID)
			}
		}
		best := 0.0
		for mask := range 1 << len(all) {
			first, second := map[int]bool{}, map[int]bool{}
			for i, id := range all {
				if mask&(1<<i) != 0 {
					first[id] = true
				} else {
					second[id] = true
				}
			}
			best = max(best, weakest([]optimizer.Result{fx.solve(0, first), fx.solve(1, second)}, refs))
		}
		if got > best+1e-12 {
			t.Fatalf("the split (%v) beats the exhaustive search (%v)", got, best)
		}
		if got == best {
			exact++
		}
		worst = min(worst, got/best)
	}
	if exact*100 < trials*95 || worst < 0.93 {
		t.Errorf("the split matched the exhaustive assignment on %d of %d fixtures, worst at %.4f of it", exact, trials, worst)
	}
}

func TestTwoIdenticalThemesShareTheItemsOut(t *testing.T) {
	fx := newFixture(rand.New(rand.NewPCG(1, 2)), 2)
	for i := range fx.positions {
		fx.positions[i].Items = fx.positions[i].Items[:1]
	}
	fx.stages[1] = fx.stages[0]
	refs := fx.alone()
	got := Split(2, refs, fx.solve)
	if got[0].Score == 0 || got[1].Score == 0 {
		t.Errorf("one theme was left with nothing: %v", []int{got[0].Score, got[1].Score})
	}
	if got[0].Score+got[1].Score > refs[0] {
		t.Errorf("one item per place can't score more than one full outfit: %v of %d", []int{got[0].Score, got[1].Score}, refs[0])
	}
}

func TestSettleKeepsTheSplitApartAndNeverScoresLess(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 17))
	for trial := range 150 {
		fx := newFixture(rng, 2+trial%2)
		refs := fx.alone()
		split := Split(len(fx.stages), refs, fx.solve)
		final := Settle(split, fx.solve)
		seen := map[int]int{}
		for th, r := range final {
			if r.Score < split[th].Score {
				t.Fatalf("trial %d theme %d: settled %d below the split's %d", trial, th, r.Score, split[th].Score)
			}
			for _, id := range ids(r) {
				if other, ok := seen[id]; ok {
					t.Fatalf("trial %d: item %d is worn in themes %d and %d", trial, id, other, th)
				}
				seen[id] = th
			}
		}
	}
}
