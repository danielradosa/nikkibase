package worth

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
)

func plainStage(weights ...float64) scoring.Stage {
	st := scoring.Stage{Attrs: [5]int8{0, 2, 4, 6, 8}}
	copy(st.Weights[:], weights)
	return st
}

func statItem(id int, slot scoring.Slot, stats ...int) scoring.Item {
	it := scoring.Item{ID: id, Slot: slot, Attrs: [5]int8{0, 2, 4, 6, 8}}
	copy(it.Stats[:], stats)
	return it
}

func TestRequiredMemberIsRepricedAfterAPickInALockedPlace(t *testing.T) {
	fx := &fixture{
		positions: []optimizer.Position{
			{Items: []scoring.Item{statItem(100, scoring.Hair, 100), statItem(101, scoring.Hair, 90)}},
			{Items: []scoring.Item{statItem(200, scoring.Shoes, 10), statItem(201, scoring.Shoes, 500)}},
		},
		posOf: map[int]int{100: 0, 101: 0, 200: 1, 201: 1},
		owned: map[int]bool{100: true, 200: true},
	}
	st := plainStage(1)
	fx.versions = []Version{
		{Key: "Story/1", Mode: "Story", Stage: st, Require: [][]int{{200, 101}}, Ideal: 1000},
		{Key: "Story/2", Mode: "Story", Stage: st, Ideal: 1000},
	}
	for _, skills := range []bool{false, true} {
		if skills {
			fx.settings = Settings{Skills: true, Levels: scoring.MaxLevels}
		}
		rows := fx.session().Rank(Filter{}, 5).Rows
		want := fx.search(fx.with(201, 101), fx.versions[0]).Score - fx.search(fx.with(201), fx.versions[0]).Score
		if want <= 0 {
			t.Fatalf("skills %v: the engine gains %d with 101 after 201, so the fixture checks nothing", skills, want)
		}
		if len(rows) != 2 || !slices.Equal(rows[0].Items, []int{201}) || !slices.Equal(rows[1].Items, []int{101}) {
			t.Fatalf("skills %v: rows %+v, want 201 and then 101", skills, rows)
		}
		if got := rows[1].Best; got.Key != "Story/1" || got.Points != want || rows[1].Stages != 1 {
			t.Errorf("skills %v: 101 gains %+v after 201, and the engine gains %d on Story/1", skills, got, want)
		}
	}
}

func TestRequiredMemberIsRepricedUnderItsOwnSkills(t *testing.T) {
	fx := &fixture{
		positions: []optimizer.Position{
			{Items: []scoring.Item{statItem(100, scoring.Hair, 30), statItem(101, scoring.Hair, 0, 29, 100)}},
			{Items: []scoring.Item{statItem(200, scoring.Dress, 60, 50)}},
			{Items: []scoring.Item{statItem(300, scoring.Shoes, 0, 60)}},
		},
		posOf: map[int]int{100: 0, 101: 0, 200: 1, 300: 2},
		owned: map[int]bool{100: true, 200: true},
	}
	fx.versions = []Version{
		{Key: "Story/1", Mode: "Story", Stage: plainStage(0, 0, 10), Ideal: 10000},
		{Key: "Story/2", Mode: "Story", Stage: plainStage(10, 10), Require: [][]int{{100, 300}}, Ideal: 10000},
	}
	for _, levels := range []scoring.Levels{scoring.MaxLevels, {Charming: 4, Smile: 6}} {
		fx.settings = Settings{Skills: true, Levels: levels}
		v := fx.versions[1]
		before := fx.search(fx.with(300), v).Score - fx.search(fx.owned2(), v).Score
		want := fx.search(fx.with(101, 300), v).Score - fx.search(fx.with(101), v).Score
		if want == before {
			t.Fatalf("%+v: 101 leaves the gain of 300 at %d, so the fixture checks nothing", levels, want)
		}
		rows := fx.session().Rank(Filter{}, 2).Rows
		if len(rows) != 2 || rows[0].Items[0] != 101 || rows[1].Items[0] != 300 {
			t.Fatalf("%+v: rows %+v, want 101 and then 300", levels, rows)
		}
		if got := rows[1].Best; got.Key != "Story/2" || got.Points != want {
			t.Errorf("%+v: 300 gains %+v after 101, and the engine gains %d on Story/2", levels, got, want)
		}
	}
}

