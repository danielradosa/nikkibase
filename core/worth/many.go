package worth

import (
	"math"

	"github.com/danielradosa/nikkibase/core/scoring"
)

type piece struct {
	at   int32
	item int32
	s, f float64
}

var spanLo, spanHi = spans()

func spans() ([]int, []int) {
	lo, hi := make([]int, len(ratios)), make([]int, len(ratios))
	for k := range ratios {
		lo[k] = -1
	}
	for worn, k := range ratioOf {
		if lo[k] < 0 {
			lo[k] = worn
		}
		hi[k] = worn
	}
	return lo, hi
}

type merge struct {
	places []int32
	mark   []int32
	val    []float64
	item   []int32
	chg    []bool
	have   []bool
	add    []int32
	rough  []float64
	done   []bool
	offs   []piece
}

func (e *evaluator) withMany(ps []piece) (float64, int, int) {
	l, sd := e.l, e.sd
	plain, torso, acc := false, false, false
	d, has, top, bottom := e.dress, e.hasDress, e.top, e.bottom
	for _, p := range ps {
		if e.br.blocked[p.at] {
			continue
		}
		v := p.s + p.f
		switch l.kinds[p.at] {
		case plainKind:
			plain = plain || v > sd.best[p.at]
		case dressKind:
			if !has || v > d {
				d, has, torso = v, true, true
			}
		case topKind:
			if v > top {
				top, torso = v, true
			}
		case bottomKind:
			if v > bottom {
				bottom, torso = v, true
			}
		case accessoryKind:
			acc = acc || e.beats(p.at, p.s, p.f)
		}
	}
	if !plain && !torso && !acc {
		return e.total, e.accChoice, e.choices[e.accChoice].worn
	}
	nonAcc := e.nonAcc
	if plain {
		nonAcc = 0
		for _, at := range l.plainAt {
			v := sd.best[at]
			if !e.br.blocked[at] {
				for _, p := range ps {
					if p.at == at && p.s+p.f > v {
						v = p.s + p.f
					}
				}
			}
			nonAcc += v
		}
	}
	t := e.torso
	if torso {
		t = torsoOf(d, has, top, bottom)
	}
	if !acc {
		return e.settle(nonAcc+t, -1)
	}
	return e.settleMany(nonAcc+t, ps)
}

func (e *evaluator) settleMany(fixed float64, ps []piece) (float64, int, int) {
	if cap(e.sums) < len(e.choices) {
		e.sums, e.worns = make([]float64, len(e.choices)), make([]int, len(e.choices))
	}
	sums, worns := e.sums[:len(e.choices)], e.worns[:len(e.choices)]
	chosen := -1
	for c := range e.choices {
		t, w := e.choiceMany(&e.choices[c], ps)
		sums[c], worns[c] = fixed+t, w
		if chosen < 0 || sums[c] > sums[chosen] {
			chosen = c
		}
	}
	for c, sum := range sums {
		if c != chosen && sum > sums[chosen]-2 {
			return e.floored(sums, worns)
		}
	}
	return sums[chosen], chosen, worns[chosen]
}

func (e *evaluator) gather(ch *choice, ps []piece, tie bool) int {
	l, sd, mg := e.l, e.sd, &e.mg
	nr := len(ratios)
	mg.offs, mg.places = mg.offs[:0], mg.places[:0]
	if len(mg.mark) < len(l.accAt) {
		mg.mark = make([]int32, len(l.accAt))
	}
	for _, p := range ps {
		if l.kinds[p.at] != accessoryKind || e.br.blocked[p.at] {
			continue
		}
		a := l.accOf[p.at]
		if ch.slot[a] < 0 {
			continue
		}
		if tie && !e.displaces(p.at, p.item, p.s, p.f) || !tie && !e.beats(p.at, p.s, p.f) {
			continue
		}
		mg.offs = append(mg.offs, p)
		if mg.mark[a] == 0 {
			mg.places = append(mg.places, a)
			mg.mark[a] = int32(len(mg.places))
		}
	}
	np := len(mg.places)
	mg.val, mg.item, mg.chg = grow(mg.val, np*nr), grow(mg.item, np*nr), grow(mg.chg, np*nr)
	mg.have = grow(mg.have, nr)
	clear(mg.have)
	absent := 0
	for _, a := range mg.places {
		if sd.pickItem[int(a)*nr] < 0 {
			absent++
		}
	}
	return absent
}

func (e *evaluator) fillRatio(k int) {
	l, sd, mg := e.l, e.sd, &e.mg
	if mg.have[k] {
		return
	}
	mg.have[k] = true
	nr, r := len(ratios), ratios[k]
	for x, a := range mg.places {
		from := int(a)*nr + k
		v, it, chg := sd.pick[from], sd.pickItem[from], false
		for _, p := range mg.offs {
			if l.accOf[p.at] != a {
				continue
			}
			if ov := r*p.s + p.f; it < 0 || first(ov, p.item, v, it) {
				v, it, chg = ov, p.item, true
			}
		}
		mg.val[x*nr+k], mg.item[x*nr+k], mg.chg[x*nr+k] = v, it, chg
	}
}

