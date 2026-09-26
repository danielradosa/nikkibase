package worth

import (
	"slices"

	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
)

type kind uint8

const (
	plainKind kind = iota
	dressKind
	topKind
	bottomKind
	accessoryKind
)

const otherCode = 10

var limit = scoring.SlotLimit(scoring.Accessory)

var ratios, ratioOf = penalties()

func penalties() ([]float64, []int) {
	var rs []float64
	of := make([]int, limit+1)
	for worn := range limit + 1 {
		if r := scoring.AccessoryPenalty(worn); worn == 0 || r != rs[len(rs)-1] {
			rs = append(rs, r)
		}
		of[worn] = len(rs) - 1
	}
	return rs, of
}

type layout struct {
	positions []optimizer.Position
	posOf     map[int]int
	items     []scoring.Item
	at        []int32
	owned     []bool
	index     map[int]int32
	kinds     []kind
	plainAt   []int32
	accOf     []int32
	accAt     []int32
	ownedIn   [][]int32
	fillIn    [][]int32
	candIn    [][]int32
	code      [][5]uint8
	stats     [][5]float64
	tagFrom   []int32
	tags      []int32
	size      []float64
	flat      []float64
	awards    int
	inexact   bool
	plain     *branch
	ownedPos  []optimizer.Position
}

func newLayout(positions []optimizer.Position, posOf map[int]int, owned func(int32) bool) *layout {
	n := 0
	for _, p := range positions {
		n += len(p.Items)
	}
	l := &layout{
		positions: positions, posOf: posOf, index: make(map[int]int32, n),
		items: make([]scoring.Item, 0, n), at: make([]int32, 0, n), owned: make([]bool, 0, n),
		code: make([][5]uint8, 0, n), stats: make([][5]float64, 0, n),
		tagFrom: make([]int32, 1, n+1), size: make([]float64, 0, n), flat: make([]float64, 0, n),
	}
	for at, p := range positions {
		k := kindOf(p)
		l.kinds = append(l.kinds, k)
		switch k {
		case accessoryKind:
			l.accOf = append(l.accOf, int32(len(l.accAt)))
			l.accAt = append(l.accAt, int32(at))
		case plainKind:
			l.accOf = append(l.accOf, -1)
			l.plainAt = append(l.plainAt, int32(at))
		default:
			l.accOf = append(l.accOf, -1)
		}
		var mine, theirs []int32
		for _, it := range p.Items {
			i := int32(len(l.items))
			have := owned == nil || owned(int32(it.ID))
			if have {
				mine = append(mine, i)
			} else {
				theirs = append(theirs, i)
			}
			if _, seen := l.index[it.ID]; !seen {
				l.index[it.ID] = i
			}
			l.items = append(l.items, it)
			l.at = append(l.at, int32(at))
			l.owned = append(l.owned, have)
			l.pack(it)
		}
		l.ownedIn = append(l.ownedIn, mine)
		l.candIn = append(l.candIn, theirs)
	}
	l.fillIn = l.ownedIn
	l.plain = l.newBranch(l.fillIn, nil)
	return l
}

func (l *layout) prune(protected func(int32) bool) {
	var buckets [32][]int32
	fill := make([][]int32, len(l.positions))
	for at := range l.positions {
		for k := range buckets {
			buckets[k] = buckets[k][:0]
		}
		owned := l.ownedIn[at]
		for _, i := range owned {
			buckets[l.pattern(i)] = append(buckets[l.pattern(i)], i)
		}
		covered := func(i int32, owner bool) bool {
			var free uint8
			for p := range 5 {
				if l.stats[i][p] == 0 {
					free |= 1 << p
				}
			}
			base := l.pattern(i) &^ free
			for sub := free; ; sub = (sub - 1) & free {
				for _, j := range buckets[base|sub] {
					if j != i && l.covers(j, i) && (!owner || j < i) {
						return true
					}
				}
				if sub == 0 {
					return false
				}
			}
		}
		kept := make([]int32, 0, len(owned))
		for _, i := range owned {
			if !covered(i, true) {
				kept = append(kept, i)
			}
		}
		cands := make([]int32, 0, len(l.candIn[at]))
		for _, i := range l.candIn[at] {
			if protected(i) || !covered(i, false) {
				cands = append(cands, i)
			}
		}
		fill[at], l.candIn[at] = kept, cands
	}
	l.fillIn = fill
	l.plain = l.newBranch(l.fillIn, nil)
}

