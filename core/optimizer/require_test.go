package optimizer

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

func indexOf(ps []Position) map[int]int {
	at := map[int]int{}
	for i, p := range ps {
		for _, it := range p.Items {
			at[it.ID] = i
		}
	}
	return at
}

func required(ps []Position, sets ...[]int) Result {
	return Require(ps, indexOf(ps), sets).Best(livelyStage, nil)
}

func satisfies(outfit []scoring.Item, set []int) bool {
	for _, it := range outfit {
		if slices.Contains(set, it.ID) {
			return true
		}
	}
	return false
}

func torso(dress, top, bottom int) []Position {
	return []Position{
		{Items: []scoring.Item{item(1, scoring.Dress, dress), item(4, scoring.Dress, dress/2)}},
		{Items: []scoring.Item{item(2, scoring.Top, top), item(5, scoring.Top, top/2)}},
		{Items: []scoring.Item{item(3, scoring.Bottom, bottom), item(6, scoring.Bottom, bottom/2)}},
		{Items: []scoring.Item{item(7, scoring.Hair, 40)}},
	}
}

func TestRequiredDressDisplacesSeparates(t *testing.T) {
	ps := torso(100, 90, 90)
	if got := Best(ps, livelyStage, nil); !slices.Equal(ids(got.Items), []int{2, 3, 7}) {
		t.Fatalf("without the rule wore %v, want the separates, or the rule proves nothing", ids(got.Items))
	}
	got := required(ps, []int{4})
	if !slices.Equal(ids(got.Items), []int{4, 7}) {
		t.Errorf("wore %v, want the required dress 4 and nothing on top or bottom", ids(got.Items))
	}
	if want := scoring.Score(got.Items, livelyStage, nil); got.Score != want {
		t.Errorf("reported %d for an outfit worth %d", got.Score, want)
	}
}

func TestRequiredSeparatesDisplaceTheDress(t *testing.T) {
	ps := torso(400, 90, 90)
	got := required(ps, []int{5})
	if !slices.Equal(ids(got.Items), []int{3, 5, 7}) {
		t.Errorf("wore %v, want the required top 5 with the best bottom and no dress", ids(got.Items))
	}
	got = required(ps, []int{6})
	if !slices.Equal(ids(got.Items), []int{2, 6, 7}) {
		t.Errorf("wore %v, want the required bottom 6 with the best top and no dress", ids(got.Items))
	}
}

func TestRequiredWorthlessTopIsStillWorn(t *testing.T) {
	ps := []Position{
		{Items: []scoring.Item{item(1, scoring.Dress, 100)}},
		{Items: []scoring.Item{item(2, scoring.Top, 0)}},
	}
	got := required(ps, []int{2})
	if !slices.Equal(ids(got.Items), []int{2}) {
		t.Errorf("wore %v, want the required top even though it scores nothing", ids(got.Items))
	}
}

func TestRequiredAccessoryIsWornAtALoss(t *testing.T) {
	ps := []Position{
		{Items: []scoring.Item{item(1, scoring.Accessory, 100)}},
		{Items: []scoring.Item{item(2, scoring.Accessory, 100)}},
		{Items: []scoring.Item{item(3, scoring.Accessory, 100)}},
		{Items: []scoring.Item{item(9, scoring.Accessory, 1), item(10, scoring.Accessory, 2)}},
	}
	free := Best(ps, livelyStage, nil)
	if slices.Contains(ids(free.Items), 9) {
		t.Fatalf("wore 9 without the rule, so the rule proves nothing")
	}
	got := required(ps, []int{9})
	if !slices.Equal(ids(got.Items), []int{1, 2, 3, 9}) {
		t.Errorf("wore %v, want the three strong accessories and the required 9", ids(got.Items))
	}
	if got.Score >= free.Score {
		t.Errorf("the required item scores %d, not below %d without it, so it is no loss", got.Score, free.Score)
	}
}

func TestOneOfAcrossTwoPlacesTakesTheBetterPlace(t *testing.T) {
	const handheld = 1
	hat := Position{Items: []scoring.Item{item(1, scoring.Accessory, 5), item(2, scoring.Accessory, 30)}}
	staff := Position{Items: []scoring.Item{item(3, scoring.Accessory, 60)}, Group: handheld, Exclusive: true}
	right := Position{Items: []scoring.Item{item(4, scoring.Accessory, 40)}, Group: handheld}
	left := Position{Items: []scoring.Item{item(5, scoring.Accessory, 40)}, Group: handheld}
	ps := []Position{hat, staff, right, left}

	if free := Best(ps, livelyStage, nil); !slices.Equal(ids(free.Items), []int{2, 4, 5}) {
		t.Fatalf("without the rule wore %v, want the better hat and the pair, or the rule proves nothing", ids(free.Items))
	}
	got := required(ps, []int{1, 3})
	if !slices.Equal(ids(got.Items), []int{2, 3}) {
		t.Errorf("wore %v, want the better hat 2 with the staff 3, which beats the weak hat 1 with the pair", ids(got.Items))
	}
	only := func(sets ...[]int) int { return required(ps, sets...).Score }
	if want := max(only([]int{1}), only([]int{3})); got.Score != want {
		t.Errorf("scored %d, and the better of the two places scores %d", got.Score, want)
	}
	if !legal(ps, got.Items) {
		t.Errorf("wore %v, which the game forbids", ids(got.Items))
	}

	ps[1].Items = []scoring.Item{item(3, scoring.Accessory, 20)}
	got = required(ps, []int{1, 3})
	if want := max(only([]int{1}), only([]int{3})); got.Score != want || !satisfies(got.Items, []int{1, 3}) {
		t.Errorf("wore %v for %d, want one of 1 and 3 for %d", ids(got.Items), got.Score, want)
	}
}

