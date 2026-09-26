package optimizer

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

func acc(id int, stat int, tags ...int) scoring.Item {
	return scoring.Item{ID: id, Slot: scoring.Accessory,
		Attrs: [5]int8{scoring.Simple, scoring.Lively, scoring.Cute, scoring.Sexy, scoring.Cool},
		Stats: [5]int{stat, stat, stat, stat, stat}, Tags: tags}
}

func stage() scoring.Stage {
	return scoring.Stage{
		Attrs:   [5]int8{scoring.Simple, scoring.Lively, scoring.Cute, scoring.Sexy, scoring.Cool},
		Weights: [5]float64{2, 2, 2, 2, 2},
	}
}

func count(items []scoring.Item, slot scoring.Slot) int {
	n := 0
	for _, it := range items {
		if it.Slot == slot {
			n++
		}
	}
	return n
}

func has(items []scoring.Item, id int) bool {
	for _, it := range items {
		if it.ID == id {
			return true
		}
	}
	return false
}

func TestHandheldGroupIsExclusive(t *testing.T) {
	const handheld = 1
	right := Position{Items: []scoring.Item{acc(1, 10)}, Group: handheld}
	left := Position{Items: []scoring.Item{acc(2, 10)}, Group: handheld}
	both := Position{Items: []scoring.Item{acc(3, 100)}, Group: handheld, Exclusive: true}

	got := Best([]Position{right, left, both}, stage(), nil)
	if count(got.Items, scoring.Accessory) != 1 || !has(got.Items, 3) {
		t.Errorf("wore %v, want the both-hands item alone", worn(got.Items))
	}

	right.Items = []scoring.Item{acc(1, 200)}
	got = Best([]Position{right, left, both}, stage(), nil)
	if has(got.Items, 3) {
		t.Errorf("wore %v, want both hands free for the pair", worn(got.Items))
	}
	if !has(got.Items, 1) || !has(got.Items, 2) {
		t.Errorf("wore %v, want the left and right pair", worn(got.Items))
	}
}

func TestUngroupedAccessoriesAreUnaffected(t *testing.T) {
	ps := []Position{
		{Items: []scoring.Item{acc(1, 10)}},
		{Items: []scoring.Item{acc(2, 10)}},
		{Items: []scoring.Item{acc(3, 10)}},
	}
	if n := count(Best(ps, stage(), nil).Items, scoring.Accessory); n != 3 {
		t.Errorf("wore %d accessories, want 3", n)
	}
}

func TestNeverExceedsSlotLimits(t *testing.T) {
	var ps []Position
	for i := range 40 {
		ps = append(ps, Position{Items: []scoring.Item{acc(i+1, 10+i)}})
	}
	for i := range 4 {
		ps = append(ps, Position{Items: []scoring.Item{{ID: 500 + i, Slot: scoring.Hosiery,
			Attrs: [5]int8{scoring.Simple, scoring.Lively, scoring.Cute, scoring.Sexy, scoring.Cool},
			Stats: [5]int{5, 5, 5, 5, 5}}}})
	}
	got := Best(ps, stage(), nil)
	if n := count(got.Items, scoring.Accessory); n > scoring.SlotLimit(scoring.Accessory) {
		t.Errorf("wore %d accessories, and the game allows %d", n, scoring.SlotLimit(scoring.Accessory))
	}
	if n := count(got.Items, scoring.Hosiery); n > scoring.SlotLimit(scoring.Hosiery) {
		t.Errorf("wore %d hosiery, and the game allows %d", n, scoring.SlotLimit(scoring.Hosiery))
	}
	if !has(got.Items, 40) {
		t.Errorf("dropped the strongest accessory: wore %v", worn(got.Items))
	}
	if want := scoring.Score(got.Items, stage(), nil); got.Score != want {
		t.Errorf("reported %d for an outfit worth %d", got.Score, want)
	}
}

func TestNoItemWornTwice(t *testing.T) {
	same := acc(7, 50)
	got := Best([]Position{{Items: []scoring.Item{same}}, {Items: []scoring.Item{same}}}, stage(), nil)
	seen := map[int]bool{}
	for _, it := range got.Items {
		if seen[it.ID] {
			t.Errorf("item %d is worn twice: %v", it.ID, worn(got.Items))
		}
		seen[it.ID] = true
	}
}

func worn(items []scoring.Item) []int {
	out := make([]int, 0, len(items))
	for _, it := range items {
		out = append(out, it.ID)
	}
	return out
}

