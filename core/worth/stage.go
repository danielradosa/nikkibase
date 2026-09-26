package worth

import (
	"slices"

	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
)

type stage struct {
	failing  bool
	missing  [][]int
	extras   []int32
	added    []int32
	branches []*branch
	cU, cK   *coef
	u, k     []*side
	place    scoring.Placement
	placed   map[scoring.Placement]*placed
	base     int
	members  []int32
	reach    []reach
	live     []int32
	pts      []int32
	vals     []float64
}

type placed struct {
	c     *coef
	sides []*side
}

type reach struct {
	p  scoring.Placement
	c  *coef
	sd *side
}

func (st *stage) clone(light bool) *stage {
	c := *st
	c.u, c.k = cloneSides(st.u), cloneSides(st.k)
	if st.placed != nil {
		c.placed = make(map[scoring.Placement]*placed, len(st.placed))
		for p, pl := range st.placed {
			c.placed[p] = &placed{pl.c, cloneSides(pl.sides)}
		}
	}
	c.added = slices.Clone(st.added)
	if st.reach != nil {
		c.reach = make([]reach, len(st.reach))
		for k, rc := range st.reach {
			c.reach[k] = reach{rc.p, rc.c, rc.sd.clone()}
		}
	}
	if light {
		c.live, c.pts, c.vals = nil, nil, nil
	} else {
		c.live, c.pts = slices.Clone(st.live), slices.Clone(st.pts)
	}
	return &c
}

func cloneSides(sides []*side) []*side {
	if sides == nil {
		return nil
	}
	out := make([]*side, len(sides))
	for i, sd := range sides {
		out[i] = sd.clone()
	}
	return out
}

func (st *stage) isMember(i int32) bool {
	_, ok := slices.BinarySearch(st.members, i)
	return ok
}

func (st *stage) points(i int32) int32 {
	if k, ok := slices.BinarySearch(st.live, i); ok {
		return st.pts[k]
	}
	return 0
}

func (s *Session) process(vi int, extras []int32) *stage {
	for _, i := range extras {
		s.taken[i] = true
	}
	st := s.examine(vi, extras)
	for _, i := range extras {
		s.taken[i] = false
	}
	return st
}

func (s *Session) examine(vi int, extras []int32) *stage {
	l, v := s.l, &s.versions[vi]
	st := &stage{extras: extras}
	switch {
	case len(v.Require) > 0:
		space := optimizer.Require(l.positionsWith(extras), l.posOf, v.Require)
		base := l.poolWith(extras)
		for _, b := range space.Branches() {
			st.branches = append(st.branches, l.branchOf(b, base))
		}
		st.members = s.membersOf(v)
	case len(extras) == 0:
		st.branches = []*branch{l.plain}
	default:
		st.branches = []*branch{l.newBranch(l.poolWith(extras), nil)}
	}

	st.cU = l.coefOf(v.Stage, nil)
	st.u = s.sides(st.branches, st.cU, nil)
	sc := s.scorer(st, v)
	if len(v.Require) > 0 {
		if unmet := optimizer.Unmet(sc.outfit(&change{}), v.Require); len(unmet) > 0 {
			st.failing = true
			for _, set := range unmet {
				missing := []int{}
				for _, id := range set {
					if !s.owns(id) {
						missing = append(missing, id)
					}
				}
				st.missing = append(st.missing, missing)
			}
			sc.release()
			return st
		}
	}
	if s.skills {
		st.place = sc.placeWith(&change{})
		st.cK = l.coefOf(v.Stage, st.place.SkillsAt(s.levels))
		st.k = s.sides(st.branches, st.cK, nil)
		sc.evK = s.evaluators(st.branches, st.k)
	}
	st.base = bestOf(sc.scoring(), &change{})
	if len(st.members) > 0 {
		sc.reachFor(scoring.Placement{CharmSmile: -1, Smile: -1})
	}
	s.scan(sc)
	sc.release()
	return st
}

func (s *Session) membersOf(v *Version) []int32 {
	var out []int32
	for _, set := range v.Require {
		for _, id := range set {
			i, ok := s.l.index[id]
			if !ok || s.l.owned[i] || s.taken[i] {
				continue
			}
			if k, found := slices.BinarySearch(out, i); !found {
				out = slices.Insert(out, k, i)
			}
		}
	}
	return out
}

