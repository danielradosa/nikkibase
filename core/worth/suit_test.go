package worth

import (
	"fmt"
	"math"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
)

func randomSuits(rng *rand.Rand, fx *fixture) []Suit {
	var ids []int
	for _, p := range fx.positions {
		for _, it := range p.Items {
			ids = append(ids, it.ID)
		}
	}
	rng.Shuffle(len(ids), func(a, b int) { ids[a], ids[b] = ids[b], ids[a] })
	var suits []Suit
	for len(ids) > 0 {
		n := min(len(ids), 1+rng.IntN(7))
		su := Suit{Key: fmt.Sprintf("suit %d", len(suits)), Items: slices.Clone(ids[:n])}
		ids = ids[n:]
		if rng.IntN(5) == 0 {
			continue
		}
		if rng.IntN(6) == 0 {
			su.Items = append(su.Items, 99990+len(suits))
		}
		suits = append(suits, su)
	}
	return suits
}

func withSuits(fx *fixture, suits []Suit) *Session {
	settings := fx.settings
	settings.Suits = suits
	s := NewSession(fx.positions, fx.posOf, fx.own, fx.versions, settings)
	s.Run(len(fx.versions))
	return s
}

func (fx *fixture) piecesOf(su Suit, not []int) []int {
	var out []int
	for _, id := range su.Items {
		if _, ok := fx.posOf[id]; ok && !fx.owned[id] && !slices.Contains(not, id) {
			out = append(out, id)
		}
	}
	return out
}

func unitNamed(s *Session, key string) int32 {
	for u := range s.units {
		if s.units[u].key == key {
			return int32(u)
		}
	}
	return -1
}

type suitTally struct {
	checks, exact, gained, several, members, moved int
}

func checkSuitGains(t *testing.T, trial int, fx *fixture, s *Session, suits []Suit, n *suitTally) {
	t.Helper()
	for vi, v := range fx.versions {
		st := s.stages[vi]
		if st.failing {
			continue
		}
		base := fx.search(fx.owned2(), v).Score
		got := map[int32]int32{}
		for _, e := range s.suitGains(vi).pts {
			got[e.u] = e.pts
		}
		for _, su := range suits {
			pieces := fx.piecesOf(su, nil)
			u := unitNamed(s, su.Key)
			if len(pieces) == 0 {
				if u >= 0 {
					t.Fatalf("trial %d: %s has nothing to get, and the session keeps it", trial, su.Key)
				}
				continue
			}
			want := fx.search(fx.with(pieces...), v).Score - base
			have := int(got[u])
			n.checks++
			if have == want {
				n.exact++
			}
			if want != 0 {
				n.gained++
				if len(pieces) > 1 {
					n.several++
				}
			}
			if slices.ContainsFunc(pieces, func(id int) bool { return st.isMember(s.l.index[id]) }) {
				n.members++
			}
			if fx.settings.Skills && !fx.settings.Levels.None() {
				un := optimizer.Require(fx.pool(fx.with(pieces...)), fx.posOf, v.Require).Best(v.Stage, nil)
				if s.effective(scoring.Place(un.Items, v.Stage)) != st.place {
					n.moved++
				}
			}
			if d := have - want; d < -1 || d > 1 {
				t.Fatalf("trial %d %s %+v: %s (%v) gains %d, and the engine gains %d", trial, v.Key, fx.settings.Levels, su.Key, pieces, have, want)
			}
		}
	}
}

