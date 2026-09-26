package worth

import (
	"math"
	"slices"

	"github.com/danielradosa/nikkibase/core/scoring"
)

type side struct {
	best     []float64
	item     []int32
	pick     []float64
	pickItem []int32
}

func (l *layout) newSide() *side {
	np, na := len(l.kinds), len(l.accAt)*len(ratios)
	return &side{best: make([]float64, np), item: make([]int32, np), pick: make([]float64, na), pickItem: make([]int32, na)}
}

func (sd *side) clone() *side {
	return &side{best: slices.Clone(sd.best), item: slices.Clone(sd.item),
		pick: slices.Clone(sd.pick), pickItem: slices.Clone(sd.pickItem)}
}

func (l *layout) fill(sd *side, pool [][]int32, c *coef) {
	nr := len(ratios)
	for at, items := range pool {
		a := l.accOf[at]
		if a < 0 {
			best, item := 0.0, int32(-1)
			for _, i := range items {
				s, f := l.value(c, i)
				if v := s + f; item < 0 || v > best {
					best, item = v, i
				}
			}
			sd.best[at], sd.item[at] = best, item
			continue
		}
		sd.best[at], sd.item[at] = 0, -1
		row, picks := sd.pick[int(a)*nr:int(a+1)*nr], sd.pickItem[int(a)*nr:int(a+1)*nr]
		clear(row)
		for k := range picks {
			picks[k] = -1
		}
		low := math.Inf(-1)
		for _, i := range items {
			s, f := l.value(c, i)
			if s >= 0 && s+f <= low {
				continue
			}
			changed := false
			for k, r := range ratios {
				if v := r*s + f; picks[k] < 0 || v > row[k] {
					row[k], picks[k] = v, i
					changed = true
				}
			}
			if changed {
				low = slices.Min(row)
			}
		}
	}
}

func (l *layout) put(sd *side, i int32, s, f float64) {
	at := l.at[i]
	if a := l.accOf[at]; a >= 0 {
		from := int(a) * len(ratios)
		row, items := sd.pick[from:from+len(ratios)], sd.pickItem[from:from+len(ratios)]
		for k, r := range ratios {
			if v := r*s + f; items[k] < 0 || first(v, i, row[k], items[k]) {
				row[k], items[k] = v, i
			}
		}
		return
	}
	if v := s + f; sd.item[at] < 0 || first(v, i, sd.best[at], sd.item[at]) {
		sd.best[at], sd.item[at] = v, i
	}
}

func first(v float64, i int32, best float64, item int32) bool {
	return v > best || v == best && item >= 0 && i < item
}

type change struct {
	n    int
	at   [2]int32
	item [2]int32
	s, f [2]float64
}

func single(at, i int32, s, f float64) change {
	return change{n: 1, at: [2]int32{at}, item: [2]int32{i}, s: [2]float64{s}, f: [2]float64{f}}
}

type choice struct {
	members []int32
	slot    []int32
	req     []int32
	opt     []int32
	reqSum  []float64
	order   []int32
	sorted  []float64
	prefix  []float64
	peak    []float64
	rank    []int32
	total   float64
	worn    int
	edge    float64
	gen     uint32
	lineGen []uint32
	lineN   []int32
	icept   []float64
	slope   []float64
	lineW   []int32
}

type evaluator struct {
	l          *layout
	br         *branch
	sd         *side
	nonAcc     float64
	dress      float64
	top        float64
	bottom     float64
	hasDress   bool
	dressItem  int32
	topItem    int32
	bottomItem int32
	torso      float64
	choices    []choice
	acc        float64
	accChoice  int
	total      float64
	score      int
	minPick    []float64
	cs, cf     float64
	sums       []float64
	worns      []int
	vals       []float64
	mg         merge
}

func (l *layout) newEvaluator() *evaluator {
	return &evaluator{l: l, minPick: make([]float64, len(l.accAt))}
}