func (e *evaluator) unmark() {
	mg := &e.mg
	for _, a := range mg.places {
		mg.mark[a] = 0
	}
	mg.places = mg.places[:0]
}

func (e *evaluator) choiceMany(ch *choice, ps []piece) (float64, int) {
	absent := e.gather(ch, ps, false)
	t, w := e.mergedChoice(ch, absent)
	e.unmark()
	return t, w
}

func (e *evaluator) mergedChoice(ch *choice, absent int) (float64, int) {
	mg := &e.mg
	switch len(mg.offs) {
	case 0:
		return ch.total, ch.worn
	case 1:
		p := mg.offs[0]
		e.offer(p.s, p.f)
		return e.choiceWith(ch, e.l.accOf[p.at])
	}
	nr, stride := len(ratios), len(ch.members)
	m, r := len(ch.opt), len(ch.req)
	most := min(r+m+absent, limit)
	sumS, sumF, rough := 0.0, 0.0, true
	for _, a := range mg.places {
		ms, mf := math.Inf(-1), math.Inf(-1)
		for _, p := range mg.offs {
			if e.l.accOf[p.at] == a {
				if p.s > ms {
					ms = p.s
				}
				if p.f > mf {
					mf = p.f
				}
			}
		}
		rough = rough && ms >= 0 && mf >= 0 && (e.sd.pickItem[int(a)*nr] < 0 || e.minPick[a] >= 0)
		sumS, sumF = sumS+ms, sumF+mf
	}
	mg.rough, mg.done = grow(mg.rough, nr), grow(mg.done, nr)
	for k := range nr {
		lo, hi := max(r, spanLo[k]), min(most, spanHi[k])
		mg.done[k] = lo > hi
		if !mg.done[k] {
			mg.rough[k] = ch.peak[k*(stride+1)+min(hi-r, m)] + ratios[k]*sumS + sumF
			if !rough {
				mg.rough[k] = math.Inf(1)
			}
		}
	}
	best, worn := math.Inf(-1), -1
	if ch.worn >= 0 && e.stays(ch, ratioOf[ch.worn], ch.worn-r) {
		best, worn = ch.total, ch.worn
		k := ratioOf[ch.worn]
		mg.done[k] = true
		if hi := min(most, spanHi[k]); worn < hi {
			if ub := ch.peak[k*(stride+1)+min(hi-r, m)] + e.increase(k); ub+1e-9*(1+math.Abs(ub)) >= best {
				if t, n := e.mergedBest(ch, k, worn+1, hi); t > best {
					best, worn = t, n
				}
			}
		}
	}
	for {
		k := -1
		for c := range nr {
			if !mg.done[c] && (k < 0 || mg.rough[c] > mg.rough[k]) {
				k = c
			}
		}
		if k < 0 || mg.rough[k]+1e-9*(1+math.Abs(mg.rough[k])) < best {
			break
		}
		mg.done[k] = true
		lo, hi := max(r, spanLo[k]), min(most, spanHi[k])
		e.fillRatio(k)
		if ub := ch.peak[k*(stride+1)+min(hi-r, m)] + e.increase(k); ub+1e-9*(1+math.Abs(ub)) < best {
			continue
		}
		t, n := e.mergedBest(ch, k, lo, hi)
		if t > best || t == best && n < worn {
			best, worn = t, n
		}
	}
	return best, worn
}

func (e *evaluator) increase(k int) float64 {
	mg, sd := &e.mg, e.sd
	nr := len(ratios)
	inc := 0.0
	for x, a := range mg.places {
		if !mg.chg[x*nr+k] {
			continue
		}
		old := 0.0
		if sd.pickItem[int(a)*nr] >= 0 {
			old = sd.pick[int(a)*nr+k]
		}
		if d := mg.val[x*nr+k] - old; d > 0 {
			inc += d
		}
	}
	return inc
}

func (e *evaluator) stays(ch *choice, k, cut int) bool {
	mg, sd := &e.mg, e.sd
	e.fillRatio(k)
	nr, na, stride := len(ratios), len(e.l.accAt), len(ch.members)
	var lastV float64
	var lastSlot int32
	if cut > 0 {
		lastV, lastSlot = ch.sorted[k*stride+cut-1], ch.slot[ch.order[k*stride+cut-1]]
	}
	for x, a := range mg.places {
		if !mg.chg[x*nr+k] {
			continue
		}
		v := mg.val[x*nr+k]
		if sd.pickItem[int(a)*nr] >= 0 && int(ch.rank[k*na+int(a)]) < cut {
			if v != sd.pick[int(a)*nr+k] {
				return false
			}
			continue
		}
		if cut > 0 && before(v, ch.slot[a], lastV, lastSlot) {
			return false
		}
	}
	return true
}