func TestSuitGainsMatchTheEngine(t *testing.T) {
	rng := rand.New(rand.NewPCG(71, 3))
	var n suitTally
	for trial := range 200 {
		layout := places
		if trial%4 == 3 {
			layout = manyAccessories()
		}
		var fx *fixture
		if trial%2 == 0 {
			fx = newFixture(rng, layout, 4, 5, trial%3 != 0)
		} else {
			var settings Settings
			if trial%3 != 0 {
				settings = Settings{Skills: true, Levels: scoring.Levels{Charming: rng.IntN(10), Smile: rng.IntN(10)}}
			}
			fx = gradedFixture(rng, layout, 4, 5, settings)
		}
		suits := randomSuits(rng, fx)
		checkSuitGains(t, trial, fx, withSuits(fx, suits), suits, &n)
	}
	if n.checks < 7000 || n.gained < 4000 || n.several < 3000 || n.exact*100 < n.checks*99 || n.members < 80 || n.moved < 400 {
		t.Errorf("%d suit checks: %d exact, %d with a gain (%d of several pieces), %d holding a required item, %d moving the skills",
			n.checks, n.exact, n.gained, n.several, n.members, n.moved)
	}
}

func checkSuitState(fx *fixture, r *suitRanker, suits []Suit, picks []int, n *suitTally) error {
	s := r.s
	for j, vi := range r.vs {
		v := fx.versions[vi]
		base := fx.search(fx.with(picks...), v).Score
		if st := r.cur[j]; st.base != base && r.mine[j] {
			if d := st.base - base; d < -1 || d > 1 {
				return fmt.Errorf("%s starts from %d, and the engine from %d", v.Key, st.base, base)
			}
		}
		for _, su := range suits {
			u := unitNamed(s, su.Key)
			if u < 0 || r.picked[u] {
				continue
			}
			pieces := fx.piecesOf(su, picks)
			want := 0
			if len(pieces) > 0 {
				want = fx.search(fx.with(append(slices.Clone(picks), pieces...)...), v).Score - base
			}
			have := int(r.pointsOf(j, u))
			n.checks++
			if have == want {
				n.exact++
			}
			if want != 0 {
				n.gained++
			}
			if d := have - want; d < -1 || d > 1 {
				return fmt.Errorf("%s: %s (%v) gains %d, and the engine gains %d", v.Key, su.Key, pieces, have, want)
			}
		}
	}
	return nil
}

func (s *Session) suitRankerFor(f Filter) *suitRanker {
	var vs []int
	for vi := range s.done {
		if v := &s.versions[vi]; f.admits(v) && !s.stages[vi].failing && v.Ideal > 0 {
			vs = append(vs, vi)
		}
	}
	return s.suitRanker(vs, f)
}

func TestSuitGreedyStateMatchesTheEngineAtEveryStep(t *testing.T) {
	rng := rand.New(rand.NewPCG(29, 11))
	var n suitTally
	steps := 0
	for trial := range 150 {
		var settings Settings
		switch trial % 3 {
		case 1:
			settings = Settings{Skills: true, Levels: scoring.MaxLevels}
		case 2:
			settings = Settings{Skills: true, Levels: scoring.Levels{Charming: rng.IntN(10), Smile: rng.IntN(10)}}
		}
		layout := places
		if trial%5 == 4 {
			layout = manyAccessories()
		}
		fx := gradedFixture(rng, layout, 4, 5, settings)
		suits := randomSuits(rng, fx)
		s := withSuits(fx, suits)
		r := s.suitRankerFor(Filter{Suits: true})
		var picks []int
		for step := 0; ; step++ {
			if err := checkSuitState(fx, r, suits, picks, &n); err != nil {
				t.Fatalf("trial %d %+v step %d (picked %v): %v", trial, settings.Levels, step, picks, err)
			}
			row, ok := r.next()
			if !ok || step == 8 {
				break
			}
			picks = append(picks, row.Items...)
			steps++
		}
	}
	if n.checks < 20000 || n.gained < 5000 || steps < 400 {
		t.Errorf("%d suit checks (%d with a gain), %d steps", n.checks, n.gained, steps)
	}
	t.Logf("%d suit checks: %d exact, %d with a gain; %d steps", n.checks, n.exact, n.gained, steps)
}

