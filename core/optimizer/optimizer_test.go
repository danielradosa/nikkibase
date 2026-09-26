package optimizer

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

var livelyStage = scoring.CustomStage([5]int{0, 100, 0, 0, 0}, nil)

func item(id int, slot scoring.Slot, lively int, tags ...int) scoring.Item {
	it := scoring.Item{ID: id, Slot: slot, Tags: tags}
	it.Attrs = [5]int8{scoring.Gorgeous, scoring.Lively, scoring.Mature, scoring.Sexy, scoring.Warm}
	it.Stats[1] = lively
	return it
}

func ids(items []scoring.Item) []int {
	out := make([]int, len(items))
	for i, it := range items {
		out[i] = it.ID
	}
	slices.Sort(out)
	return out
}

func TestPicksBestPerPosition(t *testing.T) {
	got := Best([]Position{
		{Items: []scoring.Item{item(1, scoring.Hair, 50), item(2, scoring.Hair, 80)}},
		{Items: []scoring.Item{item(3, scoring.Shoes, 40), item(4, scoring.Shoes, 10)}},
	}, livelyStage, nil)

	if want := []int{2, 3}; !slices.Equal(ids(got.Items), want) {
		t.Errorf("chose %v, want %v", ids(got.Items), want)
	}
	if got.Score != 1200 {
		t.Errorf("Score = %d, want 1200", got.Score)
	}
}

func TestDressVersusTopAndBottom(t *testing.T) {
	positions := func(dressStat int) []Position {
		return []Position{
			{Items: []scoring.Item{item(1, scoring.Dress, dressStat)}},
			{Items: []scoring.Item{item(2, scoring.Top, 90)}},
			{Items: []scoring.Item{item(3, scoring.Bottom, 90)}},
		}
	}

	if got := Best(positions(200), livelyStage, nil); !slices.Equal(ids(got.Items), []int{1}) {
		t.Errorf("strong dress: chose %v, want the dress alone", ids(got.Items))
	}
	if got := Best(positions(150), livelyStage, nil); !slices.Equal(ids(got.Items), []int{2, 3}) {
		t.Errorf("weak dress: chose %v, want top and bottom", ids(got.Items))
	}
}

func TestWearsFewerAccessoriesWhenExtrasDilute(t *testing.T) {
	var positions []Position
	for i := range 3 {
		positions = append(positions, Position{Items: []scoring.Item{item(i+1, scoring.Accessory, 100)}})
	}
	for i := range 2 {
		positions = append(positions, Position{Items: []scoring.Item{item(i+10, scoring.Accessory, 1)}})
	}

	got := Best(positions, livelyStage, nil)
	if want := []int{1, 2, 3}; !slices.Equal(ids(got.Items), want) {
		t.Errorf("chose %v, want %v", ids(got.Items), want)
	}
	if got.Score != 3000 {
		t.Errorf("Score = %d, want 3000", got.Score)
	}
}

func TestRequiredAccessoryIsWorn(t *testing.T) {
	positions := []Position{
		{Items: []scoring.Item{item(1, scoring.Accessory, 100)}},
		{Items: []scoring.Item{item(2, scoring.Accessory, 100)}},
		{Items: []scoring.Item{item(3, scoring.Accessory, 100)}},
		{Items: []scoring.Item{item(9, scoring.Accessory, 1)}, Required: true},
	}
	got := Best(positions, livelyStage, nil)
	if !slices.Contains(ids(got.Items), 9) {
		t.Errorf("chose %v, want the required item 9 included", ids(got.Items))
	}
}

func TestExcludeDropsBannedItems(t *testing.T) {
	positions := Exclude([]Position{
		{Items: []scoring.Item{item(1, scoring.Hair, 99), item(2, scoring.Hair, 50)}},
	}, 1)

	got := Best(positions, livelyStage, nil)
	if !slices.Equal(ids(got.Items), []int{2}) {
		t.Errorf("chose %v, want the banned item 1 gone", ids(got.Items))
	}
}

func TestRankedOrdersByWorth(t *testing.T) {
	ranked := Ranked([]scoring.Item{
		item(1, scoring.Hair, 10),
		item(2, scoring.Hair, 90),
		item(3, scoring.Hair, 50),
	}, livelyStage, nil)

	if want := []int{2, 3, 1}; !slices.Equal([]int{ranked[0].ID, ranked[1].ID, ranked[2].ID}, want) {
		t.Errorf("order = %v, want %v", []int{ranked[0].ID, ranked[1].ID, ranked[2].ID}, want)
	}
}

func TestMatchesBruteForce(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	slots := []scoring.Slot{scoring.Dress, scoring.Top, scoring.Bottom,
		scoring.Accessory, scoring.Accessory, scoring.Accessory,
		scoring.Accessory, scoring.Accessory, scoring.Accessory}

	for trial := range 60 {
		stage := scoring.CustomStage([5]int{rng.IntN(101), rng.IntN(101), rng.IntN(101), 0, 0},
			map[int]int{1: rng.IntN(500)})

		var positions []Position
		id := 0
		for _, slot := range slots {
			var items []scoring.Item
			for range 1 + rng.IntN(2) {
				id++
				it := item(id, slot, rng.IntN(120))
				it.Attrs[0] = int8(rng.IntN(2))
				it.Stats[0] = rng.IntN(120)
				if rng.IntN(3) == 0 {
					it.Tags = []int{1}
				}
				items = append(items, it)
			}
			positions = append(positions, Position{Items: items})
		}

		got := Best(positions, stage, nil).Score
		if want := bruteForce(positions, stage, nil); got != want {
			t.Fatalf("trial %d: Best scored %d, exhaustive search found %d", trial, got, want)
		}
		placed := scoring.Placement{CharmSmile: rng.IntN(6), Smile: rng.IntN(6)}
		levels := scoring.Levels{Charming: rng.IntN(10), Smile: rng.IntN(10)}
		sk := placed.SkillsAt(levels)
		if got, want := Best(positions, stage, sk).Score, bruteForce(positions, stage, sk); got != want {
			t.Fatalf("trial %d: with %+v at %+v Best scored %d, exhaustive search found %d", trial, placed, levels, got, want)
		}
	}
}

