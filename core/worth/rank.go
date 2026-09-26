package worth

import (
	"slices"

	"github.com/danielradosa/nikkibase/core/scoring"
)

type bundle struct {
	items [2]int32
	pts   []int32
	worth float64
	count int32
	dead  bool
}

type tracker struct {
	s      *Session
	vs     []int
	cur    []*stage
	mine   []bool
	seen   []int
	extras []int32
	slots  []bool
	at     []bool
	light  bool
}

type ranker struct {
	tracker
	worth   []float64
	count   []int32
	picked  []bool
	pairs   []bundle
	pairsOf map[int32][]int
}

func (s *Session) Rank(f Filter, limit int) Ranking {
	var out Ranking
	var vs []int
	for vi := range s.done {
		v := &s.versions[vi]
		if !f.admits(v) {
			continue
		}
		if st := s.stages[vi]; st.failing {
			missing := make([][]int, len(st.missing))
			for k, set := range st.missing {
				missing[k] = slices.Clone(set)
			}
			out.Needed = append(out.Needed, Needed{Key: v.Key, Missing: missing})
			continue
		}
		if v.Ideal > 0 {
			vs = append(vs, vi)
		}
	}
	if limit <= 0 {
		return out
	}
	if f.Suits {
		out.Rows = s.suitRows(vs, f, limit)
		return out
	}
	r := s.ranker(vs, f)
	for len(out.Rows) < limit {
		row, ok := r.next()
		if !ok {
			break
		}
		out.Rows = append(out.Rows, row)
	}
	return out
}

func (f Filter) admits(v *Version) bool {
	return (len(f.Modes) == 0 || slices.Contains(f.Modes, v.Mode)) && !slices.Contains(f.Skip, v.Key)
}

func (s *Session) track(vs []int, f Filter) tracker {
	t := tracker{s: s, vs: vs, cur: make([]*stage, len(vs)), mine: make([]bool, len(vs)), seen: make([]int, len(vs))}
	if len(f.Slots) > 0 {
		t.slots = make([]bool, scoring.Spirit+1)
		for _, slot := range f.Slots {
			if int(slot) < len(t.slots) {
				t.slots[slot] = true
			}
		}
	}
	if len(f.Positions) > 0 {
		t.at = make([]bool, len(s.l.positions))
		for _, at := range f.Positions {
			if at >= 0 && at < len(t.at) {
				t.at[at] = true
			}
		}
	}
	for j, vi := range vs {
		t.cur[j] = s.stages[vi]
	}
	return t
}

func (s *Session) ranker(vs []int, f Filter) *ranker {
	n := len(s.l.items)
	r := &ranker{tracker: s.track(vs, f), worth: make([]float64, n), count: make([]int32, n), picked: make([]bool, n)}
	for j := range vs {
		st := r.cur[j]
		for e, i := range st.live {
			r.setSingle(j, i, 0, st.pts[e])
		}
	}
	for _, items := range s.pairs {
		if !r.admits(items[0]) && !r.admits(items[1]) {
			continue
		}
		if r.pairsOf == nil {
			r.pairsOf = map[int32][]int{}
		}
		for _, i := range items {
			r.pairsOf[i] = append(r.pairsOf[i], len(r.pairs))
		}
		r.pairs = append(r.pairs, bundle{items: items, pts: make([]int32, len(vs))})
	}
	if len(r.pairs) > 0 {
		for j := range vs {
			r.refreshPairs(j, nil)
		}
	}
	return r
}

func (t *tracker) admits(i int32) bool {
	l := t.s.l
	slot := l.items[i].Slot
	return (t.slots == nil || int(slot) < len(t.slots) && t.slots[slot]) && (t.at == nil || t.at[l.at[i]])
}

func (t *tracker) ideal(j int) int {
	return t.s.versions[t.vs[j]].Ideal
}

func (r *ranker) setSingle(j int, i int32, old, now int32) {
	if old == now {
		return
	}
	if old > 0 {
		r.worth[i] -= pct(old, r.ideal(j))
		r.count[i]--
	}
	if now > 0 {
		r.worth[i] += pct(now, r.ideal(j))
		r.count[i]++
	}
}

func (r *ranker) setPair(j, k int, now int32) {
	b := &r.pairs[k]
	old := b.pts[j]
	if old == now {
		return
	}
	if old > 0 {
		b.worth -= pct(old, r.ideal(j))
		b.count--
	}
	if now > 0 {
		b.worth += pct(now, r.ideal(j))
		b.count++
	}
	b.pts[j] = now
}

func ahead(worth float64, count int32, id int, otherWorth float64, otherCount int32, otherID int) bool {
	if worth != otherWorth {
		return worth > otherWorth
	}
	if count != otherCount {
		return count > otherCount
	}
	return id < otherID
}