func TestSuitGreedyMatchesExhaustiveGreedy(t *testing.T) {
	rng := rand.New(rand.NewPCG(37, 6))
	compared, ended := 0, 0
	for trial := range 60 {
		fx := newFixture(rng, places, 3, 5, trial%2 == 1)
		suits := randomSuits(rng, fx)
		s := withSuits(fx, suits)
		rows := s.Rank(Filter{Suits: true}, 6).Rows
		var views []int
		for vi := range fx.versions {
			views = append(views, vi)
		}
		slack := 0.0
		for _, vi := range views {
			slack += 2 * pct(1, fx.versions[vi].Ideal)
		}
		var got []int
		for step, row := range rows {
			su := suits[slices.IndexFunc(suits, func(su Suit) bool { return su.Key == row.Suit })]
			pieces := fx.piecesOf(su, got)
			slices.Sort(pieces)
			if !slices.Equal(pieces, slices.Sorted(slices.Values(row.Items))) {
				t.Fatalf("trial %d step %d: %s lists %v, and its pieces you don't own are %v", trial, step, row.Suit, row.Items, pieces)
			}
			bestWorth := math.Inf(-1)
			for _, other := range suits {
				if pieces := fx.piecesOf(other, got); len(pieces) > 0 {
					bestWorth = max(bestWorth, exactWorth(fx, got, pieces, views))
				}
			}
			exact := exactWorth(fx, got, row.Items, views)
			if math.Abs(row.Worth-exact) > slack/2 {
				t.Fatalf("trial %d step %d: %s is worth %.4f, and the engine says %.4f", trial, step, row.Suit, row.Worth, exact)
			}
			if exact < bestWorth-slack {
				t.Fatalf("trial %d step %d: picked %s worth %.4f, and the best suit is worth %.4f", trial, step, row.Suit, exact, bestWorth)
			}
			got = append(got, row.Items...)
			compared++
		}
		if len(rows) < 6 {
			for _, other := range suits {
				if pieces := fx.piecesOf(other, got); len(pieces) > 0 && exactWorth(fx, got, pieces, views) > slack {
					t.Fatalf("trial %d: the ranking stops after %d rows, and %s is still worth %.4f", trial, len(rows), other.Key, exactWorth(fx, got, pieces, views))
				}
			}
			ended++
		}
	}
	if compared < 150 || ended < 5 {
		t.Errorf("compared only %d greedy steps, %d rankings ran out", compared, ended)
	}
}

func TestSuitRankIsDeterministicAndLeavesTheSessionAlone(t *testing.T) {
	rng := rand.New(rand.NewPCG(19, 2))
	for trial := range 12 {
		fx := newFixture(rng, places, 4, 9, trial%2 == 0)
		suits := randomSuits(rng, fx)
		whole := withSuits(fx, suits)
		settings := fx.settings
		settings.Suits = suits
		chunked := NewSession(fx.positions, fx.posOf, fx.own, fx.versions, settings)
		for {
			if done, total := chunked.Run(2); done == total {
				break
			}
		}
		items := whole.Rank(Filter{}, 8)
		filters := []Filter{{Suits: true}, {Suits: true, Modes: []string{"Story"}}, {Suits: true, Slots: []scoring.Slot{scoring.Accessory}}, {Suits: true, Positions: []int{5, 12, 13}}}
		for _, f := range filters {
			first := whole.Rank(f, 8)
			again := whole.Rank(f, 8)
			other := chunked.Rank(f, 8)
			if !reflect.DeepEqual(first, again) || !reflect.DeepEqual(first, other) {
				t.Fatalf("trial %d filter %+v: rankings differ\n%+v\n%+v\n%+v", trial, f, first, again, other)
			}
		}
		if after := whole.Rank(Filter{}, 8); !reflect.DeepEqual(items, after) {
			t.Fatalf("trial %d: the item ranking changed after ranking suits\n%+v\n%+v", trial, items, after)
		}
		checkSuitGains(t, trial, fx, whole, suits, &suitTally{})
		checkGains(t, trial, fx, whole, &tally{})
	}
}