func (s *Session) sides(brs []*branch, c *coef, added []int32) []*side {
	out := make([]*side, len(brs))
	for b, br := range brs {
		sd := s.l.newSide()
		s.l.fill(sd, br.pool, c)
		for _, i := range added {
			s.l.addTo(sd, br, c, i)
		}
		out[b] = sd
	}
	return out
}

func (l *layout) addTo(sd *side, br *branch, c *coef, i int32) {
	if br.blocked[l.at[i]] {
		return
	}
	s, f := l.value(c, i)
	l.put(sd, i, s, f)
}

type scorer struct {
	s     *Session
	v     *Version
	st    *stage
	evU   []*evaluator
	evK   []*evaluator
	extra map[scoring.Placement][]*evaluator
	pl    placer
}

func (s *Session) scorer(st *stage, v *Version) *scorer {
	sc := &scorer{s: s, v: v, st: st}
	sc.evU = s.evaluators(st.branches, st.u)
	if st.k != nil {
		sc.evK = s.evaluators(st.branches, st.k)
	}
	return sc
}

func (s *Session) evaluators(brs []*branch, sides []*side) []*evaluator {
	out := make([]*evaluator, len(brs))
	for b, br := range brs {
		e := s.acquire()
		e.build(br, sides[b])
		out[b] = e
	}
	return out
}

func (sc *scorer) release() {
	sc.s.free = append(sc.s.free, sc.evU...)
	sc.s.free = append(sc.s.free, sc.evK...)
	for _, evs := range sc.extra {
		sc.s.free = append(sc.s.free, evs...)
	}
	sc.evU, sc.evK, sc.extra = nil, nil, nil
}

func (sc *scorer) scoring() []*evaluator {
	if sc.evK != nil {
		return sc.evK
	}
	return sc.evU
}

func bestOf(evs []*evaluator, c *change) int {
	best := 0
	for b, e := range evs {
		if score := scoring.Floor(e.with(c)); b == 0 || score > best {
			best = score
		}
	}
	return best
}

func (sc *scorer) beats(evs []*evaluator, c *change) bool {
	if len(evs) == 1 && c.n == 1 {
		return evs[0].beats(c.at[0], c.s[0], c.f[0])
	}
	for _, e := range evs {
		for j := range c.n {
			if e.beats(c.at[j], c.s[j], c.f[j]) {
				return true
			}
		}
	}
	return false
}

func (sc *scorer) liveU(c *change) bool {
	if len(sc.evU) == 1 && c.n == 1 {
		if !sc.s.skills {
			return sc.evU[0].beats(c.at[0], c.s[0], c.f[0])
		}
		return sc.evU[0].displaces(c.at[0], c.item[0], c.s[0], c.f[0])
	}
	if !sc.s.skills {
		return sc.beats(sc.evU, c)
	}
	for _, e := range sc.evU {
		for j := range c.n {
			if e.displaces(c.at[j], c.item[j], c.s[j], c.f[j]) {
				return true
			}
		}
	}
	return false
}

func (sc *scorer) mightBeat(evs []*evaluator, c *coef, at int32, s, f float64) bool {
	if !c.plus || s < 0 {
		return true
	}
	bound := c.lift*s*(1+1e-9) + f
	for _, e := range evs {
		if e.br.blocked[at] {
			continue
		}
		if a := e.l.accOf[at]; a >= 0 {
			if bound > e.minPick[a] {
				return true
			}
		} else if bound > e.sd.best[at] {
			return true
		}
	}
	return false
}

func (sc *scorer) closed(at int32) bool {
	for _, br := range sc.st.branches {
		if !br.blocked[at] {
			return false
		}
	}
	return true
}

func (sc *scorer) outfit(c *change) []scoring.Item {
	chosen, best := 0, 0
	for b, e := range sc.evU {
		if score := scoring.Floor(e.with(c)); b == 0 || score > best {
			chosen, best = b, score
		}
	}
	sc.s.buf = sc.evU[chosen].outfit(sc.s.buf, c)
	return sc.s.buf
}

func (sc *scorer) valued(c *change, cf *coef) change {
	out := *c
	for j := range c.n {
		out.s[j], out.f[j] = sc.s.l.value(cf, c.item[j])
	}
	return out
}

