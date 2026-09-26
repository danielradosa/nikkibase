package worth

import (
	"math"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
)

const handheld = 1

type place struct {
	slot  scoring.Slot
	group uint8
	whole bool
}

var places = []place{
	{scoring.Hair, 0, false}, {scoring.Dress, 0, false}, {scoring.Top, 0, false}, {scoring.Bottom, 0, false},
	{scoring.Coat, 0, false}, {scoring.Hosiery, 0, false}, {scoring.Hosiery, 0, false}, {scoring.Shoes, 0, false},
	{scoring.Spirit, 0, false},
	{scoring.Accessory, handheld, false}, {scoring.Accessory, handheld, false}, {scoring.Accessory, handheld, true},
	{scoring.Accessory, 0, false}, {scoring.Accessory, 0, false}, {scoring.Accessory, 0, false},
	{scoring.Accessory, 0, false}, {scoring.Accessory, 0, false},
}

type fixture struct {
	positions []optimizer.Position
	posOf     map[int]int
	owned     map[int]bool
	versions  []Version
	settings  Settings
}

func (fx *fixture) own(id int32) bool { return fx.owned[int(id)] }

func randomItem(rng *rand.Rand, id int, slot scoring.Slot) scoring.Item {
	it := scoring.Item{ID: id, Slot: slot}
	for p := range 5 {
		it.Attrs[p] = int8(2*p + rng.IntN(2))
		it.Stats[p] = rng.IntN(150)
	}
	if rng.IntN(3) == 0 {
		it.Tags = []int{1 + rng.IntN(2)}
	}
	if slot == scoring.Spirit {
		it.FlatBonus = rng.IntN(300)
	}
	return it
}

func randomStage(rng *rand.Rand) scoring.Stage {
	var sliders [5]int
	for p := range sliders {
		sliders[p] = rng.IntN(201) - 100
	}
	return scoring.CustomStage(sliders, map[int]int{1: rng.IntN(900), 2: rng.IntN(400)})
}