func (l *layout) pattern(i int32) uint8 {
	var bits uint8
	for p, c := range l.code[i] {
		bits |= (c & 1) << p
	}
	return bits
}

func (l *layout) covers(j, i int32) bool {
	if l.size[i] != l.size[j] || l.flat[i] > l.flat[j] {
		return false
	}
	for p := range 5 {
		si, sj := l.stats[i][p], l.stats[j][p]
		if si < 0 || sj < 0 {
			return false
		}
		if si > 0 && (l.code[i][p] != l.code[j][p] || l.code[i][p] == otherCode || sj < si) {
			return false
		}
	}
	mine, theirs := l.tags[l.tagFrom[i]:l.tagFrom[i+1]], l.tags[l.tagFrom[j]:l.tagFrom[j+1]]
	for _, t := range mine {
		if count(mine, t) > count(theirs, t) {
			return false
		}
	}
	return true
}

func count(tags []int32, t int32) int {
	n := 0
	for _, x := range tags {
		if x == t {
			n++
		}
	}
	return n
}

func kindOf(p optimizer.Position) kind {
	if len(p.Items) == 0 {
		return plainKind
	}
	switch p.Items[0].Slot {
	case scoring.Dress:
		return dressKind
	case scoring.Top:
		return topKind
	case scoring.Bottom:
		return bottomKind
	case scoring.Accessory:
		return accessoryKind
	}
	return plainKind
}

func (l *layout) pack(it scoring.Item) {
	var code [5]uint8
	var stats [5]float64
	for p := range 5 {
		c := it.Attrs[p]
		if c < 0 || c >= otherCode {
			c = otherCode
		}
		code[p] = uint8(c)
		stats[p] = float64(it.Stats[p])
	}
	l.code = append(l.code, code)
	l.stats = append(l.stats, stats)
	for _, t := range it.Tags {
		if t < 0 {
			l.inexact = true
			continue
		}
		l.tags = append(l.tags, int32(t))
		l.awards = max(l.awards, t+1)
	}
	l.tagFrom = append(l.tagFrom, int32(len(l.tags)))
	size, flat := 0.0, 0.0
	if it.Slot <= scoring.Spirit {
		size = scoring.SlotSize(it.Slot)
	} else {
		l.inexact = true
	}
	if it.Slot == scoring.Spirit {
		flat = float64(it.FlatBonus)
	}
	l.size = append(l.size, size)
	l.flat = append(l.flat, flat)
}

type coef struct {
	w     [5][otherCode + 1]float64
	award []float64
	exact bool
	lift  float64
	plus  bool
	st    scoring.Stage
	sk    scoring.Skills
}

func (l *layout) coefOf(st scoring.Stage, sk scoring.Skills) *coef {
	c := &coef{award: make([]float64, l.awards), exact: !l.inexact, lift: 1, plus: true, st: st, sk: sk}
	for _, m := range sk {
		c.lift = max(c.lift, m)
		c.plus = c.plus && m >= 0
	}
	for p := range 5 {
		c.plus = c.plus && st.Weights[p] >= 0
		code := st.Attrs[p]
		if code < 0 || code >= otherCode {
			c.exact = false
			continue
		}
		m := 1.0
		if v, ok := sk[int(code)]; ok {
			m = v
		}
		c.w[p][code] = st.Weights[p] * m
	}
	for t, a := range st.Tags {
		if t >= 0 && t < len(c.award) {
			c.award[t] = float64(a)
		}
	}
	return c
}

func (l *layout) value(c *coef, i int32) (float64, float64) {
	if !c.exact {
		return scoring.Contribution(l.items[i], c.st, c.sk)
	}
	code, stats := &l.code[i], &l.stats[i]
	scaled := 0.0
	for p := range 5 {
		scaled += c.w[p][code[p]] * stats[p]
	}
	fixed := 0.0
	size := l.size[i]
	for _, t := range l.tags[l.tagFrom[i]:l.tagFrom[i+1]] {
		fixed += c.award[t] * size
	}
	return scaled, fixed + l.flat[i]
}

