package worth

import (
	"slices"
	"strconv"

	"github.com/danielradosa/nikkibase/core/scoring"
)

type unit struct {
	key    string
	pieces []int32
	first  int
}

type unitPts struct {
	u     int32
	pts   int32
	alive uint64
}

const everyPiece = ^uint64(0)

type suitBase struct {
	done bool
	pts  []unitPts
}

func (s *Session) addSuits(suits []Suit) {
	l := s.l
	for _, su := range suits {
		if su.Key == "" {
			continue
		}
		var pieces []int32
		for _, id := range su.Items {
			if i, ok := l.index[id]; ok && !l.owned[i] {
				pieces = append(pieces, i)
			}
		}
		if len(pieces) == 0 {
			continue
		}
		slices.Sort(pieces)
		pieces = slices.Compact(pieces)
		first := l.items[pieces[0]].ID
		for _, i := range pieces {
			first = min(first, l.items[i].ID)
		}
		if s.unitsOf == nil {
			s.unitsOf = make([][]int32, len(l.items))
		}
		u := int32(len(s.units))
		for _, i := range pieces {
			s.unitsOf[i] = append(s.unitsOf[i], u)
		}
		s.units = append(s.units, unit{key: su.Key, pieces: pieces, first: first})
	}
	s.suits = make([]suitBase, len(s.versions))
}

func (s *Session) suitGains(vi int) *suitBase {
	b := &s.suits[vi]
	if b.done {
		return b
	}
	b.done = true
	st := s.stages[vi]
	if st.failing || s.unitsOf == nil {
		return b
	}
	var cands []int32
	for _, i := range st.live {
		cands = append(cands, s.unitsOf[i]...)
	}
	if len(cands) == 0 {
		return b
	}
	slices.Sort(cands)
	cands = slices.Compact(cands)
	sc := s.scorer(st, &s.versions[vi])
	b.pts = make([]unitPts, 0, len(cands))
	for _, u := range cands {
		gain, alive, _ := sc.unitGain(u, everyPiece, nil)
		b.pts = append(b.pts, unitPts{u, gain, alive})
	}
	sc.release()
	return b
}

func (sc *scorer) unitGain(u int32, mask uint64, taken []bool) (int32, uint64, bool) {
	s, st, l := sc.s, sc.st, sc.s.l
	pieces := s.units[u].pieces
	if len(st.members) > 0 {
		rest := s.cons[:0]
		member := false
		for _, i := range pieces {
			if taken == nil || !taken[i] {
				rest = append(rest, i)
				member = member || st.isMember(i)
			}
		}
		s.cons = rest
		if member {
			gain, p := sc.enginePlaced(rest...)
			sc.reachFor(p)
			return gain, everyPiece, true
		}
	}
	moved, up := s.pieces[1][:0], s.pieces[2][:0]
	alive := uint64(0)
	for k, i := range pieces {
		if k < 64 && mask&(1<<k) == 0 || taken != nil && taken[i] {
			continue
		}
		at := l.at[i]
		v, f := l.value(st.cU, i)
		live := false
		for _, e := range sc.evU {
			if s.skills && e.displaces(at, i, v, f) || !s.skills && e.beats(at, v, f) {
				live = true
				break
			}
		}
		if live {
			moved = append(moved, piece{at, i, v, f})
		}
		if s.skills && (live || sc.mightBeat(sc.evK, st.cK, at, v, f)) {
			v, f = l.value(st.cK, i)
			for _, e := range sc.evK {
				if e.beats(at, v, f) {
					up = append(up, piece{at, i, v, f})
					live = true
					break
				}
			}
		}
		if live && k < 64 {
			alive |= 1 << k
		}
	}
	s.pieces[1], s.pieces[2] = moved, up
	if !s.skills {
		if len(moved) == 0 {
			return 0, 0, false
		}
		return int32(sc.manyBest(sc.evU, moved) - st.base), alive, true
	}
	if len(moved) == 0 && len(up) == 0 {
		return 0, 0, false
	}
	p := st.place
	switch len(moved) {
	case 0:
	case 1:
		c := single(moved[0].at, moved[0].item, moved[0].s, moved[0].f)
		p = sc.placeWith(&c)
	default:
		p = sc.placeMany(moved)
	}
	if p == st.place {
		return int32(sc.manyBest(sc.evK, up) - st.base), alive, true
	}
	evs, c := sc.placedFor(p)
	all := s.pieces[0][:0]
	for _, i := range pieces {
		if taken != nil && taken[i] {
			continue
		}
		at := l.at[i]
		v, f := l.value(c, i)
		for _, e := range evs {
			if e.beats(at, v, f) {
				all = append(all, piece{at, i, v, f})
				break
			}
		}
	}
	s.pieces[0] = all
	return int32(sc.manyBest(evs, all) - st.base), alive, true
}