func (sc *scorer) gain(cu, ck *change, liveU bool) int32 {
	st := sc.st
	if !sc.s.skills {
		return int32(bestOf(sc.evU, cu) - st.base)
	}
	p := st.place
	if liveU {
		p = sc.placeWith(cu)
	}
	if p == st.place {
		return int32(bestOf(sc.evK, ck) - st.base)
	}
	evs, cf := sc.placedFor(p)
	c := sc.valued(cu, cf)
	return int32(bestOf(evs, &c) - st.base)
}

func (sc *scorer) placedFor(p scoring.Placement) ([]*evaluator, *coef) {
	s, st := sc.s, sc.st
	pl := st.placed[p]
	if pl == nil {
		c := s.l.coefOf(sc.v.Stage, p.SkillsAt(s.levels))
		pl = &placed{c, s.sides(st.branches, c, st.added)}
		if st.placed == nil {
			st.placed = map[scoring.Placement]*placed{}
		}
		st.placed[p] = pl
	}
	evs, ok := sc.extra[p]
	if !ok {
		evs = s.evaluators(st.branches, pl.sides)
		if sc.extra == nil {
			sc.extra = map[scoring.Placement][]*evaluator{}
		}
		sc.extra[p] = evs
	}
	return evs, pl.c
}

func (sc *scorer) changeOf(items []int32) (cu, ck change, liveU, live bool) {
	l, st := sc.s.l, sc.st
	cu.n = len(items)
	for j, i := range items {
		cu.at[j], cu.item[j] = l.at[i], i
		cu.s[j], cu.f[j] = l.value(st.cU, i)
	}
	liveU = sc.liveU(&cu)
	live = liveU
	if sc.s.skills {
		ck = sc.valued(&cu, st.cK)
		live = live || sc.beats(sc.evK, &ck)
	}
	return cu, ck, liveU, live
}

func (sc *scorer) engineGain(items ...int32) int32 {
	gain, _ := sc.enginePlaced(items...)
	return gain
}

func (sc *scorer) memberGain(m int32) int32 {
	gain, p := sc.enginePlaced(m)
	sc.reachFor(p)
	return gain
}

func (sc *scorer) enginePlaced(items ...int32) (int32, scoring.Placement) {
	s, st, v := sc.s, sc.st, sc.v
	extras := append(append(slices.Clone(st.extras), st.added...), items...)
	space := optimizer.Require(s.l.positionsWith(extras), s.l.posOf, v.Require)
	if s.skills {
		r, p := space.BestPlacedAt(v.Stage, s.levels)
		return int32(r.Score - st.base), p
	}
	return int32(space.Best(v.Stage, nil).Score - st.base), scoring.Placement{CharmSmile: -1, Smile: -1}
}

func (sc *scorer) reachFor(p scoring.Placement) {
	s, st, l := sc.s, sc.st, sc.s.l
	p = s.effective(p)
	for _, rc := range st.reach {
		if rc.p == p {
			return
		}
	}
	c := l.coefOf(sc.v.Stage, p.SkillsAt(s.levels))
	sd := l.newSide()
	l.fill(sd, l.poolWith(st.extras), c)
	for _, i := range st.added {
		v, f := l.value(c, i)
		l.put(sd, i, v, f)
	}
	st.reach = append(st.reach, reach{p, c, sd})
}

func (s *Session) switchTo(sc *scorer, p scoring.Placement) {
	l, st := s.l, sc.st
	if st.placed == nil {
		st.placed = map[scoring.Placement]*placed{}
	}
	st.placed[st.place] = &placed{st.cK, st.k}
	pl := st.placed[p]
	if pl == nil {
		c := l.coefOf(sc.v.Stage, p.SkillsAt(s.levels))
		pl = &placed{c, s.sides(st.branches, c, st.added)}
	}
	delete(st.placed, p)
	st.place, st.cK, st.k = p, pl.c, pl.sides
	s.free = append(s.free, sc.evK...)
	for _, evs := range sc.extra {
		s.free = append(s.free, evs...)
	}
	sc.extra = nil
	sc.evK = s.evaluators(st.branches, st.k)
}