type branch struct {
	pool     [][]int32
	blocked  []bool
	required []bool
	choices  [][]int32
}

func (l *layout) newBranch(pool [][]int32, narrowed []bool) *branch {
	b := &branch{pool: pool, blocked: make([]bool, len(l.kinds)), required: make([]bool, len(l.accAt))}
	var dress, separates bool
	for at, n := range narrowed {
		if !n {
			continue
		}
		b.blocked[at] = true
		switch l.kinds[at] {
		case dressKind:
			dress = true
		case topKind, bottomKind:
			separates = true
		case accessoryKind:
			b.required[l.accOf[at]] = true
		}
	}
	for at, k := range l.kinds {
		if dress && (k == topKind || k == bottomKind) || separates && k == dressKind {
			b.blocked[at] = true
		}
	}
	b.choices = l.choicesFor(b.required)
	return b
}

func (l *layout) choicesFor(required []bool) [][]int32 {
	var ungrouped []int32
	grouped := map[uint8][]int32{}
	var keys []uint8
	for a, at := range l.accAt {
		g := l.positions[at].Group
		if g == 0 {
			ungrouped = append(ungrouped, int32(a))
			continue
		}
		if _, seen := grouped[g]; !seen {
			keys = append(keys, g)
		}
		grouped[g] = append(grouped[g], int32(a))
	}
	slices.Sort(keys)
	choices := [][]int32{ungrouped}
	for _, g := range keys {
		var shared []int32
		var whole [][]int32
		for _, a := range grouped[g] {
			if l.positions[l.accAt[a]].Exclusive {
				whole = append(whole, []int32{a})
			} else {
				shared = append(shared, a)
			}
		}
		options := keepRequired(append([][]int32{shared}, whole...), required)
		var next [][]int32
		for _, prefix := range choices {
			for _, o := range options {
				next = append(next, append(slices.Clone(prefix), o...))
			}
		}
		choices = next
	}
	return choices
}

func keepRequired(options [][]int32, required []bool) [][]int32 {
	count := func(o []int32) int {
		n := 0
		for _, a := range o {
			if required[a] {
				n++
			}
		}
		return n
	}
	most := 0
	for _, o := range options {
		most = max(most, count(o))
	}
	if most == 0 {
		return options
	}
	var kept [][]int32
	for _, o := range options {
		if count(o) == most {
			kept = append(kept, o)
		}
	}
	return kept
}

func (l *layout) branchOf(positions []optimizer.Position, base [][]int32) *branch {
	pool := make([][]int32, len(positions))
	narrowed := make([]bool, len(positions))
	for at, p := range positions {
		narrowed[at] = p.Required
		switch {
		case len(p.Items) == 0:
		case !p.Required:
			pool[at] = base[at]
		default:
			for _, it := range p.Items {
				pool[at] = append(pool[at], l.index[it.ID])
			}
		}
	}
	return l.newBranch(pool, narrowed)
}

func (l *layout) poolWith(extras []int32) [][]int32 {
	return l.withExtras(l.fillIn, extras)
}

func (l *layout) withExtras(base [][]int32, extras []int32) [][]int32 {
	if len(extras) == 0 {
		return base
	}
	pool := slices.Clone(base)
	for _, i := range extras {
		at := l.at[i]
		k, _ := slices.BinarySearch(pool[at], i)
		pool[at] = slices.Insert(slices.Clone(pool[at]), k, i)
	}
	return pool
}

func (l *layout) positionsWith(extras []int32) []optimizer.Position {
	if l.ownedPos == nil {
		l.ownedPos = make([]optimizer.Position, len(l.positions))
		for at, p := range l.positions {
			p.Items = l.itemsOf(l.ownedIn[at])
			l.ownedPos[at] = p
		}
	}
	if len(extras) == 0 {
		return l.ownedPos
	}
	out := slices.Clone(l.ownedPos)
	pool := l.withExtras(l.ownedIn, extras)
	for _, i := range extras {
		out[l.at[i]].Items = l.itemsOf(pool[l.at[i]])
	}
	return out
}

func (l *layout) itemsOf(indices []int32) []scoring.Item {
	items := make([]scoring.Item, len(indices))
	for j, i := range indices {
		items[j] = l.items[i]
	}
	return items
}