func (sc *scorer) manyBest(evs []*evaluator, ps []piece) int {
	switch len(ps) {
	case 0:
		return bestOf(evs, &change{})
	case 1:
		c := single(ps[0].at, ps[0].item, ps[0].s, ps[0].f)
		return bestOf(evs, &c)
	}
	best := 0
	for b, e := range evs {
		t, _, _ := e.withMany(ps)
		if score := scoring.Floor(t); b == 0 || score > best {
			best = score
		}
	}
	return best
}

func (sc *scorer) placeMany(moved []piece) scoring.Placement {
	s := sc.s
	chosen, best, choice, worn := 0, 0, 0, 0
	for b, e := range sc.evU {
		t, c, w := e.withMany(moved)
		if score := scoring.Floor(t); b == 0 || score > best {
			chosen, best, choice, worn = b, score, c, w
		}
	}
	e := sc.evU[chosen]
	s.buf = e.outfitMany(s.buf, moved, choice, worn)
	return s.effective(scoring.Place(s.buf, sc.v.Stage))
}

func (l *layout) displacesIn(sd *side, i int32, s, f float64) bool {
	at := l.at[i]
	a := l.accOf[at]
	if a < 0 {
		return first(s+f, i, sd.best[at], sd.item[at])
	}
	from := int(a) * len(ratios)
	for k, r := range ratios {
		if first(r*s+f, i, sd.pick[from+k], sd.pickItem[from+k]) {
			return true
		}
	}
	return false
}

type suitRanker struct {
	tracker
	list    [][]unitPts
	dormant [][]unitPts
	scanned []bool
	cands   []unitPts
	worth   []float64
	count   []int32
	admit   []bool
	picked  []bool
	taken   []bool
	buf     []int32
}

const (
	keepListed = iota
	rescanAll
	fromLive
)

func (s *Session) suitRanker(vs []int, f Filter) *suitRanker {
	n := len(s.units)
	t := s.track(vs, f)
	t.light = true
	r := &suitRanker{tracker: t, list: make([][]unitPts, len(vs)), dormant: make([][]unitPts, len(vs)),
		scanned: make([]bool, len(vs)), worth: make([]float64, n),
		count: make([]int32, n), admit: make([]bool, n), picked: make([]bool, n), taken: make([]bool, len(s.l.items))}
	for u := range s.units {
		for _, i := range s.units[u].pieces {
			if r.admits(i) {
				r.admit[u] = true
				break
			}
		}
	}
	for j, vi := range vs {
		b := s.suitGains(vi)
		list := make([]unitPts, 0, len(b.pts))
		for _, e := range b.pts {
			if r.admit[e.u] {
				list = append(list, e)
				r.set(j, e.u, 0, e.pts)
			}
		}
		r.list[j] = list
	}
	return r
}

type suitRun struct {
	key  string
	done int
	r    *suitRanker
	rows []Row
	over bool
}