func TestSharedDressMeetsTwoSets(t *testing.T) {
	for _, dress := range []int{60, 400} {
		ps := torso(dress, 90, 90)
		got := required(ps, []int{5, 4}, []int{6, 4})
		for _, set := range [][]int{{5, 4}, {6, 4}} {
			if !satisfies(got.Items, set) {
				t.Errorf("dress %d: wore %v, which meets none of %v", dress, ids(got.Items), set)
			}
		}
		if !wearable(got.Items) {
			t.Errorf("dress %d: wore %v, a dress with separates", dress, ids(got.Items))
		}
		want := max(required(ps, []int{4}).Score, required(ps, []int{5}, []int{6}).Score)
		if got.Score != want {
			t.Errorf("dress %d: scored %d, and the better of the dress and the two separates scores %d", dress, got.Score, want)
		}
	}
}

func TestMissingSetIsLeftUnmet(t *testing.T) {
	ps := torso(100, 90, 90)
	sets := [][]int{{99}, {4}, {98, 97}}
	got := Require(ps, indexOf(ps), sets).Best(livelyStage, nil)
	if !slices.Equal(ids(got.Items), []int{4, 7}) {
		t.Errorf("wore %v, want the required dress 4 despite the sets nobody owns", ids(got.Items))
	}
	if unmet := Unmet(got.Items, sets); !reflect.DeepEqual(unmet, [][]int{{99}, {98, 97}}) {
		t.Errorf("unmet = %v, want the two sets the pool has nothing for", unmet)
	}

	excluded := Exclude(ps, 4)
	got = Require(excluded, indexOf(ps), [][]int{{4}}).Best(livelyStage, nil)
	if unmet := Unmet(got.Items, [][]int{{4}}); len(unmet) != 1 {
		t.Errorf("an excluded item still counts as held: wore %v", ids(got.Items))
	}
}

func TestSetsThatCannotBeWornTogetherKeepTheFirst(t *testing.T) {
	ps := torso(100, 90, 90)
	sets := [][]int{{1}, {2}}
	got := Require(ps, indexOf(ps), sets).Best(livelyStage, nil)
	if !slices.Equal(ids(got.Items), []int{1, 7}) {
		t.Errorf("wore %v, want the dress the first set asks for", ids(got.Items))
	}
	if unmet := Unmet(got.Items, sets); !reflect.DeepEqual(unmet, [][]int{{2}}) {
		t.Errorf("unmet = %v, want the top that cannot go with the dress", unmet)
	}

	const handheld = 1
	hands := []Position{
		{Items: []scoring.Item{item(1, scoring.Accessory, 10)}, Group: handheld},
		{Items: []scoring.Item{item(2, scoring.Accessory, 10)}, Group: handheld, Exclusive: true},
	}
	got = Require(hands, indexOf(hands), [][]int{{2}, {1}}).Best(livelyStage, nil)
	if !slices.Equal(ids(got.Items), []int{2}) || !legal(hands, got.Items) {
		t.Errorf("wore %v, want the both-hands item alone", ids(got.Items))
	}
}

func TestWithoutSetsRequireIsBest(t *testing.T) {
	ps := torso(100, 90, 90)
	for _, sets := range [][][]int{nil, {}} {
		space := Require(ps, indexOf(ps), sets)
		if got, want := space.Best(livelyStage, nil), Best(ps, livelyStage, nil); !reflect.DeepEqual(got, want) {
			t.Errorf("sets %v: %+v, and Best gives %+v", sets, got, want)
		}
		got, placement := space.BestPlaced(livelyStage)
		want, wantPlacement := BestPlaced(ps, livelyStage)
		if placement != wantPlacement || !reflect.DeepEqual(got, want) {
			t.Errorf("sets %v: placed %+v %+v, and BestPlaced gives %+v %+v", sets, placement, got, wantPlacement, want)
		}
	}
}