func newFixture(rng *rand.Rand, layout []place, most int, stages int, skills bool) *fixture {
	fx := &fixture{posOf: map[int]int{}, owned: map[int]bool{}}
	var ids []int
	for at, pl := range layout {
		var items []scoring.Item
		for n := range 1 + rng.IntN(most) {
			id := (at+1)*100 + n
			items = append(items, randomItem(rng, id, pl.slot))
			fx.posOf[id] = at
			ids = append(ids, id)
			if rng.IntN(2) == 0 {
				fx.owned[id] = true
			}
		}
		fx.positions = append(fx.positions, optimizer.Position{Items: items, Group: pl.group, Exclusive: pl.whole})
	}
	if skills {
		fx.settings = Settings{Skills: true, Levels: scoring.Levels{Charming: rng.IntN(10), Smile: rng.IntN(10)}}
	}
	modes := []string{"Story", "Commission", "Arena"}
	for n := range stages {
		v := Version{Key: modes[n%len(modes)] + "/" + string(rune('a'+n)), Mode: modes[n%len(modes)], Stage: randomStage(rng)}
		if rng.IntN(3) == 0 {
			for range 1 + rng.IntN(2) {
				set := []int{ids[rng.IntN(len(ids))]}
				if rng.IntN(3) == 0 {
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

func (fx *fixture) pool(has func(id int) bool) []optimizer.Position {
	out := make([]optimizer.Position, len(fx.positions))
	for at, p := range fx.positions {
		q := p
		q.Items = nil
		for _, it := range p.Items {
			if has(it.ID) {
				q.Items = append(q.Items, it)
			}
		}
		out[at] = q
	}
	return out
}

func (fx *fixture) search(has func(id int) bool, v Version) optimizer.Result {
	space := optimizer.Require(fx.pool(has), fx.posOf, v.Require)
	if fx.settings.Skills && !fx.settings.Levels.None() {
		r, _ := space.BestPlacedAt(v.Stage, fx.settings.Levels)
		return r
	}
	return space.Best(v.Stage, nil)
}

func (fx *fixture) with(extra ...int) func(id int) bool {
	return func(id int) bool { return fx.owned[id] || slices.Contains(extra, id) }
}

func (fx *fixture) session() *Session {
	s := NewSession(fx.positions, fx.posOf, fx.own, fx.versions, fx.settings)
	s.Run(len(fx.versions))
	return s
}

func (fx *fixture) unowned() []int {
	var out []int
	for _, p := range fx.positions {
		for _, it := range p.Items {
			if !fx.owned[it.ID] {
				out = append(out, it.ID)
			}
		}
	}
	return out
}

func (s *Session) gainOf(vi, id int) int {
	return int(s.stages[vi].points(s.l.index[id]))
}

func (s *Session) isLive(vi, id int) bool {
	_, ok := slices.BinarySearch(s.stages[vi].live, s.l.index[id])
	return ok
}

type tally struct {
	pairs, exact, gained, skipped int
}

func checkGains(t *testing.T, trial int, fx *fixture, s *Session, n *tally) {
	t.Helper()
	for vi, v := range fx.versions {
		base := fx.search(fx.owned2(), v)
		st := s.stages[vi]
		if unmet := optimizer.Unmet(base.Items, v.Require); len(unmet) > 0 {
			if !st.failing {
				t.Fatalf("trial %d %s: the engine misses %v, and the session does not report the stage as failing", trial, v.Key, unmet)
			}
			continue
		}
		if st.failing {
			t.Fatalf("trial %d %s: reported as failing, and the engine meets every rule", trial, v.Key)
		}
		if d := st.base - base.Score; d < -1 || d > 1 {
			t.Fatalf("trial %d %s: the session starts from %d, the engine from %d", trial, v.Key, st.base, base.Score)
		}
		for _, id := range fx.unowned() {
			want := fx.search(fx.with(id), v).Score - base.Score
			got := s.gainOf(vi, id)
			n.pairs++
			if got == want {
				n.exact++
			}
			if want != 0 {
				n.gained++
			}
			if !s.isLive(vi, id) {
				n.skipped++
				if want != 0 {
					t.Fatalf("trial %d %s: item %d was skipped, and the engine gains %d with it", trial, v.Key, id, want)
				}
			}
			if d := got - want; d < -1 || d > 1 {
				t.Fatalf("trial %d %s %+v: item %d gains %d, and the engine gains %d", trial, v.Key, fx.settings, id, got, want)
			}
		}
	}
}

func (fx *fixture) owned2() func(id int) bool {
	return func(id int) bool { return fx.owned[id] }
}

func TestGainsMatchTheEngine(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 8))
	var n tally
	for trial := range 150 {
		fx := newFixture(rng, places, 4, 6, trial%2 == 1)
		checkGains(t, trial, fx, fx.session(), &n)
	}
	if n.pairs < 10000 || n.gained < 1500 || n.skipped < 3000 || n.exact*100 < n.pairs*97 {
		t.Errorf("checked %d pairs: %d exact, %d with a gain, %d skipped", n.pairs, n.exact, n.gained, n.skipped)
	}
}

func manyAccessories() []place {
	out := slices.Clone(places[:9])
	out = append(out, place{scoring.Accessory, handheld, false}, place{scoring.Accessory, handheld, false}, place{scoring.Accessory, handheld, true})
	for range 21 {
		out = append(out, place{scoring.Accessory, 0, false})
	}
	return out
}

func TestGainsMatchTheEngineUnderTheFullPenalty(t *testing.T) {
	rng := rand.New(rand.NewPCG(13, 21))
	var n tally
	for trial := range 60 {
		fx := newFixture(rng, manyAccessories(), 3, 4, trial%2 == 1)
		checkGains(t, trial, fx, fx.session(), &n)
	}
	if n.pairs < 5000 || n.gained < 1000 {
		t.Errorf("checked %d pairs: %d exact, %d with a gain, %d skipped", n.pairs, n.exact, n.gained, n.skipped)
	}
}

func withCopies(rng *rand.Rand, fx *fixture) {
	for at := range fx.positions {
		items := fx.positions[at].Items
		for _, it := range slices.Clone(items) {
			if rng.IntN(2) == 0 {
				continue
			}
			c := it
			c.ID = it.ID + 50
			c.Tags = slices.Clone(it.Tags)
			switch rng.IntN(4) {
			case 0:
			case 1:
				p := rng.IntN(5)
				c.Stats[p] = rng.IntN(c.Stats[p] + 1)
			case 2:
				c.Tags = nil
				c.FlatBonus = rng.IntN(c.FlatBonus + 1)
			default:
				c.Stats[rng.IntN(5)] += 1 + rng.IntN(20)
			}
			items = append(items, c)
			fx.posOf[c.ID] = at
			if rng.IntN(2) == 0 {
				fx.owned[c.ID] = true
			}
		}
		fx.positions[at].Items = items
	}
}

func TestGainsMatchTheEngineWithCopies(t *testing.T) {
	rng := rand.New(rand.NewPCG(41, 2))
	var n tally
	pruned := 0
	for trial := range 120 {
		fx := newFixture(rng, places, 3, 0, trial%2 == 1)
		withCopies(rng, fx)
		for k := range 6 {
			v := Version{Key: "Story/" + string(rune('a'+k)), Mode: "Story", Stage: randomStage(rng)}
			if k == 0 {
				v.Require = [][]int{{fx.positions[0].Items[len(fx.positions[0].Items)-1].ID}}
			}
			v.Ideal = fx.search(func(int) bool { return true }, v).Score
			fx.versions = append(fx.versions, v)
		}
		s := fx.session()
		for at := range s.l.positions {
			pruned += len(s.l.ownedIn[at]) - len(s.l.fillIn[at])
		}
		checkGains(t, trial, fx, s, &n)
	}
	if n.pairs < 5000 || n.gained < 500 || pruned < 300 {
		t.Errorf("checked %d pairs: %d exact, %d with a gain, %d skipped; %d owned items pruned", n.pairs, n.exact, n.gained, n.skipped, pruned)
	}
}

func TestOddAttributeCodesFallBackToContribution(t *testing.T) {
	rng := rand.New(rand.NewPCG(8, 13))
	var n tally
	odd := 0
	for trial := range 30 {
		fx := newFixture(rng, places, 3, 4, trial%2 == 1)
		for k := range fx.versions {
			if k%2 == 0 {
				fx.versions[k].Stage.Attrs[rng.IntN(5)] = 12
			}
		}
		s := fx.session()
		for _, v := range fx.versions {
			if !s.l.coefOf(v.Stage, nil).exact {
				odd++
			}
		}
		checkGains(t, trial, fx, s, &n)
	}
	if odd < 30 || n.gained < 300 {
		t.Errorf("%d stages took the slow path, %d gains checked", odd, n.gained)
	}
}

func TestSkillsChangePlacement(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 3))
	moved := 0
	for range 80 {
		fx := newFixture(rng, places, 4, 6, true)
		s := fx.session()
		for vi, v := range fx.versions {
			st := s.stages[vi]
			if st.failing {
				continue
			}
			for _, id := range fx.unowned() {
				un := optimizer.Require(fx.pool(fx.with(id)), fx.posOf, v.Require).Best(v.Stage, nil)
				if s.effective(scoring.Place(un.Items, v.Stage)) != st.place {
					moved++
				}
			}
		}
	}
	if moved < 200 {
		t.Errorf("only %d items moved the skills, so the placement fallback is barely exercised", moved)
	}
}

func TestQuickPlacementIsExact(t *testing.T) {
	rng := rand.New(rand.NewPCG(31, 7))
	checked, quick := 0, 0
	for trial := range 60 {
		layout := places
		if trial%3 == 0 {
			layout = manyAccessories()
		}
		fx := newFixture(rng, layout, 4, 5, true)
		s := fx.session()
		for vi := range fx.versions {
			st := s.stages[vi]
			if st.failing {
				continue
			}
			sc := s.scorer(st, &s.versions[vi])
			try := func(items ...int32) {
				c, _, _, _ := sc.changeOf(items)
				got, want := sc.placeWith(&c), sc.exactPlace(&c)
				if got != want {
					t.Fatalf("trial %d %s items %v: quick placement %+v, exact %+v", trial, fx.versions[vi].Key, items, got, want)
				}
				checked++
				if sc.pl.ready {
					quick++
				}
			}
			for _, id := range fx.unowned() {
				try(s.l.index[id])
			}
			var tops, bottoms []int32
			for i, it := range s.l.items {
				switch {
				case s.l.owned[i]:
				case it.Slot == scoring.Top:
					tops = append(tops, int32(i))
				case it.Slot == scoring.Bottom:
					bottoms = append(bottoms, int32(i))
				}
			}
			for _, top := range tops {
				for _, bottom := range bottoms {
					try(top, bottom)
				}
			}
			sc.release()
		}
	}
	if checked < 5000 || quick < checked/2 {
		t.Errorf("checked %d placements, %d after the quick path was set up", checked, quick)
	}
}

func TestNeededListsTheMissingItems(t *testing.T) {
	fx := &fixture{posOf: map[int]int{}, owned: map[int]bool{}}
	for at, pl := range places {
		var items []scoring.Item
		for n := range 2 {
			id := (at+1)*100 + n
			items = append(items, scoring.Item{ID: id, Slot: pl.slot, Attrs: [5]int8{0, 2, 4, 6, 8}, Stats: [5]int{10 + n, 10, 10, 10, 10}})
			fx.posOf[id] = at
			if n == 0 {
				fx.owned[id] = true
			}
		}
		fx.positions = append(fx.positions, optimizer.Position{Items: items, Group: pl.group, Exclusive: pl.whole})
	}
	st := scoring.CustomStage([5]int{-50, -20, -10, 0, 0}, nil)
	fx.versions = []Version{
		{Key: "Story/1", Mode: "Story", Stage: st, Require: [][]int{{101}, {200, 201}, {201, 9999}}, Ideal: 100},
		{Key: "Story/2", Mode: "Story", Stage: st, Require: [][]int{{100}}, Ideal: 100},
		{Key: "Story/3", Mode: "Story", Stage: st, Require: [][]int{{200}}, Ideal: 100},
		{Key: "Arena/4", Mode: "Arena", Stage: st, Require: [][]int{{9998, 9999}}, Ideal: 100},
	}
	s := fx.session()
	got := s.Rank(Filter{}, 3)
	want := []Needed{
		{Key: "Story/1", Missing: [][]int{{101}, {201, 9999}}},
		{Key: "Arena/4", Missing: [][]int{{9998, 9999}}},
	}
	if !reflect.DeepEqual(got.Needed, want) {
		t.Errorf("needed %+v, want %+v", got.Needed, want)
	}
	for _, row := range got.Rows {
		for _, ex := range row.Examples {
			if ex.Key == "Story/1" || ex.Key == "Arena/4" {
				t.Errorf("row %v counts %s, which the player fails", row.Items, ex.Key)
			}
		}
	}
	if only := s.Rank(Filter{Modes: []string{"Story"}, Skip: []string{"Story/1"}}, 0); len(only.Needed) != 0 {
		t.Errorf("the filter keeps %+v", only.Needed)
	}
	if len(got.Rows) == 0 {
		t.Fatal("nothing ranked, so the failing stages were never tested against the rows")
	}
}

func exactWorth(fx *fixture, extra []int, cands []int, views []int) float64 {
	total := 0.0
	for _, vi := range views {
		v := fx.versions[vi]
		base := fx.search(fx.with(extra...), v)
		if len(optimizer.Unmet(fx.search(fx.owned2(), v).Items, v.Require)) > 0 {
			continue
		}
		got := fx.search(fx.with(append(slices.Clone(extra), cands...)...), v).Score
		total += pct(int32(max(0, got-base.Score)), v.Ideal)
	}
	return total
}

func TestGreedyMatchesExhaustiveGreedy(t *testing.T) {
	rng := rand.New(rand.NewPCG(17, 4))
	compared, bundles := 0, 0
	for trial := range 40 {
		fx := newFixture(rng, places[:12], 3, 5, trial%2 == 1)
		s := fx.session()
		rows := s.Rank(Filter{}, 6).Rows
		var views []int
		for vi := range fx.versions {
			views = append(views, vi)
		}
		var got []int
		for step, row := range rows {
			slack := 0.0
			for _, vi := range views {
				slack += 2 * pct(1, fx.versions[vi].Ideal)
			}
			if len(row.Items) == 2 {
				bundles++
				apart := exactWorth(fx, got, row.Items[:1], views) + exactWorth(fx, got, row.Items[1:], views)
				if together := exactWorth(fx, got, row.Items, views); together <= apart-slack {
					t.Fatalf("trial %d step %d: the pair %v is worth %.4f together and %.4f apart", trial, step, row.Items, together, apart)
				}
			}
			bestWorth := math.Inf(-1)
			for _, id := range fx.unowned() {
				if slices.Contains(got, id) {
					continue
				}
				bestWorth = max(bestWorth, exactWorth(fx, got, []int{id}, views))
			}
			exact := exactWorth(fx, got, row.Items, views)
			if math.Abs(row.Worth-exact) > slack/2 {
				t.Fatalf("trial %d step %d: %v is worth %.4f, and the engine says %.4f", trial, step, row.Items, row.Worth, exact)
			}
			if exact < bestWorth-slack {
				t.Fatalf("trial %d step %d: picked %v worth %.4f, and the best single item is worth %.4f", trial, step, row.Items, exact, bestWorth)
			}
			got = append(got, row.Items...)
			compared++
		}
	}
	if compared < 100 || bundles < 3 {
		t.Errorf("compared only %d greedy steps, %d of them pairs", compared, bundles)
	}
}

func TestSeparatesComeAsAPair(t *testing.T) {
	item := func(id int, slot scoring.Slot, stat int) scoring.Item {
		return scoring.Item{ID: id, Slot: slot, Attrs: [5]int8{1, 3, 5, 7, 9}, Stats: [5]int{stat, 0, 0, 0, 0}}
	}
	fx := &fixture{
		positions: []optimizer.Position{
			{Items: []scoring.Item{item(100, scoring.Hair, 50)}},
			{Items: []scoring.Item{item(200, scoring.Dress, 300)}},
			{Items: []scoring.Item{item(300, scoring.Top, 50), item(301, scoring.Top, 200)}},
			{Items: []scoring.Item{item(400, scoring.Bottom, 50), item(401, scoring.Bottom, 200)}},
			{Items: []scoring.Item{item(500, scoring.Shoes, 10), item(501, scoring.Shoes, 30)}},
		},
		posOf: map[int]int{100: 0, 200: 1, 300: 2, 301: 2, 400: 3, 401: 3, 500: 4, 501: 4},
		owned: map[int]bool{100: true, 200: true, 300: true, 400: true, 500: true},
	}
	st := scoring.CustomStage([5]int{100, 0, 0, 0, 0}, nil)
	fx.versions = []Version{{Key: "Story/1", Mode: "Story", Stage: st}}
	fx.versions[0].Ideal = fx.search(func(int) bool { return true }, fx.versions[0]).Score
	for _, skills := range []bool{false, true} {
		if skills {
			fx.settings = Settings{Skills: true, Levels: scoring.MaxLevels}
		}
		s := fx.session()
		if got := s.gainOf(0, 301) + s.gainOf(0, 401); got != 0 {
			t.Fatalf("skills %v: the top and bottom gain %d on their own, so they need no pair", skills, got)
		}
		rows := s.Rank(Filter{}, 5).Rows
		if len(rows) != 2 || !slices.Equal(rows[0].Items, []int{301, 401}) || !slices.Equal(rows[1].Items, []int{501}) {
			t.Fatalf("skills %v: rows %+v, want the pair 301+401 first and the shoes second", skills, rows)
		}
		want := fx.search(fx.with(301, 401), fx.versions[0]).Score - fx.search(fx.owned2(), fx.versions[0]).Score
		if rows[0].Best.Points != want || rows[0].Stages != 1 {
			t.Errorf("skills %v: the pair gains %+v, and the engine gains %d", skills, rows[0].Best, want)
		}
		if only := s.Rank(Filter{Slots: []scoring.Slot{scoring.Shoes}}, 5).Rows; len(only) != 1 || only[0].Items[0] != 501 {
			t.Errorf("skills %v: the shoes filter gives %+v", skills, only)
		}
		if only := s.Rank(Filter{Positions: []int{4}}, 5).Rows; len(only) != 1 || only[0].Items[0] != 501 {
			t.Errorf("skills %v: the shoes place gives %+v", skills, only)
		}
		if tops := s.Rank(Filter{Positions: []int{2}}, 5).Rows; len(tops) != 1 || !slices.Equal(tops[0].Items, []int{301, 401}) {
			t.Errorf("skills %v: the top place gives %+v, want the pair it belongs to", skills, tops)
		}
		for _, f := range []Filter{{Positions: []int{99}}, {Positions: []int{-1}}, {Slots: []scoring.Slot{scoring.Shoes}, Positions: []int{2}}} {
			if rows := s.Rank(f, 5).Rows; len(rows) != 0 {
				t.Errorf("skills %v, filter %+v: gives %+v, want nothing", skills, f, rows)
			}
		}
	}
}

func TestRankIsDeterministicAndLeavesTheSessionAlone(t *testing.T) {
	rng := rand.New(rand.NewPCG(9, 9))
	for trial := range 12 {
		fx := newFixture(rng, places, 4, 9, trial%2 == 0)
		whole := fx.session()
		chunked := NewSession(fx.positions, fx.posOf, fx.own, fx.versions, fx.settings)
		for {
			if done, total := chunked.Run(2); done == total {
				break
			}
		}
		filters := []Filter{{}, {Modes: []string{"Story"}}, {Modes: []string{"Commission", "Arena"}, Slots: []scoring.Slot{scoring.Accessory}}, {Positions: []int{5, 12, 13}}}
		for _, f := range filters {
			first := whole.Rank(f, 8)
			again := whole.Rank(f, 8)
			other := chunked.Rank(f, 8)
			if !reflect.DeepEqual(first, again) || !reflect.DeepEqual(first, other) {
				t.Fatalf("trial %d filter %+v: rankings differ\n%+v\n%+v\n%+v", trial, f, first, again, other)
			}
		}
		for _, slots := range [][]scoring.Slot{{scoring.Accessory}, {scoring.Top, scoring.Bottom}, {scoring.Hosiery}} {
			var at []int
			for k, pl := range places {
				if slices.Contains(slots, pl.slot) {
					at = append(at, k)
				}
			}
			bySlot, byPlace := whole.Rank(Filter{Slots: slots}, 8), whole.Rank(Filter{Positions: at}, 8)
			if !reflect.DeepEqual(bySlot, byPlace) {
				t.Fatalf("trial %d: slots %v rank differently from their places %v\n%+v\n%+v", trial, slots, at, bySlot, byPlace)
			}
		}
		checkGains(t, trial, fx, whole, &tally{})
	}
}

func TestPartialRunRanksWhatIsDone(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 1))
	fx := newFixture(rng, places, 3, 6, false)
	s := NewSession(fx.positions, fx.posOf, fx.own, fx.versions, fx.settings)
	if done, total := s.Run(0); done != 0 || total != 6 {
		t.Fatalf("Run(0) = %d, %d", done, total)
	}
	if got := s.Rank(Filter{}, 5); len(got.Rows) != 0 || len(got.Needed) != 0 {
		t.Fatalf("nothing is done yet, and Rank gives %+v", got)
	}
	s.Run(2)
	for _, row := range s.Rank(Filter{}, 5).Rows {
		for _, ex := range row.Examples {
			if ex.Key != fx.versions[0].Key && ex.Key != fx.versions[1].Key {
				t.Errorf("row %v cites %s, which is not done yet", row.Items, ex.Key)
			}
		}
	}
	if done, _ := s.Run(100); done != 6 {
		t.Errorf("done = %d, want 6", done)
	}
}
