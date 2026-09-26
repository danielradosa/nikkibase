package worth

import (
	"github.com/danielradosa/nikkibase/core/scoring"
)

type placer struct {
	ready  bool
	branch int
	chosen int
	worn   int
	dress  bool
	main   [5]float64
	acc    [5]float64
	place  scoring.Placement
}

func (sc *scorer) exactPlace(c *change) scoring.Placement {
	return sc.s.effective(scoring.Place(sc.outfit(c), sc.v.Stage))
}

func (sc *scorer) placeWith(c *change) scoring.Placement {
	if c.n == 0 {
		return sc.exactPlace(c)
	}
	pl := &sc.pl
	if !pl.ready {
		sc.setupPlacer()
	}
	if len(sc.evU) > 1 {
		chosen, best := 0, 0
		for b, e := range sc.evU {
			if score := scoring.Floor(e.with(c)); b == 0 || score > best {
				chosen, best = b, score
			}
		}
		if chosen != pl.branch {
			return sc.exactPlace(c)
		}
	}
	e := sc.evU[pl.branch]
	l := sc.s.l
	var dm, da [5]float64
	fixed := e.nonAcc + e.torso
	var a int32 = -1
	at := c.at[0]
	switch k := l.kinds[at]; {
	case c.n == 1 && k == plainKind:
		v := c.s[0] + c.f[0]
		if e.br.blocked[at] || !first(v, c.item[0], e.sd.best[at], e.sd.item[at]) {
			return pl.place
		}
		sc.shift(&dm, c.item[0], 1)
		sc.shift(&dm, e.sd.item[at], -1)
		fixed = e.plainWith(at, v) + e.torso
	case c.n == 1 && k == accessoryKind:
		if !e.displaces(at, c.item[0], c.s[0], c.f[0]) {
			return pl.place
		}
		a = l.accOf[at]
		e.offer(c.s[0], c.f[0])
	default:
		d, has, top, bottom := e.dress, e.hasDress, e.top, e.bottom
		dressItem, topItem, bottomItem := e.dressItem, e.topItem, e.bottomItem
		for j := range c.n {
			at := c.at[j]
			if e.br.blocked[at] {
				continue
			}
			v := c.s[j] + c.f[j]
			switch l.kinds[at] {
			case dressKind:
				if !has || first(v, c.item[j], d, dressItem) {
					d, has, dressItem = v, true, c.item[j]
				}
			case topKind:
				if first(v, c.item[j], top, topItem) {
					top, topItem = v, c.item[j]
				}
			case bottomKind:
				if first(v, c.item[j], bottom, bottomItem) {
					bottom, bottomItem = v, c.item[j]
				}
			default:
				return sc.exactPlace(c)
			}
		}
		if dressItem == e.dressItem && topItem == e.topItem && bottomItem == e.bottomItem {
			return pl.place
		}
		if pl.dress {
			sc.shift(&dm, e.dressItem, -1)
		} else {
			sc.shift(&dm, e.topItem, -1)
			sc.shift(&dm, e.bottomItem, -1)
		}
		if has && d >= top+bottom {
			sc.shift(&dm, dressItem, 1)
		} else {
			sc.shift(&dm, topItem, 1)
			sc.shift(&dm, bottomItem, 1)
		}
		fixed = e.nonAcc + torsoOf(d, has, top, bottom)
	}
	if _, chosen, worn := e.settle(fixed, a); chosen != pl.chosen || worn != pl.worn {
		return sc.exactPlace(c)
	}
	if a >= 0 && !sc.accessoryShift(e, a, c.item[0], &da) {
		return sc.exactPlace(c)
	}
	ratio := scoring.AccessoryPenalty(pl.worn)
	var points [5]float64
	for p := range points {
		points[p] = pl.main[p] + dm[p] + ratio*(pl.acc[p]+da[p])
	}
	placed, ok := settled(&points, &sc.v.Stage)
	if !ok {
		return sc.exactPlace(c)
	}
	return sc.s.effective(placed)
}

func (sc *scorer) setupPlacer() {
	pl := &sc.pl
	pl.ready = true
	pl.branch = 0
	for b, e := range sc.evU {
		if e.score > sc.evU[pl.branch].score {
			pl.branch = b
		}
	}
	e := sc.evU[pl.branch]
	pl.chosen, pl.worn = e.accChoice, e.choices[e.accChoice].worn
	pl.dress = e.hasDress && e.dress >= e.top+e.bottom
	items := e.outfit(sc.s.buf, &change{})
	sc.s.buf = items
	st := sc.v.Stage
	pl.main, pl.acc = [5]float64{}, [5]float64{}
	for _, it := range items {
		sum := &pl.main
		if it.Slot == scoring.Accessory {
			sum = &pl.acc
		}
		for p := range 5 {
			if it.Attrs[p] == st.Attrs[p] {
				sum[p] += st.Weights[p] * float64(it.Stats[p])
			}
		}
	}
	pl.place = sc.s.effective(scoring.Place(items, st))
}

func (sc *scorer) shift(sum *[5]float64, i int32, sign float64) {
	if i < 0 {
		return
	}
	l, w := sc.s.l, &sc.st.cU.w
	code, stats := &l.code[i], &l.stats[i]
	for p := range 5 {
		sum[p] += sign * (w[p][code[p]] * stats[p])
	}
}

func (sc *scorer) accessoryShift(e *evaluator, a, item int32, da *[5]float64) bool {
	pl := &sc.pl
	ch := &e.choices[pl.chosen]
	if ch.slot[a] < 0 {
		return true
	}
	nr, na, stride := len(ratios), len(e.l.accAt), len(ch.members)
	k := ratioOf[pl.worn]
	n := pl.worn - len(ch.req)
	nv := e.offered(k)
	vals := ch.sorted[k*stride:]
	order := ch.order[k*stride:]
	displace := func() bool {
		if n == 0 {
			return true
		}
		switch last := vals[n-1]; {
		case nv < last:
			return true
		case nv == last:
			return false
		}
		sc.shift(da, item, 1)
		sc.shift(da, e.sd.pickItem[int(order[n-1])*nr+k], -1)
		return true
	}
	if e.sd.pickItem[int(a)*nr] < 0 {
		return displace()
	}
	if !first(nv, item, e.sd.pick[int(a)*nr+k], e.sd.pickItem[int(a)*nr+k]) {
		return true
	}
	if int(ch.rank[k*na+int(a)]) < n {
		sc.shift(da, item, 1)
		sc.shift(da, e.sd.pickItem[int(a)*nr+k], -1)
		return true
	}
	return displace()
}

func settled(points *[5]float64, st *scoring.Stage) (scoring.Placement, bool) {
	first, second, third := -1, -1, -1
	scale := 0.0
	for p, v := range points {
		if st.Weights[p] <= 0 {
			continue
		}
		if v > scale {
			scale = v
		} else if -v > scale {
			scale = -v
		}
		switch {
		case first < 0 || v > points[first]:
			first, second, third = p, first, second
		case second < 0 || v > points[second]:
			second, third = p, second
		case third < 0 || v > points[third]:
			third = p
		}
	}
	tol := 1e-9 * (1 + scale)
	if second >= 0 && points[first]-points[second] <= tol || third >= 0 && points[second]-points[third] <= tol {
		return scoring.Placement{}, false
	}
	placed := scoring.Placement{CharmSmile: -1, Smile: -1}
	if first >= 0 {
		placed.CharmSmile = int(st.Attrs[first])
	}
	if second >= 0 {
		placed.Smile = int(st.Attrs[second])
	}
	return placed, true
}