func torsoOf(dress float64, hasDress bool, top, bottom float64) float64 {
	if hasDress && dress >= top+bottom {
		return dress
	}
	return top + bottom
}

func (e *evaluator) build(br *branch, sd *side) {
	l := e.l
	e.br, e.sd = br, sd
	e.nonAcc = 0
	for _, at := range l.plainAt {
		e.nonAcc += sd.best[at]
	}
	e.dress, e.top, e.bottom = 0, 0, 0
	e.dressItem, e.topItem, e.bottomItem = -1, -1, -1
	for at, k := range l.kinds {
		i, v := sd.item[at], sd.best[at]
		if i < 0 {
			continue
		}
		switch k {
		case dressKind:
			if e.dressItem < 0 || v > e.dress {
				e.dress, e.dressItem = v, i
			}
		case topKind:
			if e.topItem < 0 || v > e.top {
				e.top, e.topItem = v, i
			}
		case bottomKind:
			if e.bottomItem < 0 || v > e.bottom {
				e.bottom, e.bottomItem = v, i
			}
		}
	}
	e.hasDress = e.dressItem >= 0
	e.torso = torsoOf(e.dress, e.hasDress, e.top, e.bottom)

	nr := len(ratios)
	for a := range l.accAt {
		low := sd.pick[a*nr]
		for _, v := range sd.pick[a*nr+1 : (a+1)*nr] {
			low = min(low, v)
		}
		e.minPick[a] = low
	}
	if cap(e.choices) < len(br.choices) {
		e.choices = append(e.choices[:cap(e.choices)], make([]choice, len(br.choices)-cap(e.choices))...)
	}
	e.choices = e.choices[:len(br.choices)]
	for c, members := range br.choices {
		e.fillChoice(&e.choices[c], members)
	}
	e.total, e.accChoice, _ = e.settle(e.nonAcc+e.torso, -1)
	e.acc = e.choices[e.accChoice].total
	e.score = scoring.Floor(e.total)
}