func (s *Session) suitRows(vs []int, f Filter, limit int) []Row {
	key := f.key()
	c := s.suitRun
	if c == nil || c.key != key || c.done != s.done {
		c = &suitRun{key: key, done: s.done, r: s.suitRanker(vs, f)}
		s.suitRun = c
	}
	for !c.over && len(c.rows) < limit {
		row, ok := c.r.next()
		if !ok {
			c.over = true
			break
		}
		c.rows = append(c.rows, row)
	}
	return slices.Clone(c.rows[:min(limit, len(c.rows))])
}

func (f Filter) key() string {
	var b []byte
	for _, list := range [][]string{f.Modes, f.Skip} {
		for _, v := range list {
			b = strconv.AppendQuote(b, v)
		}
		b = append(b, '|')
	}
	for _, slot := range f.Slots {
		b = strconv.AppendInt(append(b, ','), int64(slot), 10)
	}
	b = append(b, '|')
	for _, at := range f.Positions {
		b = strconv.AppendInt(append(b, ','), int64(at), 10)
	}
	return string(b)
}

func (r *suitRanker) set(j int, u int32, old, now int32) {
	if old == now {
		return
	}
	if old > 0 {
		r.worth[u] -= pct(old, r.ideal(j))
		r.count[u]--
	}
	if now > 0 {
		r.worth[u] += pct(now, r.ideal(j))
		r.count[u]++
	}
}

func (r *suitRanker) remaining(u int32) []int32 {
	out := r.buf[:0]
	for _, i := range r.s.units[u].pieces {
		if !r.taken[i] {
			out = append(out, i)
		}
	}
	r.buf = out
	return out
}

func (r *suitRanker) next() (Row, bool) {
	units := r.s.units
	chosen := int32(-1)
	for u := range int32(len(units)) {
		if !r.admit[u] || r.picked[u] || r.count[u] == 0 {
			continue
		}
		if chosen < 0 || ahead(r.worth[u], r.count[u], units[u].first, r.worth[chosen], r.count[chosen], units[chosen].first) {
			chosen = u
		}
	}
	if chosen < 0 {
		return Row{}, false
	}
	pieces := slices.Clone(r.remaining(chosen))
	ids := make([]int, len(pieces))
	for k, i := range pieces {
		ids[k] = r.s.l.items[i].ID
	}
	row := r.row(ids, func(j int) int32 { return r.pointsOf(j, chosen) })
	row.Suit = units[chosen].key
	r.take(chosen, pieces)
	return row, true
}

func (r *suitRanker) pointsOf(j int, u int32) int32 {
	list := r.list[j]
	if k, ok := slices.BinarySearchFunc(list, u, func(e unitPts, u int32) int { return int(e.u - u) }); ok {
		return list[k].pts
	}
	return 0
}

func (r *suitRanker) take(u int32, pieces []int32) {
	r.picked[u] = true
	for _, i := range pieces {
		r.taken[i] = true
	}
	r.extras = append(r.extras, pieces...)
	for j := range r.vs {
		switch {
		case r.changes(j, pieces):
			r.update(j, pieces)
		case r.reachesMembers(j, pieces):
			r.repriceMembers(j)
		}
	}
}

func (r *suitRanker) changes(j int, picks []int32) bool {
	s, st, l := r.s, r.cur[j], r.s.l
	for _, i := range picks {
		if st.isMember(i) {
			return true
		}
		at := l.at[i]
		sU, fU := l.value(st.cU, i)
		var sK, fK float64
		if st.k != nil {
			sK, fK = l.value(st.cK, i)
		}
		for b, br := range st.branches {
			if br.blocked[at] {
				continue
			}
			if s.skills && l.displacesIn(st.u[b], i, sU, fU) || !s.skills && l.improves(st.u[b], i, sU, fU) {
				return true
			}
			if st.k != nil && l.improves(st.k[b], i, sK, fK) {
				return true
			}
		}
		for _, pl := range st.placed {
			v, f := l.value(pl.c, i)
			for b, br := range st.branches {
				if !br.blocked[at] && l.improves(pl.sides[b], i, v, f) {
					return true
				}
			}
		}
	}
	return false
}