func TestSuitFiltersAdmitASuitByAnyPiece(t *testing.T) {
	rng := rand.New(rand.NewPCG(43, 5))
	for trial := range 20 {
		fx := newFixture(rng, places, 3, 6, trial%2 == 1)
		suits := randomSuits(rng, fx)
		s := withSuits(fx, suits)
		all := s.Rank(Filter{Suits: true}, 1000).Rows
		for _, slots := range [][]scoring.Slot{{scoring.Accessory}, {scoring.Dress}, {scoring.Hair, scoring.Shoes}} {
			var at []int
			for k, pl := range places {
				if slices.Contains(slots, pl.slot) {
					at = append(at, k)
				}
			}
			bySlot, byPlace := s.Rank(Filter{Suits: true, Slots: slots}, 1000), s.Rank(Filter{Suits: true, Positions: at}, 1000)
			if !reflect.DeepEqual(bySlot, byPlace) {
				t.Fatalf("trial %d: slots %v rank suits differently from their places %v", trial, slots, at)
			}
			for _, row := range bySlot.Rows {
				ok := false
				for _, id := range row.Items {
					ok = ok || slices.Contains(slots, fx.positions[fx.posOf[id]].Items[0].Slot)
				}
				if !ok {
					t.Fatalf("trial %d: %s has no piece in %v and is listed", trial, row.Suit, slots)
				}
			}
			if len(bySlot.Rows) > 0 && len(all) > 0 && bySlot.Rows[0].Suit == all[0].Suit && bySlot.Rows[0].Worth != all[0].Worth {
				t.Fatalf("trial %d: the first suit is worth %.4f with the filter and %.4f without", trial, bySlot.Rows[0].Worth, all[0].Worth)
			}
		}
	}
}

func TestSuitRowsNameTheSuitAndItsPieces(t *testing.T) {
	item := func(id int, slot scoring.Slot, stat int) scoring.Item {
		return scoring.Item{ID: id, Slot: slot, Attrs: [5]int8{1, 3, 5, 7, 9}, Stats: [5]int{stat, 0, 0, 0, 0}}
	}
	fx := &fixture{
		positions: []optimizer.Position{
			{Items: []scoring.Item{item(100, scoring.Hair, 50), item(101, scoring.Hair, 60)}},
			{Items: []scoring.Item{item(200, scoring.Dress, 300), item(201, scoring.Dress, 250), item(202, scoring.Dress, 280)}},
			{Items: []scoring.Item{item(300, scoring.Top, 50), item(301, scoring.Top, 200)}},
			{Items: []scoring.Item{item(400, scoring.Bottom, 50), item(401, scoring.Bottom, 200)}},
			{Items: []scoring.Item{item(500, scoring.Shoes, 10), item(501, scoring.Shoes, 30)}},
			{Items: []scoring.Item{item(600, scoring.Coat, 5), item(601, scoring.Coat, 25)}},
		},
		posOf: map[int]int{100: 0, 101: 0, 200: 1, 201: 1, 202: 1, 300: 2, 301: 2, 400: 3, 401: 3, 500: 4, 501: 4, 600: 5, 601: 5},
		owned: map[int]bool{100: true, 200: true, 300: true, 400: true, 500: true, 600: true},
	}
	fx.versions = []Version{{Key: "Story/1", Mode: "Story", Stage: scoring.CustomStage([5]int{100, 0, 0, 0, 0}, nil)}}
	fx.versions[0].Ideal = fx.search(func(int) bool { return true }, fx.versions[0]).Score
	suits := []Suit{
		{Key: "Separates", Items: []int{401, 301, 300}},
		{Key: "Dresses", Items: []int{201, 202}},
		{Key: "Shoes and hair", Items: []int{501, 101, 5555}},
		{Key: "Owned", Items: []int{100, 200}},
		{Key: "", Items: []int{601}},
	}
	for _, skills := range []bool{false, true} {
		fx.settings = Settings{}
		if skills {
			fx.settings = Settings{Skills: true, Levels: scoring.MaxLevels}
		}
		s := withSuits(fx, suits)
		rows := s.Rank(Filter{Suits: true}, 10).Rows
		if len(rows) != 2 || rows[0].Suit != "Separates" || !slices.Equal(rows[0].Items, []int{301, 401}) ||
			rows[1].Suit != "Shoes and hair" || !slices.Equal(rows[1].Items, []int{101, 501}) {
			t.Fatalf("skills %v: rows %+v, want Separates (301, 401) and then Shoes and hair (101, 501)", skills, rows)
		}
		want := fx.search(fx.with(301, 401), fx.versions[0]).Score - fx.search(fx.owned2(), fx.versions[0]).Score
		if rows[0].Best.Points != want || rows[0].Stages != 1 || rows[0].Examples[0].Key != "Story/1" {
			t.Errorf("skills %v: Separates gains %+v, and the engine gains %d", skills, rows[0].Best, want)
		}
		if items := s.Rank(Filter{}, 10).Rows; len(items) == 0 || items[0].Suit != "" {
			t.Errorf("skills %v: item rows %+v carry a suit", skills, items)
		}
		if only := s.Rank(Filter{Suits: true, Slots: []scoring.Slot{scoring.Dress}}, 10).Rows; len(only) != 0 {
			t.Errorf("skills %v: the dress filter lists %+v, and no dress suit gains", skills, only)
		}
		if none := s.Rank(Filter{Suits: true}, 0); len(none.Rows) != 0 {
			t.Errorf("skills %v: limit 0 gives %+v", skills, none.Rows)
		}
	}
	if rows := withSuits(fx, nil).Rank(Filter{Suits: true}, 5).Rows; len(rows) != 0 {
		t.Errorf("without suits the suit view lists %+v", rows)
	}
}