func TestEveryPickReachesPlacementsFoundLater(t *testing.T) {
	fx := &fixture{
		positions: []optimizer.Position{
			{Items: []scoring.Item{statItem(100, scoring.Hair, 30), statItem(101, scoring.Hair, 0, 29, 100)}},
			{Items: []scoring.Item{statItem(200, scoring.Dress, 60, 50)}},
			{Items: []scoring.Item{statItem(300, scoring.Coat, 0, 20)}},
			{Items: []scoring.Item{statItem(400, scoring.Shoes, 0, 30)}},
		},
		posOf: map[int]int{100: 0, 101: 0, 200: 1, 300: 2, 400: 3},
		owned: map[int]bool{100: true, 200: true},
	}
	fx.versions = []Version{
		{Key: "Story/1", Mode: "Story", Stage: plainStage(0, 0, 10), Ideal: 10000},
		{Key: "Story/2", Mode: "Story", Stage: plainStage(10, 10), Ideal: 10000},
	}
	for _, levels := range []scoring.Levels{scoring.MaxLevels, {Charming: 4, Smile: 6}} {
		fx.settings = Settings{Skills: true, Levels: levels}
		rows := fx.session().Rank(Filter{}, 3).Rows
		if len(rows) != 3 || rows[0].Items[0] != 101 || rows[1].Items[0] != 400 || rows[2].Items[0] != 300 {
			t.Fatalf("%+v: rows %+v, want 101, 400 and 300", levels, rows)
		}
		want := fx.search(fx.with(101, 400, 300), fx.versions[1]).Score - fx.search(fx.with(101, 400), fx.versions[1]).Score
		if got := rows[2].Best; got.Key != "Story/2" || got.Points != want {
			t.Errorf("%+v: 300 gains %+v after 101 and 400, and the engine gains %d on Story/2", levels, got, want)
		}
	}
}

func TestTiedItemEarlierInOrderMovesTheSkills(t *testing.T) {
	st := plainStage(10, 10)
	cases := []struct {
		name  string
		hair  []scoring.Item
		owned []int
		cand  int
		gain  bool
	}{
		{"earlier than the owned hair", []scoring.Item{
			statItem(101, scoring.Hair, 100, 0), statItem(100, scoring.Hair, 0, 100),
		}, []int{100}, 101, true},
		{"later than a covered owned hair", []scoring.Item{
			statItem(100, scoring.Hair, 0, 100), statItem(101, scoring.Hair, 100, 0),
			{ID: 102, Slot: scoring.Hair, Attrs: [5]int8{0, 2, 4, 7, 8}, Stats: [5]int{0, 100, 0, 30, 0}},
		}, []int{100, 102}, 101, false},
	}
	for _, c := range cases {
		fx := &fixture{
			positions: []optimizer.Position{
				{Items: c.hair},
				{Items: []scoring.Item{statItem(200, scoring.Dress, 50, 45)}},
			},
			posOf: map[int]int{200: 1},
			owned: map[int]bool{200: true},
		}
		for _, it := range c.hair {
			fx.posOf[it.ID] = 0
		}
		for _, id := range c.owned {
			fx.owned[id] = true
		}
		fx.versions = []Version{{Key: "Story/1", Mode: "Story", Stage: st, Ideal: 4000}}
		for _, levels := range []scoring.Levels{scoring.MaxLevels, {Charming: 4, Smile: 6}} {
			fx.settings = Settings{Skills: true, Levels: levels}
			want := fx.search(fx.with(c.cand), fx.versions[0]).Score - fx.search(fx.owned2(), fx.versions[0]).Score
			if (want > 0) != c.gain {
				t.Fatalf("%s at %+v: the engine gains %d, so the fixture checks nothing", c.name, levels, want)
			}
			s := fx.session()
			if got := s.gainOf(0, c.cand); got != want {
				t.Errorf("%s at %+v: %d gains %d, and the engine gains %d", c.name, levels, c.cand, got, want)
			}
			rows := s.Rank(Filter{}, 5).Rows
			if c.gain && (len(rows) != 1 || rows[0].Items[0] != c.cand || rows[0].Best.Points != want) {
				t.Errorf("%s at %+v: rows %+v, want %d gaining %d", c.name, levels, rows, c.cand, want)
			}
			if !c.gain && len(rows) != 0 {
				t.Errorf("%s at %+v: rows %+v, want none", c.name, levels, rows)
			}
		}
	}
}