func (e *evaluator) settle(fixed float64, a int32) (total float64, chosen, worn int) {
	if cap(e.sums) < len(e.choices) {
		e.sums, e.worns = make([]float64, len(e.choices)), make([]int, len(e.choices))
	}
	sums, worns := e.sums[:len(e.choices)], e.worns[:len(e.choices)]
	chosen = -1
	for c := range e.choices {
		ch := &e.choices[c]
		t, w := ch.total, ch.worn
		if a >= 0 && ch.slot[a] >= 0 {
			t, w = e.choiceWith(ch, a)
		}
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

func (e *evaluator) floored(sums []float64, worns []int) (total float64, chosen, worn int) {
	best := 0
	chosen = -1
	for c, sum := range sums {
		if score := scoring.Floor(sum); chosen < 0 || score > best {
			best, chosen, total, worn = score, c, sum, worns[c]
		}
	}
	return total, chosen, worn
}

func grow[T any](s []T, n int) []T {
	if cap(s) < n {
		return make([]T, n)
	}
	return s[:n]
}

func (e *evaluator) fillChoice(ch *choice, members []int32) {
	l, sd, br := e.l, e.sd, e.br
	na, nr, stride := len(l.accAt), len(ratios), len(members)
	ch.members = members
	ch.slot = grow(ch.slot, na)
	ch.reqSum = grow(ch.reqSum, nr)
	ch.order = grow(ch.order, nr*stride)
	ch.sorted = grow(ch.sorted, nr*stride)
	ch.prefix = grow(ch.prefix, nr*(stride+1))
	ch.peak = grow(ch.peak, nr*(stride+1))
	ch.rank = grow(ch.rank, nr*na)
	ch.lineGen = grow(ch.lineGen, na)
	ch.lineN = grow(ch.lineN, na)
	ch.icept = grow(ch.icept, nr*na)
	ch.slope = grow(ch.slope, nr*na)
	ch.lineW = grow(ch.lineW, nr*na)
	ch.gen++
	for a := range ch.slot {
		ch.slot[a] = -1
	}
	for i := range ch.rank {
		ch.rank[i] = -1
	}
	ch.req, ch.opt = ch.req[:0], ch.opt[:0]
	for j, a := range members {
		ch.slot[a] = int32(j)
		if sd.pickItem[int(a)*nr] < 0 {
			continue
		}
		if br.required[a] {
			ch.req = append(ch.req, a)
		} else {
			ch.opt = append(ch.opt, a)
		}
	}
	m := len(ch.opt)
	for k := range nr {
		sum := 0.0
		for _, a := range ch.req {
			sum += sd.pick[int(a)*nr+k]
		}
		ch.reqSum[k] = sum
		ord := ch.order[k*stride : k*stride+m]
		if k == 0 {
			copy(ord, ch.opt)
		} else {
			copy(ord, ch.order[(k-1)*stride:(k-1)*stride+m])
		}
		for x := 1; x < m; x++ {
			a := ord[x]
			v := sd.pick[int(a)*nr+k]
			y := x
			for ; y > 0; y-- {
				b := ord[y-1]
				w := sd.pick[int(b)*nr+k]
				if v < w || v == w && ch.slot[a] > ch.slot[b] {
					break
				}
				ord[y] = b
			}
			ord[y] = a
		}
		vals := ch.sorted[k*stride : k*stride+m]
		pre := ch.prefix[k*(stride+1) : k*(stride+1)+m+1]
		peak := ch.peak[k*(stride+1) : k*(stride+1)+m+1]
		pre[0], peak[0] = sum, sum
		for j, a := range ord {
			v := sd.pick[int(a)*nr+k]
			vals[j] = v
			pre[j+1] = pre[j] + v
			peak[j+1] = peak[j]
			if pre[j+1] > peak[j] {
				peak[j+1] = pre[j+1]
			}
			ch.rank[k*na+int(a)] = int32(j)
		}
	}
	r := len(ch.req)
	ch.total, ch.worn = math.Inf(-1), -1
	for worn := r; worn <= min(r+m, limit); worn++ {
		k := ratioOf[worn]
		if t := ch.prefix[k*(stride+1)+worn-r]; t > ch.total {
			ch.total, ch.worn = t, worn
		}
	}
	ch.edge = ch.total + 1e-9*(1+math.Abs(ch.total))
}

func (e *evaluator) beats(at int32, s, f float64) bool {
	if e.br.blocked[at] {
		return false
	}
	a := e.l.accOf[at]
	if a < 0 {
		return s+f > e.sd.best[at]
	}
	if s >= 0 && s+f <= e.minPick[a] {
		return false
	}
	from := int(a) * len(ratios)
	row := e.sd.pick[from : from+len(ratios)]
	for k, r := range ratios {
		if r*s+f > row[k] {
			return true
		}
	}
	return false
}

func (e *evaluator) displaces(at, i int32, s, f float64) bool {
	if e.br.blocked[at] {
		return false
	}
	a := e.l.accOf[at]
	if a < 0 {
		return first(s+f, i, e.sd.best[at], e.sd.item[at])
	}
	if s >= 0 && s+f < e.minPick[a] {
		return false
	}
	from := int(a) * len(ratios)
	for k, r := range ratios {
		if first(r*s+f, i, e.sd.pick[from+k], e.sd.pickItem[from+k]) {
			return true
		}
	}
	return false
}

func (e *evaluator) with(c *change) float64 {
	if c.n == 1 {
		at := c.at[0]
		if e.br.blocked[at] {
			return e.total
		}
		switch e.l.kinds[at] {
		case plainKind:
			return e.withPlain(at, c.s[0]+c.f[0])
		case accessoryKind:
			t, _, _ := e.withAccessory(at, c.s[0], c.f[0])
			return t
		}
	}
	d, has, top, bottom, changed := e.torsoWith(c)
	if !changed {
		return e.total
	}
	t, _, _ := e.settle(e.nonAcc+torsoOf(d, has, top, bottom), -1)
	return t
}

func (e *evaluator) torsoWith(c *change) (dress float64, hasDress bool, top, bottom float64, changed bool) {
	dress, hasDress, top, bottom = e.dress, e.hasDress, e.top, e.bottom
	for j := range c.n {
		at := c.at[j]
		if e.br.blocked[at] {
			continue
		}
		v := c.s[j] + c.f[j]
		switch e.l.kinds[at] {
		case dressKind:
			if !hasDress || v > dress {
				dress, hasDress, changed = v, true, true
			}
		case topKind:
			if v > top {
				top, changed = v, true
			}
		case bottomKind:
			if v > bottom {
				bottom, changed = v, true
			}
		}
	}
	return dress, hasDress, top, bottom, changed
}

func (e *evaluator) withPlain(at int32, v float64) float64 {
	if v <= e.sd.best[at] {
		return e.total
	}
	t, _, _ := e.settle(e.plainWith(at, v)+e.torso, -1)
	return t
}

func (e *evaluator) plainWith(at int32, v float64) float64 {
	nonAcc := 0.0
	for _, p := range e.l.plainAt {
		if p == at {
			nonAcc += v
		} else {
			nonAcc += e.sd.best[p]
		}
	}
	return nonAcc
}

func (e *evaluator) withAccessory(at int32, s, f float64) (float64, int, int) {
	if !e.beats(at, s, f) {
		return e.total, e.accChoice, e.choices[e.accChoice].worn
	}
	e.offer(s, f)
	return e.settle(e.nonAcc+e.torso, e.l.accOf[at])
}

func (e *evaluator) offer(s, f float64) {
	e.cs, e.cf = s, f
}

func (e *evaluator) offered(k int) float64 {
	return ratios[k]*e.cs + e.cf
}

func (e *evaluator) choiceWith(ch *choice, a int32) (float64, int) {
	if ch.lineGen[a] != ch.gen {
		e.lines(ch, a)
	}
	from := int(a) * len(ratios)
	best, second, worn := math.Inf(-1), math.Inf(-1), -1
	for i := range int(ch.lineN[a]) {
		if v := ch.icept[from+i] + ch.slope[from+i]*e.cs; v > best {
			best, second, worn = v, best, int(ch.lineW[from+i])
		} else if v > second {
			second = v
		}
	}
	total := best + e.cf
	if total > ch.edge {
		tol := 1e-9 * (1 + math.Abs(total))
		if second+e.cf >= total-tol || worn != ch.worn && ch.total >= total-tol {
			return e.exactWith(ch, a)
		}
		return total, worn
	}
	rival := best
	if worn == ch.worn {
		rival = second
	}
	if rival+e.cf >= ch.total-1e-9*(1+math.Abs(ch.total)) {
		return e.exactWith(ch, a)
	}
	return ch.total, ch.worn
}

func (e *evaluator) exactWith(ch *choice, a int32) (float64, int) {
	sd, nr := e.sd, len(ratios)
	absent := sd.pickItem[int(a)*nr] < 0
	m := len(ch.opt)
	if absent {
		m++
	}
	e.vals = grow(e.vals, m)
	r := len(ch.req)
	total, worn, last := math.Inf(-1), -1, -1
	for n := r; n <= min(r+m, limit); n++ {
		k := ratioOf[n]
		if k != last {
			last = k
			for j, o := range ch.opt {
				v := sd.pick[int(o)*nr+k]
				if o == a {
					v = max(v, e.offered(k))
				}
				e.vals[j] = v
			}
			if absent {
				e.vals[m-1] = e.offered(k)
			}
			for x := 1; x < m; x++ {
				v, y := e.vals[x], x
				for ; y > 0 && e.vals[y-1] < v; y-- {
					e.vals[y] = e.vals[y-1]
				}
				e.vals[y] = v
			}
		}
		t := ch.reqSum[k]
		for _, v := range e.vals[:n-r] {
			t += v
		}
		if t > total {
			total, worn = t, n
		}
	}
	return total, worn
}

func (e *evaluator) lines(ch *choice, a int32) {
	nr, na, stride := len(ratios), len(e.l.accAt), len(ch.members)
	r, m := len(ch.req), len(ch.opt)
	present := e.sd.pickItem[int(a)*nr] >= 0
	most := m
	if !present {
		most++
	}
	from, n, last := int(a)*nr, 0, -1
	for worn := r + 1; worn <= min(r+most, limit); worn++ {
		j, k := worn-r, ratioOf[worn]
		pre := ch.prefix[k*(stride+1):]
		var c float64
		switch {
		case present && int(ch.rank[k*na+int(a)]) < j:
			c = pre[j] - e.sd.pick[int(a)*nr+k]
		case j <= m:
			c = pre[j-1]
		default:
			c = pre[m]
		}
		switch {
		case k != last:
			ch.icept[from+n], ch.slope[from+n], ch.lineW[from+n] = c, ratios[k], int32(worn)
			n, last = n+1, k
		case c > ch.icept[from+n-1]:
			ch.icept[from+n-1], ch.lineW[from+n-1] = c, int32(worn)
		}
	}
	ch.lineN[a], ch.lineGen[a] = int32(n), ch.gen
}

func (e *evaluator) outfit(buf []scoring.Item, c *change) []scoring.Item {
	l, sd := e.l, e.sd
	out := buf[:0]
	var accAt int32 = -1
	var accItem int32
	var accS, accF float64
	nonAcc := e.nonAcc
	for _, at := range l.plainAt {
		i := sd.item[at]
		for j := range c.n {
			if v := c.s[j] + c.f[j]; c.at[j] == at && !e.br.blocked[at] && first(v, c.item[j], sd.best[at], sd.item[at]) {
				i = c.item[j]
				nonAcc = e.plainWith(at, v)
			}
		}
		if i >= 0 {
			out = append(out, l.items[i])
		}
	}
	dressItem, topItem, bottomItem := e.dressItem, e.topItem, e.bottomItem
	d, has, top, bottom := e.dress, e.hasDress, e.top, e.bottom
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
		case accessoryKind:
			if e.displaces(at, c.item[j], c.s[j], c.f[j]) {
				accAt, accItem, accS, accF = at, c.item[j], c.s[j], c.f[j]
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

	var a int32 = -1
	if accAt >= 0 {
		a = l.accOf[accAt]
		e.offer(accS, accF)
	}
	_, chosen, worn := e.settle(nonAcc+torsoOf(d, has, top, bottom), a)
	ch := &e.choices[chosen]
	nr := len(ratios)
	k := ratioOf[worn]
	for _, r := range ch.req {
		out = append(out, l.items[sd.pickItem[int(r)*nr+k]])
	}
	n := worn - len(ch.req)
	stride := len(ch.members)
	insert := false
	ov := e.offered(k)
	if a >= 0 && ch.slot[a] >= 0 {
		insert = sd.pickItem[int(a)*nr] < 0 || first(ov, accItem, sd.pick[int(a)*nr+k], sd.pickItem[int(a)*nr+k])
	}
	replaced := insert
	for _, o := range ch.order[k*stride : k*stride+len(ch.opt)] {
		if n == 0 {
			break
		}
		if replaced && o == a {
			continue
		}
		v := sd.pick[int(o)*nr+k]
		if insert && (ov > v || ov == v && ch.slot[a] < ch.slot[o]) {
			out = append(out, l.items[accItem])
			insert = false
			if n--; n == 0 {
				break
			}
		}
		out = append(out, l.items[sd.pickItem[int(o)*nr+k]])
		n--
	}
	if insert && n > 0 {
		out = append(out, l.items[accItem])
	}
	return out
}