func TestSuitRowsFollowThePartialRun(t *testing.T) {
	rng := rand.New(rand.NewPCG(53, 8))
	for trial := range 10 {
		fx := newFixture(rng, places, 3, 6, trial%2 == 1)
		suits := randomSuits(rng, fx)
		settings := fx.settings
		settings.Suits = suits
		s := NewSession(fx.positions, fx.posOf, fx.own, fx.versions, settings)
		if got := s.Rank(Filter{Suits: true}, 5); len(got.Rows) != 0 {
			t.Fatalf("trial %d: nothing is done yet, and the suits rank as %+v", trial, got.Rows)
		}
		s.Run(2)
		for _, row := range s.Rank(Filter{Suits: true}, 5).Rows {
			for _, ex := range row.Examples {
				if ex.Key != fx.versions[0].Key && ex.Key != fx.versions[1].Key {
					t.Errorf("trial %d: %s cites %s, which is not done yet", trial, row.Suit, ex.Key)
				}
			}
		}
		s.Run(100)
		want := withSuits(fx, suits).Rank(Filter{Suits: true}, 6)
		short := s.Rank(Filter{Suits: true}, 3)
		long := s.Rank(Filter{Suits: true}, 6)
		if !reflect.DeepEqual(long, want) {
			t.Fatalf("trial %d: after the rest of the run the suits rank as\n%+v\nand a whole run ranks them as\n%+v", trial, long, want)
		}
		if !reflect.DeepEqual(short.Rows, want.Rows[:min(3, len(want.Rows))]) {
			t.Fatalf("trial %d: the first three rows %+v differ from the longer ranking %+v", trial, short.Rows, want.Rows)
		}
		if again := s.Rank(Filter{Suits: true}, 2); !reflect.DeepEqual(again.Rows, want.Rows[:min(2, len(want.Rows))]) {
			t.Fatalf("trial %d: asking for fewer rows gives %+v", trial, again.Rows)
		}
	}
}