func (r *suitRanker) update(j int, picks []int32) {
	s := r.s
	vi := r.vs[j]
	v := &s.versions[vi]
	st := r.cur[j]
	rebuild := false
	for _, i := range picks {
		rebuild = rebuild || st.isMember(i)
	}
	if !rebuild {
		st = r.catchUp(j)
		sc := s.scorer(st, v)
		mode := keepListed
		if s.skills {
			if p := sc.placeWith(&change{}); p != st.place {
				if len(v.Require) > 0 {
					rebuild = true
				} else {
					s.switchTo(sc, p)
					mode = rescanAll
				}
			}
		}
		if !rebuild {
			st.base = bestOf(sc.scoring(), &change{})
			r.reevaluate(j, sc, mode)
			sc.release()
			return
		}
		sc.release()
	}
	fresh := s.process(vi, slices.Clone(r.extras))
	r.cur[j], r.mine[j], r.seen[j] = fresh, true, len(r.extras)
	sc := s.scorer(fresh, v)
	r.reevaluate(j, sc, fromLive)
	sc.release()
}

func (r *suitRanker) repriceMembers(j int) {
	s := r.s
	st := r.catchUp(j)
	sc := &scorer{s: s, v: &s.versions[r.vs[j]], st: st}
	list := r.list[j]
	for k, e := range list {
		if r.picked[e.u] || !r.hasMember(st, e) {
			continue
		}
		now, _, _ := sc.unitGain(e.u, e.alive, r.taken)
		r.set(j, e.u, e.pts, now)
		list[k].pts = now
	}
}

func (r *suitRanker) hasMember(st *stage, e unitPts) bool {
	for _, i := range r.s.units[e.u].pieces {
		if !r.taken[i] && st.isMember(i) {
			return true
		}
	}
	return false
}

func (r *suitRanker) keepListed(j int, sc *scorer) {
	s := r.s
	list := r.list[j]
	kept := make([]unitPts, 0, len(list))
	var dormant []unitPts
	for _, e := range list {
		if r.picked[e.u] {
			r.set(j, e.u, e.pts, 0)
			continue
		}
		gain, alive, live := sc.unitGain(e.u, e.alive, r.taken)
		if live {
			r.set(j, e.u, e.pts, gain)
			kept = append(kept, unitPts{e.u, gain, alive})
			continue
		}
		r.set(j, e.u, e.pts, 0)
		if r.scanned[j] && s.skills {
			if mask := sc.liftAlive(e.u, r.taken); mask != 0 {
				dormant = append(dormant, unitPts{e.u, 0, mask})
			}
		}
	}
	r.list[j] = kept
	if len(dormant) > 0 {
		r.dormant[j] = mergeUnits(nil, r.dormant[j], dormant)
	}
}