func TestRequireLeavesThePositionsAlone(t *testing.T) {
	const handheld = 1
	ps := append(torso(100, 90, 90),
		Position{Items: []scoring.Item{item(20, scoring.Accessory, 10)}, Group: handheld},
		Position{Items: []scoring.Item{item(21, scoring.Accessory, 10)}, Group: handheld, Exclusive: true})
	before := make([]Position, len(ps))
	for i, p := range ps {
		before[i] = p
		before[i].Items = slices.Clone(p.Items)
	}
	space := Require(ps, indexOf(ps), [][]int{{4}, {20, 21}, {99}})
	space.Best(livelyStage, nil)
	space.BestPlaced(livelyStage)
	if !reflect.DeepEqual(ps, before) {
		t.Errorf("searching changed the caller's positions")
	}
	for _, branch := range space.branches {
		if len(branch[1].Items) != 0 || len(branch[2].Items) != 0 || !branch[0].Required {
			t.Errorf("a branch keeps separates beside a required dress: %+v", branch[:3])
		}
		if branch[4].Group != handheld || !branch[5].Exclusive {
			t.Errorf("a branch lost the hand-held group: %+v", branch[4:])
		}
	}
	if len(space.branches) != 2 {
		t.Errorf("%d branches, want one per hand-held place", len(space.branches))
	}
}

func TestRequiredMatchesExhaustiveSearch(t *testing.T) {
	rng := rand.New(rand.NewPCG(24, 9))
	const handheld = 1
	slots := []struct {
		slot  scoring.Slot
		group uint8
		whole bool
	}{
		{scoring.Dress, 0, false}, {scoring.Top, 0, false}, {scoring.Bottom, 0, false},
		{scoring.Hair, 0, false},
		{scoring.Accessory, handheld, false}, {scoring.Accessory, handheld, false}, {scoring.Accessory, handheld, true},
		{scoring.Accessory, 0, false}, {scoring.Accessory, 0, false}, {scoring.Accessory, 0, false},
	}
	var checked, oneOf, missing, costly int
	for trial := range 120 {
		st := scoring.CustomStage([5]int{rng.IntN(101), rng.IntN(101), rng.IntN(101), 0, 0},
			map[int]int{1: rng.IntN(500)})
		var ps []Position
		var all []int
		id := 0
		for _, s := range slots {
			var items []scoring.Item
			for range 1 + rng.IntN(2) {
				id++
				it := item(id, s.slot, rng.IntN(120))
				it.Attrs[0] = int8(rng.IntN(2))
				it.Stats[0] = rng.IntN(120)
				if rng.IntN(3) == 0 {
					it.Tags = []int{1}
				}
				items = append(items, it)
				all = append(all, id)
			}
			ps = append(ps, Position{Items: items, Group: s.group, Exclusive: s.whole})
		}

		var sets [][]int
		for range 1 + rng.IntN(3) {
			set := []int{all[rng.IntN(len(all))]}
			if rng.IntN(2) == 0 {
				set = append(set, all[rng.IntN(len(all))])
			}
			if rng.IntN(6) == 0 {
				set = []int{1000 + rng.IntN(10)}
			}
			sets = append(sets, slices.Compact(set))
		}

		space := Require(ps, indexOf(ps), sets)
		got := space.Best(st, nil)
		if !legal(ps, got.Items) || !wearable(got.Items) {
			t.Fatalf("trial %d: wore %v, which the game forbids", trial, ids(got.Items))
		}
		if want := scoring.Score(got.Items, st, nil); got.Score != want {
			t.Fatalf("trial %d: reported %d for an outfit worth %d", trial, got.Score, want)
		}

		var held [][]int
		for _, set := range sets {
			if slices.ContainsFunc(set, func(id int) bool { return id < 1000 }) {
				held = append(held, set)
			} else {
				missing++
			}
		}
		want, feasible := exhaustiveWith(ps, st, held)
		if !feasible {
			continue
		}
		checked++
		for _, set := range sets {
			if len(set) > 1 && indexOf(ps)[set[0]] != indexOf(ps)[set[1]] {
				oneOf++
			}
		}
		if Best(ps, st, nil).Score > want {
			costly++
		}
		if unmet := Unmet(got.Items, held); len(unmet) > 0 {
			t.Fatalf("trial %d: wore %v, which meets none of %v", trial, ids(got.Items), unmet)
		}
		if got.Score != want {
			t.Fatalf("trial %d: scored %d, and the best outfit meeting %v scores %d", trial, got.Score, held, want)
		}
		if auto, _ := space.BestPlaced(st); len(Unmet(auto.Items, held)) > 0 {
			t.Fatalf("trial %d: with skills wore %v, which misses %v", trial, ids(auto.Items), Unmet(auto.Items, held))
		}
	}
	if checked < 60 || oneOf < 20 || missing < 10 || costly < 30 {
		t.Errorf("the trials checked %d outfits, %d sets across two places, %d sets nobody holds, %d rules that cost points",
			checked, oneOf, missing, costly)
	}
}

func exhaustiveWith(ps []Position, st scoring.Stage, sets [][]int) (int, bool) {
	best := -1
	var walk func(i int, chosen []scoring.Item)
	walk = func(i int, chosen []scoring.Item) {
		if i == len(ps) {
			if legal(ps, chosen) && wearable(chosen) && len(Unmet(chosen, sets)) == 0 {
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
	return best, best >= 0
}