var grades = []int{10, 20, 30, 40}

func gradedItem(rng *rand.Rand, id int, slot scoring.Slot) scoring.Item {
	it := scoring.Item{ID: id, Slot: slot}
	for p := range 5 {
		it.Attrs[p] = int8(2*p + rng.IntN(2))
		if rng.IntN(3) > 0 {
			it.Stats[p] = grades[rng.IntN(len(grades))]
		}
	}
	if rng.IntN(4) == 0 {
		it.Tags = []int{1 + rng.IntN(2)}
	}
	if slot == scoring.Spirit {
		it.FlatBonus = 10 * rng.IntN(3)
	}
	return it
}

func gradedStage(rng *rand.Rand) scoring.Stage {
	var sliders [5]int
	for p := range sliders {
		sliders[p] = []int{-100, -50, 0, 50, 100}[rng.IntN(5)]
	}
	return scoring.CustomStage(sliders, map[int]int{1: 100 * rng.IntN(3), 2: 50 * rng.IntN(3)})
}

func gradedFixture(rng *rand.Rand, layout []place, most, stages int, settings Settings) *fixture {
	fx := &fixture{posOf: map[int]int{}, owned: map[int]bool{}, settings: settings}
	var ids []int
	for at, pl := range layout {
		var items []scoring.Item
		for n := range 1 + rng.IntN(most) {
			id := (at+1)*100 + n
			items = append(items, gradedItem(rng, id, pl.slot))
			fx.posOf[id] = at
			ids = append(ids, id)
			if rng.IntN(2) == 0 {
				fx.owned[id] = true
			}
		}
		fx.positions = append(fx.positions, optimizer.Position{Items: items, Group: pl.group, Exclusive: pl.whole})
	}
	modes := []string{"Story", "Commission", "Arena"}
	for n := range stages {
		v := Version{Key: modes[n%len(modes)] + "/" + string(rune('a'+n)), Mode: modes[n%len(modes)], Stage: gradedStage(rng)}
		if rng.IntN(2) == 0 {
			for range 1 + rng.IntN(2) {
				set := []int{ids[rng.IntN(len(ids))]}
				for rng.IntN(2) == 0 && len(set) < 3 {
					set = append(set, ids[rng.IntN(len(ids))])
				}
				v.Require = append(v.Require, slices.Compact(set))
			}
		}
		v.Ideal = fx.search(func(int) bool { return true }, v).Score
		fx.versions = append(fx.versions, v)
	}
	return fx
}

func requireAnAccessory(rng *rand.Rand, fx *fixture) {
	var owned []int
	for _, p := range fx.positions {
		for _, it := range p.Items {
			if it.Slot == scoring.Accessory && fx.owned[it.ID] {
				owned = append(owned, it.ID)
			}
		}
	}
	for k := range fx.versions {
		if len(fx.versions[k].Require) == 0 && len(owned) > 0 {
			fx.versions[k].Require = [][]int{{owned[rng.IntN(len(owned))]}}
		}
	}
}

func scoringIDs(items []scoring.Item, st scoring.Stage) []int {
	var ids []int
	for _, it := range items {
		if s, f := scoring.Contribution(it, st, nil); s+f != 0 {
			ids = append(ids, it.ID)
		}
	}
	slices.Sort(ids)
	return ids
}