func (r *suitRanker) reevaluate(j int, sc *scorer, mode int) {
	if mode == keepListed {
		r.keepListed(j, sc)
		return
	}
	s := r.s
	old := r.list[j]
	cands := r.cands[:0]
	switch mode {
	case rescanAll:
		if !r.scanned[j] {
			r.dormant[j] = r.dormantOf(sc, old)
			r.scanned[j] = true
		}
		for _, e := range old {
			cands = append(cands, unitPts{e.u, e.pts, everyPiece})
		}
		for _, d := range r.dormant[j] {
			if !r.picked[d.u] && sc.beatsK(d.u, d.alive, r.taken) {
				cands = append(cands, d)
			}
		}
		slices.SortFunc(cands, func(a, b unitPts) int { return int(a.u - b.u) })
	default:
		for _, i := range sc.st.live {
			for _, u := range s.unitsOf[i] {
				if r.admit[u] && !r.picked[u] {
					cands = append(cands, unitPts{u: u, alive: everyPiece})
				}
			}
		}
		slices.SortFunc(cands, func(a, b unitPts) int { return int(a.u - b.u) })
		cands = slices.CompactFunc(cands, func(a, b unitPts) bool { return a.u == b.u })
		r.scanned[j], r.dormant[j] = false, nil
	}
	r.cands = cands
	next := make([]unitPts, 0, len(cands))
	var dormant []unitPts
	for _, c := range cands {
		if r.picked[c.u] {
			continue
		}
		gain, alive, live := sc.unitGain(c.u, c.alive, r.taken)
		switch {
		case live:
			next = append(next, unitPts{c.u, gain, alive})
		case r.scanned[j] && s.skills:
			if mask := sc.liftAlive(c.u, r.taken); mask != 0 {
				dormant = append(dormant, unitPts{c.u, 0, mask})
			}
		}
	}
	if r.scanned[j] && (len(dormant) > 0 || mode == rescanAll) {
		kept := r.dormant[j][:0]
		k := 0
		for _, d := range r.dormant[j] {
			for k < len(next) && next[k].u < d.u {
				k++
			}
			if k < len(next) && next[k].u == d.u || r.picked[d.u] {
				continue
			}
			if x, found := slices.BinarySearchFunc(dormant, d.u, func(e unitPts, u int32) int { return int(e.u - u) }); found {
				d = dormant[x]
				dormant = slices.Delete(dormant, x, x+1)
			}
			kept = append(kept, d)
		}
		if len(dormant) > 0 {
			kept = mergeUnits(nil, kept, dormant)
		}
		r.dormant[j] = kept
	}
	a, b := 0, 0
	for a < len(old) || b < len(next) {
		switch {
		case b == len(next) || a < len(old) && old[a].u < next[b].u:
			r.set(j, old[a].u, old[a].pts, 0)
			a++
		case a == len(old) || next[b].u < old[a].u:
			r.set(j, next[b].u, 0, next[b].pts)
			b++
		default:
			r.set(j, old[a].u, old[a].pts, next[b].pts)
			a++
			b++
		}
	}
	r.list[j] = next
}

func (r *suitRanker) dormantOf(sc *scorer, listed []unitPts) []unitPts {
	s := r.s
	var out []unitPts
	k := 0
	for u := range int32(len(s.units)) {
		for k < len(listed) && listed[k].u < u {
			k++
		}
		if !r.admit[u] || r.picked[u] || k < len(listed) && listed[k].u == u {
			continue
		}
		if mask := sc.liftAlive(u, r.taken); mask != 0 {
			out = append(out, unitPts{u: u, alive: mask})
		}
	}
	return out
}

func (sc *scorer) liftAlive(u int32, taken []bool) uint64 {
	s, st, l := sc.s, sc.st, sc.s.l
	var mask uint64
	for k, i := range s.units[u].pieces {
		if taken[i] {
			continue
		}
		if k >= 64 {
			return everyPiece
		}
		at := l.at[i]
		v, f := l.value(st.cU, i)
		if sc.mightBeat(sc.evU, st.cK, at, v, f) {
			mask |= 1 << k
		}
	}
	return mask
}

func (sc *scorer) beatsK(u int32, mask uint64, taken []bool) bool {
	s, st, l := sc.s, sc.st, sc.s.l
	for k, i := range s.units[u].pieces {
		if k < 64 && mask&(1<<k) == 0 || taken[i] {
			continue
		}
		at := l.at[i]
		v, f := l.value(st.cU, i)
		if !sc.mightBeat(sc.evK, st.cK, at, v, f) {
			continue
		}
		v, f = l.value(st.cK, i)
		if c := single(at, i, v, f); sc.beats(sc.evK, &c) {
			return true
		}
	}
	return false
}

func mergeUnits(out, a, b []unitPts) []unitPts {
	for len(a) > 0 || len(b) > 0 {
		switch {
		case len(b) == 0 || len(a) > 0 && a[0].u < b[0].u:
			out, a = append(out, a[0]), a[1:]
		case len(a) == 0 || b[0].u < a[0].u:
			out, b = append(out, b[0]), b[1:]
		default:
			out, a, b = append(out, a[0]), a[1:], b[1:]
		}
	}
	return out
}