func (r *ranker) next() (Row, bool) {
	l := r.s.l
	chosen := int32(-1)
	for i := range int32(len(l.items)) {
		if r.count[i] == 0 || r.picked[i] || !r.admits(i) {
			continue
		}
		if chosen < 0 || ahead(r.worth[i], r.count[i], l.items[i].ID, r.worth[chosen], r.count[chosen], l.items[chosen].ID) {
			chosen = i
		}
	}
	pair := -1
	for k := range r.pairs {
		b := &r.pairs[k]
		if b.dead || b.count == 0 || !r.complementary(b) {
			continue
		}
		if pair < 0 || ahead(b.worth, b.count, r.firstID(b), r.pairs[pair].worth, r.pairs[pair].count, r.firstID(&r.pairs[pair])) {
			pair = k
		}
	}
	if pair >= 0 && (chosen < 0 || ahead(r.pairs[pair].worth, r.pairs[pair].count, r.firstID(&r.pairs[pair]),
		r.worth[chosen], r.count[chosen], l.items[chosen].ID)) {
		b := &r.pairs[pair]
		items := b.items
		row := r.row([]int{l.items[items[0]].ID, l.items[items[1]].ID}, func(j int) int32 { return b.pts[j] })
		r.take(items[:])
		return row, true
	}
	if chosen < 0 {
		return Row{}, false
	}
	row := r.row([]int{l.items[chosen].ID}, func(j int) int32 { return r.cur[j].points(chosen) })
	r.take([]int32{chosen})
	return row, true
}

func (r *ranker) complementary(b *bundle) bool {
	apart := r.worth[b.items[0]] + r.worth[b.items[1]]
	return b.worth > apart+1e-9*max(1, b.worth)
}

func (r *ranker) firstID(b *bundle) int {
	return min(r.s.l.items[b.items[0]].ID, r.s.l.items[b.items[1]].ID)
}

func (t *tracker) row(ids []int, at func(j int) int32) Row {
	row := Row{Items: ids}
	top := make([]Example, 0, examples+1)
	for j, vi := range t.vs {
		p := at(j)
		if p <= 0 {
			continue
		}
		v := &t.s.versions[vi]
		e := Example{Key: v.Key, Points: int(p), Pct: pct(p, v.Ideal)}
		row.Worth += e.Pct
		row.Stages++
		k := len(top)
		for k > 0 && top[k-1].Pct < e.Pct {
			k--
		}
		if k < examples {
			top = append(top[:k], append([]Example{e}, top[k:]...)...)[:min(len(top)+1, examples)]
		}
	}
	row.Examples = top
	if len(top) > 0 {
		row.Best = top[0]
	}
	return row
}

func (r *ranker) take(items []int32) {
	for _, i := range items {
		r.picked[i] = true
		r.extras = append(r.extras, i)
		for _, k := range r.pairsOf[i] {
			r.pairs[k].dead = true
		}
	}
	for j := range r.vs {
		switch {
		case r.touches(j, items):
			r.update(j, items)
		case r.reachesMembers(j, items):
			r.reprice(j)
		}
	}
}

func (r *ranker) touches(j int, items []int32) bool {
	st, l := r.cur[j], r.s.l
	for _, i := range items {
		if _, ok := slices.BinarySearch(st.live, i); ok {
			return true
		}
		for _, pl := range st.placed {
			s, f := l.value(pl.c, i)
			for b, br := range st.branches {
				if !br.blocked[l.at[i]] && l.improves(pl.sides[b], i, s, f) {
					return true
				}
			}
		}
	}
	return len(st.members) > 0 && r.pairsWithMembers(st)
}

func (t *tracker) reachesMembers(j int, items []int32) bool {
	st := t.cur[j]
	if len(st.members) == 0 {
		return false
	}
	for _, i := range items {
		if t.s.reaches(st, i) {
			return true
		}
	}
	return false
}

func (r *ranker) reprice(j int) {
	st := r.catchUp(j)
	sc := &scorer{s: r.s, v: &r.s.versions[r.vs[j]], st: st}
	for e, i := range st.live {
		if r.picked[i] || !st.isMember(i) {
			continue
		}
		now := sc.memberGain(i)
		r.setSingle(j, i, st.pts[e], now)
		st.pts[e] = now
	}
}

func (t *tracker) catchUp(j int) *stage {
	s, st := t.s, t.cur[j]
	if !t.mine[j] {
		st = st.clone(t.light)
		t.cur[j], t.mine[j] = st, true
	}
	for _, i := range t.extras[t.seen[j]:] {
		for b, br := range st.branches {
			s.l.addTo(st.u[b], br, st.cU, i)
			if st.k != nil {
				s.l.addTo(st.k[b], br, st.cK, i)
			}
			for _, pl := range st.placed {
				s.l.addTo(pl.sides[b], br, pl.c, i)
			}
		}
		for _, rc := range st.reach {
			v, f := s.l.value(rc.c, i)
			s.l.put(rc.sd, i, v, f)
		}
		st.added = append(st.added, i)
	}
	t.seen[j] = len(t.extras)
	return st
}