func TestBestPlaced(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	slots := []scoring.Slot{scoring.Hair, scoring.Dress, scoring.Top, scoring.Bottom, scoring.Shoes,
		scoring.Accessory, scoring.Accessory, scoring.Accessory, scoring.Accessory, scoring.Accessory}

	placements := map[scoring.Placement]bool{}
	for trial := range 80 {
		var sliders [5]int
		for p := range sliders {
			sliders[p] = rng.IntN(201) - 100
		}
		stage := scoring.CustomStage(sliders, map[int]int{1: rng.IntN(800)})

		var positions []Position
		id := 0
		for _, slot := range slots {
			var items []scoring.Item
			for range 1 + rng.IntN(3) {
				id++
				it := scoring.Item{ID: id, Slot: slot}
				for p := range 5 {
					it.Attrs[p] = int8(p*2 + rng.IntN(2))
					it.Stats[p] = rng.IntN(150)
				}
				if rng.IntN(3) == 0 {
					it.Tags = []int{1}
				}
				items = append(items, it)
			}
			positions = append(positions, Position{Items: items})
		}

		unskilled := Best(positions, stage, nil)
		wantPlacement := scoring.Place(unskilled.Items, stage)
		want := Best(positions, stage, wantPlacement.Skills())
		got, placement := BestPlaced(positions, stage)
		if placement != wantPlacement {
			t.Fatalf("trial %d: placed %+v, and the unskilled best outfit's points say %+v", trial, placement, wantPlacement)
		}
		sameItem := func(a, b scoring.Item) bool { return a.ID == b.ID }
		if got.Score != want.Score || !slices.EqualFunc(got.Items, want.Items, sameItem) {
			t.Fatalf("trial %d: BestPlaced wears %v for %d, and Best with its skills wears %v for %d",
				trial, ids(got.Items), got.Score, ids(want.Items), want.Score)
		}
		if got.Score < unskilled.Score {
			t.Fatalf("trial %d: %d with skills is below %d without", trial, got.Score, unskilled.Score)
		}
		if from, p := BestPlacedFrom(positions, stage, unskilled); p != placement || from.Score != got.Score {
			t.Fatalf("trial %d: from the unskilled result %+v scores %d, and from scratch %+v scores %d",
				trial, p, from.Score, placement, got.Score)
		}
		placements[placement] = true

		space := Space{branches: [][]Position{positions}}
		if atMax, p := space.BestPlacedAt(stage, scoring.MaxLevels); p != placement || atMax.Score != got.Score || !slices.EqualFunc(atMax.Items, got.Items, sameItem) {
			t.Fatalf("trial %d: max levels gave %+v %d, BestPlaced %+v %d", trial, p, atMax.Score, placement, got.Score)
		}
		levels := scoring.Levels{Charming: rng.IntN(10), Smile: rng.IntN(10)}
		sk := wantPlacement.SkillsAt(levels)
		wantAt := Best(positions, stage, sk)
		if rescored := scoring.Score(unskilled.Items, stage, sk); rescored > wantAt.Score {
			wantAt = unskilled
			wantAt.Score = rescored
		}
		at, p := space.BestPlacedAt(stage, levels)
		if p != wantPlacement || at.Score != wantAt.Score || !slices.EqualFunc(at.Items, wantAt.Items, sameItem) {
			t.Fatalf("trial %d: at %+v placed %+v for %d, want %+v for %d", trial, levels, p, at.Score, wantPlacement, wantAt.Score)
		}
		if at.Score < unskilled.Score {
			t.Fatalf("trial %d: at %+v %d is below %d without skills", trial, levels, at.Score, unskilled.Score)
		}
		if from, fp := space.BestPlacedFromAt(stage, unskilled, levels); fp != p || from.Score != at.Score {
			t.Fatalf("trial %d: from the unskilled result at %+v %+v %d, from scratch %+v %d", trial, levels, fp, from.Score, p, at.Score)
		}
	}
	if len(placements) < 10 {
		t.Errorf("the trials placed the skills only %d ways", len(placements))
	}
}

func bruteForce(positions []Position, st scoring.Stage, sk scoring.Skills) int {
	best := 0
	var walk func(i int, chosen []scoring.Item)
	walk = func(i int, chosen []scoring.Item) {
		if i == len(positions) {
			if wearable(chosen) {
				best = max(best, scoring.Score(chosen, st, sk))
			}
			return
		}
		walk(i+1, chosen)
		for _, it := range positions[i].Items {
			walk(i+1, append(slices.Clone(chosen), it))
		}
	}
	walk(0, nil)
	return best
}

func wearable(outfit []scoring.Item) bool {
	var dress, split bool
	for _, it := range outfit {
		switch it.Slot {
		case scoring.Dress:
			dress = true
		case scoring.Top, scoring.Bottom:
			split = true
		}
	}
	return !(dress && split)
}