func TestWornOutfitsMatchTheEngineOnTies(t *testing.T) {
	rng := rand.New(rand.NewPCG(99, 1))
	checked := 0
	for trial := range 300 {
		fx := gradedFixture(rng, manyAccessories(), 3, 4, Settings{Skills: true, Levels: scoring.MaxLevels})
		requireAnAccessory(rng, fx)
		s := fx.session()
		for vi, v := range fx.versions {
			st := s.stages[vi]
			if st.failing {
				continue
			}
			sc := s.scorer(st, &s.versions[vi])
			for _, id := range append([]int{0}, fx.unowned()...) {
				var c change
				if id != 0 {
					if i := s.l.index[id]; st.isMember(i) {
						continue
					} else {
						c, _, _, _ = sc.changeOf([]int32{i})
					}
				}
				engine := optimizer.Require(fx.pool(fx.with(id)), fx.posOf, v.Require).Best(v.Stage, nil).Items
				want, place := scoringIDs(engine, v.Stage), s.effective(scoring.Place(engine, v.Stage))
				if got := scoringIDs(sc.outfit(&c), v.Stage); !slices.Equal(got, want) {
					t.Fatalf("trial %d %s with %d: the model wears %v, the engine %v", trial, v.Key, id, got, want)
				}
				if got := sc.placeWith(&c); got != place {
					t.Fatalf("trial %d %s with %d: the model places skills on %+v, the engine on %+v", trial, v.Key, id, got, place)
				}
				checked++
			}
			sc.release()
		}
	}
	if checked < 25000 {
		t.Errorf("checked %d outfits", checked)
	}
}

func (s *Session) rankerFor(f Filter) *ranker {
	var vs []int
	for vi := range s.done {
		if v := &s.versions[vi]; f.admits(v) && !s.stages[vi].failing && v.Ideal > 0 {
			vs = append(vs, vi)
		}
	}
	return s.ranker(vs, f)
}

type stateTally struct {
	checks, gained, pairs, steps int
}

func checkState(fx *fixture, r *ranker, picks []int, n *stateTally) error {
	l := r.s.l
	for j, vi := range r.vs {
		v := fx.versions[vi]
		st := r.cur[j]
		base := fx.search(fx.with(picks...), v).Score
		if d := st.base - base; d < -1 || d > 1 {
			return fmt.Errorf("%s starts from %d, and the engine from %d", v.Key, st.base, base)
		}
		for _, id := range fx.unowned() {
			if slices.Contains(picks, id) {
				continue
			}
			i := l.index[id]
			want := fx.search(fx.with(append(slices.Clone(picks), id)...), v).Score - base
			got := int(st.points(i))
			n.checks++
			if want != 0 {
				n.gained++
			}
			if _, live := slices.BinarySearch(st.live, i); !live && want != 0 {
				return fmt.Errorf("%s: item %d is skipped, and the engine gains %d with it", v.Key, id, want)
			}
			if d := got - want; d < -1 || d > 1 {
				return fmt.Errorf("%s: item %d gains %d, and the engine gains %d", v.Key, id, got, want)
			}
		}
		for k := range r.pairs {
			b := &r.pairs[k]
			if b.dead {
				continue
			}
			top, bottom := l.items[b.items[0]].ID, l.items[b.items[1]].ID
			want := fx.search(fx.with(append(slices.Clone(picks), top, bottom)...), v).Score - base
			n.pairs++
			if d := int(b.pts[j]) - want; d < -1 || d > 1 {
				return fmt.Errorf("%s: the pair %d+%d gains %d, and the engine gains %d", v.Key, top, bottom, b.pts[j], want)
			}
		}
	}
	return nil
}

func TestGreedyStateMatchesTheEngineAtEveryStep(t *testing.T) {
	rng := rand.New(rand.NewPCG(23, 5))
	var n stateTally
	for trial := range 240 {
		var settings Settings
		switch trial % 3 {
		case 1:
			settings = Settings{Skills: true, Levels: scoring.MaxLevels}
		case 2:
			settings = Settings{Skills: true, Levels: scoring.Levels{Charming: rng.IntN(10), Smile: rng.IntN(10)}}
		}
		fx := gradedFixture(rng, places, 4, 5, settings)
		s := fx.session()
		r := s.rankerFor(Filter{})
		var picks []int
		for step := 0; ; step++ {
			if err := checkState(fx, r, picks, &n); err != nil {
				t.Fatalf("trial %d %+v step %d (picked %v): %v", trial, settings, step, picks, err)
			}
			row, ok := r.next()
			if !ok || step == 8 {
				break
			}
			picks = append(picks, row.Items...)
			n.steps++
		}
	}
	if n.checks < 100000 || n.gained < 25000 || n.pairs < 900 || n.steps < 1300 {
		t.Errorf("%d item checks (%d with a gain), %d pair checks, %d steps", n.checks, n.gained, n.pairs, n.steps)
	}
}