func (e *evaluator) addOrder(ch *choice, k int) []int32 {
	mg := &e.mg
	nr := len(ratios)
	add := mg.add[:0]
	for x := range mg.places {
		if !mg.chg[x*nr+k] {
			continue
		}
		v, s := mg.val[x*nr+k], ch.slot[mg.places[x]]
		y := len(add)
		add = append(add, int32(x))
		for ; y > 0; y-- {
			o := add[y-1]
			w := mg.val[int(o)*nr+k]
			if w > v || w == v && ch.slot[mg.places[o]] < s {
				break
			}
			add[y] = o
		}
		add[y] = int32(x)
	}
	mg.add = add
	return add
}

func (e *evaluator) mergedBest(ch *choice, k, lo, hi int) (float64, int) {
	mg, sd := &e.mg, e.sd
	nr, stride := len(ratios), len(ch.members)
	m, r := len(ch.opt), len(ch.req)
	add := e.addOrder(ch, k)
	old := ch.order[k*stride : k*stride+m]
	i, j := 0, 0
	run, taken := ch.reqSum[k], 0
	best, worn := math.Inf(-1), -1
	for n := lo; n <= hi; n++ {
		for taken < n-r {
			for i < m {
				if x := mg.mark[old[i]]; x == 0 || !mg.chg[int(x-1)*nr+k] {
					break
				}
				i++
			}
			switch {
			case i < m && (j == len(add) || before(sd.pick[int(old[i])*nr+k], ch.slot[old[i]], mg.val[int(add[j])*nr+k], ch.slot[mg.places[add[j]]])):
				run += sd.pick[int(old[i])*nr+k]
				i++
			case j < len(add):
				run += mg.val[int(add[j])*nr+k]
				j++
			default:
				return best, worn
			}
			taken++
		}
		if run > best {
			best, worn = run, n
		}
	}
	return best, worn
}

func before(v float64, slot int32, w float64, other int32) bool {
	return v > w || v == w && slot < other
}

func (e *evaluator) outfitMany(buf []scoring.Item, ps []piece, chosen, worn int) []scoring.Item {
	l, sd := e.l, e.sd
	out := buf[:0]
	for _, at := range l.plainAt {
		v, i := sd.best[at], sd.item[at]
		if !e.br.blocked[at] {
			for _, p := range ps {
				if p.at == at && first(p.s+p.f, p.item, v, i) {
					v, i = p.s+p.f, p.item
				}
			}
		}
		if i >= 0 {
			out = append(out, l.items[i])
		}
	}
	d, has, top, bottom := e.dress, e.hasDress, e.top, e.bottom
	dressItem, topItem, bottomItem := e.dressItem, e.topItem, e.bottomItem
	for _, p := range ps {
		if e.br.blocked[p.at] {
			continue
		}
		v := p.s + p.f
		switch l.kinds[p.at] {
		case dressKind:
			if !has || first(v, p.item, d, dressItem) {
				d, has, dressItem = v, true, p.item
			}
		case topKind:
			if first(v, p.item, top, topItem) {
				top, topItem = v, p.item
			}
		case bottomKind:
			if first(v, p.item, bottom, bottomItem) {
				bottom, bottomItem = v, p.item
			}
		}
	}
	if has && d >= top+bottom {
		out = append(out, l.items[dressItem])
	} else {
		if topItem >= 0 {
			out = append(out, l.items[topItem])
		}
		if bottomItem >= 0 {
			out = append(out, l.items[bottomItem])
		}
	}
	if chosen < 0 || worn < 0 {
		return out
	}
	ch := &e.choices[chosen]
	nr := len(ratios)
	k := ratioOf[worn]
	for _, r := range ch.req {
		out = append(out, l.items[sd.pickItem[int(r)*nr+k]])
	}
	e.gather(ch, ps, true)
	out = e.wornMerged(out, ch, k, worn-len(ch.req))
	e.unmark()
	return out
}

func (e *evaluator) wornMerged(out []scoring.Item, ch *choice, k, n int) []scoring.Item {
	l, sd, mg := e.l, e.sd, &e.mg
	nr, stride := len(ratios), len(ch.members)
	e.fillRatio(k)
	add := e.addOrder(ch, k)
	old := ch.order[k*stride : k*stride+len(ch.opt)]
	i, j := 0, 0
	for ; n > 0; n-- {
		for i < len(old) {
			if x := mg.mark[old[i]]; x == 0 || !mg.chg[int(x-1)*nr+k] {
				break
			}
			i++
		}
		switch {
		case i < len(old) && (j == len(add) || before(sd.pick[int(old[i])*nr+k], ch.slot[old[i]], mg.val[int(add[j])*nr+k], ch.slot[mg.places[add[j]]])):
			out = append(out, l.items[sd.pickItem[int(old[i])*nr+k]])
			i++
		case j < len(add):
			out = append(out, l.items[mg.item[int(add[j])*nr+k]])
			j++
		default:
			return out
		}
	}
	return out
}