func (s *Session) moveTo(sc *scorer, p scoring.Placement, picked []bool) {
	l, st := s.l, sc.st
	s.switchTo(sc, p)

	live := make([]int32, 0, len(st.live))
	pts := make([]int32, 0, len(st.pts))
	vals := make([]float64, 0, len(st.vals))
	next := 0
	for at := range l.kinds {
		at := int32(at)
		if sc.closed(at) {
			continue
		}
		for _, i := range l.candIn[at] {
			for next < len(st.live) && st.live[next] < i {
				next++
			}
			if next < len(st.live) && st.live[next] == i {
				sK, fK := l.value(st.cK, i)
				live, pts = append(live, i), append(pts, st.pts[next])
				vals = append(vals, st.vals[4*next], st.vals[4*next+1], sK, fK)
				continue
			}
			if picked[i] {
				continue
			}
			sK, fK := l.value(st.cK, i)
			if c := single(at, i, sK, fK); sc.beats(sc.evK, &c) {
				sU, fU := l.value(st.cU, i)
				live, pts = append(live, i), append(pts, 0)
				vals = append(vals, sU, fU, sK, fK)
			}
		}
	}
	st.live, st.pts, st.vals = live, pts, vals
}

type best struct {
	value float64
	item  int32
}

func (s *Session) scan(sc *scorer) {
	l, st := s.l, sc.st
	seek := st.extras == nil && len(sc.v.Require) == 0
	var torso [3]best
	for k := range torso {
		torso[k].item = -1
	}
	for at, k := range l.kinds {
		at := int32(at)
		if sc.closed(at) {
			continue
		}
		slot := int(k) - int(dressKind)
		tracked := seek && slot >= 0 && slot < len(torso)
		for _, i := range l.candIn[at] {
			if s.taken[i] || st.isMember(i) {
				continue
			}
			sU, fU := l.value(st.cU, i)
			cu := single(at, i, sU, fU)
			liveU := sc.liveU(&cu)
			live := liveU
			var ck change
			if s.skills && (liveU || tracked || sc.mightBeat(sc.evK, st.cK, at, sU, fU)) {
				sK, fK := l.value(st.cK, i)
				ck = single(at, i, sK, fK)
				live = live || sc.beats(sc.evK, &ck)
				if tracked && (torso[slot].item < 0 || sK+fK > torso[slot].value) {
					torso[slot] = best{sK + fK, i}
				}
			} else if tracked && (torso[slot].item < 0 || sU+fU > torso[slot].value) {
				torso[slot] = best{sU + fU, i}
			}
			if !live {
				continue
			}
			st.live = append(st.live, i)
			st.vals = append(st.vals, sU, fU)
			if s.skills {
				st.vals = append(st.vals, ck.s[0], ck.f[0])
			}
			st.pts = append(st.pts, sc.gain(&cu, &ck, liveU))
		}
	}
	stride := s.stride()
	for _, m := range st.members {
		k, _ := slices.BinarySearch(st.live, m)
		st.live = slices.Insert(st.live, k, m)
		st.pts = slices.Insert(st.pts, k, sc.memberGain(m))
		st.vals = slices.Insert(st.vals, k*stride, make([]float64, stride)...)
	}
	if seek {
		s.discover(sc.scoring()[0], torso)
	}
}

func (s *Session) stride() int {
	if s.skills {
		return 4
	}
	return 2
}

func (st *stage) valued(e int, stride int, at, i int32) (cu, ck change) {
	v := st.vals[e*stride:]
	cu = single(at, i, v[0], v[1])
	if stride == 4 {
		ck = single(at, i, v[2], v[3])
	}
	return cu, ck
}

func (s *Session) discover(e *evaluator, cands [3]best) {
	owned := [3]best{{e.dress, e.dressItem}, {e.top, e.topItem}, {e.bottom, e.bottomItem}}
	var chosen [3]best
	var fromCands [3]bool
	for k := range chosen {
		o, c := owned[k], cands[k]
		switch {
		case c.item < 0:
			chosen[k] = o
		case o.item < 0 || c.value > o.value || c.value == o.value && c.item < o.item:
			chosen[k], fromCands[k] = c, true
		default:
			chosen[k] = o
		}
	}
	dress, top, bottom := chosen[0], chosen[1], chosen[2]
	if !fromCands[1] || !fromCands[2] || dress.item >= 0 && dress.value >= top.value+bottom.value {
		return
	}
	pair := [2]int32{top.item, bottom.item}
	if !s.paired[pair] {
		s.paired[pair] = true
		s.pairs = append(s.pairs, pair)
	}
}