func TestGroupsMatchBruteForce(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 11))
	const handheld = 1
	for trial := range 150 {
		st := scoring.CustomStage([5]int{rng.IntN(101), rng.IntN(101), rng.IntN(101), rng.IntN(101), 0},
			map[int]int{1: rng.IntN(3000)})
		var ps []Position
		id := 0
		add := func(group uint8, whole bool) {
			var items []scoring.Item
			for range 1 + rng.IntN(2) {
				id++
				it := scoring.Item{ID: id, Slot: scoring.Accessory}
				for p := range 5 {
					it.Attrs[p] = int8(p*2 + rng.IntN(2))
					it.Stats[p] = rng.IntN(60)
				}
				if rng.IntN(3) == 0 {
					it.Tags = []int{1}
				}
				items = append(items, it)
			}
			ps = append(ps, Position{Items: items, Group: group, Exclusive: whole})
		}
		add(handheld, false)
		add(handheld, false)
		add(handheld, true)
		for range 3 + rng.IntN(4) {
			add(0, false)
		}

		got := Best(ps, st, nil)
		if want := exhaustive(ps, st); got.Score != want {
			t.Fatalf("trial %d: Best scored %d, exhaustive search found %d", trial, got.Score, want)
		}
		if !legal(ps, got.Items) {
			t.Fatalf("trial %d: Best returned an outfit the game forbids: %v", trial, worn(got.Items))
		}
	}
}

func exhaustive(ps []Position, st scoring.Stage) int {
	best := 0
	var walk func(i int, chosen []scoring.Item)
	walk = func(i int, chosen []scoring.Item) {
		if i == len(ps) {
			if legal(ps, chosen) {
				best = max(best, scoring.Score(chosen, st, nil))
			}
			return
		}
		walk(i+1, chosen)
		for _, it := range ps[i].Items {
			walk(i+1, append(slices.Clone(chosen), it))
		}
	}
	walk(0, nil)
	return best
}

func legal(ps []Position, outfit []scoring.Item) bool {
	posOf := map[int]int{}
	for i, p := range ps {
		for _, it := range p.Items {
			posOf[it.ID] = i
		}
	}
	filled := map[uint8][]int{}
	for _, it := range outfit {
		i := posOf[it.ID]
		if g := ps[i].Group; g != 0 {
			filled[g] = append(filled[g], i)
		}
	}
	for _, positions := range filled {
		if len(positions) < 2 {
			continue
		}
		for _, i := range positions {
			if ps[i].Exclusive {
				return false
			}
		}
	}
	return true
}

func TestExcludeKeepsGroups(t *testing.T) {
	const handheld = 1
	ps := []Position{
		{Items: []scoring.Item{acc(1, 10)}, Group: handheld},
		{Items: []scoring.Item{acc(2, 10)}, Group: handheld},
		{Items: []scoring.Item{acc(3, 100)}, Group: handheld, Exclusive: true},
		{Items: []scoring.Item{acc(4, 5)}},
	}
	got := Best(Exclude(ps, 4), stage(), nil)
	if !legal(ps, got.Items) {
		t.Errorf("after an exclusion, wore %v, which the game forbids", worn(got.Items))
	}
}

func TestRequiredInGroupIsKept(t *testing.T) {
	const handheld = 1
	ps := []Position{
		{Items: []scoring.Item{acc(1, 5)}, Group: handheld, Required: true},
		{Items: []scoring.Item{acc(2, 5)}, Group: handheld},
		{Items: []scoring.Item{acc(3, 500)}, Group: handheld, Exclusive: true},
	}
	got := Best(ps, stage(), nil)
	if !has(got.Items, 1) {
		t.Errorf("wore %v, and left out the required item", worn(got.Items))
	}
	if !legal(ps, got.Items) {
		t.Errorf("wore %v, which the game forbids", worn(got.Items))
	}
}

func TestMoreRequiredThanTheLimitStillDresses(t *testing.T) {
	limit := scoring.SlotLimit(scoring.Accessory)
	var ps []Position
	for i := range limit + 3 {
		ps = append(ps, Position{Items: []scoring.Item{acc(i+1, 10+i)}, Required: true})
	}
	got := Best(ps, stage(), nil)
	if len(got.Items) != limit {
		t.Fatalf("wore %d accessories, want the %d the game allows", len(got.Items), limit)
	}
	for id := 1; id <= 3; id++ {
		if has(got.Items, id) {
			t.Errorf("wore %d, one of the three least valuable, over a stronger required item", id)
		}
	}
}