func (s *Session) reaches(st *stage, i int32) bool {
	l := s.l
	at := l.at[i]
	a := l.accOf[at]
	for _, rc := range st.reach {
		v, f := l.value(rc.c, i)
		sd := rc.sd
		if a < 0 {
			if sd.item[at] < 0 || v+f >= sd.best[at] {
				return true
			}
			continue
		}
		from := int(a) * len(ratios)
		for k, r := range ratios {
			if sd.pickItem[from+k] < 0 || r*v+f >= sd.pick[from+k] {
				return true
			}
		}
	}
	return false
}

func (r *ranker) pairsWithMembers(st *stage) bool {
	for _, m := range st.members {
		for _, k := range r.pairsOf[m] {
			if !r.pairs[k].dead {
				return true
			}
		}
	}
	return false
}

func (l *layout) improves(sd *side, i int32, s, f float64) bool {
	at := l.at[i]
	a := l.accOf[at]
	if a < 0 {
		return s+f > sd.best[at]
	}
	from := int(a) * len(ratios)
	for k, r := range ratios {
		if r*s+f > sd.pick[from+k] {
			return true
		}
	}
	return false
}

func (r *ranker) update(j int, items []int32) {
	s := r.s
	vi := r.vs[j]
	v := &s.versions[vi]
	st := r.cur[j]
	rebuild := false
	for _, i := range items {
		rebuild = rebuild || st.isMember(i)
	}
	if !rebuild {
		st = r.catchUp(j)
		sc := s.scorer(st, v)
		if p := st.place; s.skills {
			p = sc.placeWith(&change{})
			if p != st.place && len(v.Require) > 0 {
				sc.release()
				rebuild = true
			} else if p != st.place {
				s.moveTo(sc, p, r.picked)
			}
		}
		if !rebuild {
			st.base = bestOf(sc.scoring(), &change{})
			stride := s.stride()
			for e, i := range st.live {
				var now int32
				switch {
				case r.picked[i]:
				case st.isMember(i):
					now = sc.memberGain(i)
				default:
					cu, ck := st.valued(e, stride, s.l.at[i], i)
					liveU := sc.liveU(&cu)
					if liveU || s.skills && sc.beats(sc.evK, &ck) {
						now = sc.gain(&cu, &ck, liveU)
					}
				}
				r.setSingle(j, i, st.pts[e], now)
				st.pts[e] = now
			}
			r.refreshPairs(j, sc)
			sc.release()
			return
		}
	}
	fresh := s.process(vi, slices.Clone(r.extras))
	old := r.cur[j]
	a, b := 0, 0
	for a < len(old.live) || b < len(fresh.live) {
		switch {
		case b == len(fresh.live) || a < len(old.live) && old.live[a] < fresh.live[b]:
			r.setSingle(j, old.live[a], old.pts[a], 0)
			a++
		case a == len(old.live) || fresh.live[b] < old.live[a]:
			r.setSingle(j, fresh.live[b], 0, fresh.pts[b])
			b++
		default:
			r.setSingle(j, old.live[a], old.pts[a], fresh.pts[b])
			a++
			b++
		}
	}
	r.cur[j], r.mine[j], r.seen[j] = fresh, true, len(r.extras)
	r.refreshPairs(j, nil)
}

func (r *ranker) refreshPairs(j int, sc *scorer) {
	if len(r.pairs) == 0 {
		return
	}
	st := r.cur[j]
	var todo []int
	for k := range r.pairs {
		b := &r.pairs[k]
		if b.dead {
			continue
		}
		_, liveT := slices.BinarySearch(st.live, b.items[0])
		_, liveB := slices.BinarySearch(st.live, b.items[1])
		if liveT || liveB || b.pts[j] != 0 {
			todo = append(todo, k)
		}
	}
	if len(todo) == 0 {
		return
	}
	own := sc == nil
	if own {
		sc = r.s.scorer(st, &r.s.versions[r.vs[j]])
	}
	for _, k := range todo {
		r.setPair(j, k, sc.bundleGain(r.pairs[k].items[:]))
	}
	if own {
		sc.release()
	}
}

func (sc *scorer) bundleGain(items []int32) int32 {
	st := sc.st
	for _, i := range items {
		if st.isMember(i) {
			return sc.engineGain(items...)
		}
	}
	cu, ck, liveU, live := sc.changeOf(items)
	if !live {
		return 0
	}
	return sc.gain(&cu, &ck, liveU)
}
