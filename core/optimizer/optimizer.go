package optimizer

import (
	"cmp"
	"math"
	"slices"

	"github.com/danielradosa/nikkibase/core/scoring"
)

type Position struct {
	Items     []scoring.Item
	Required  bool
	Group     uint8
	Exclusive bool
}

type Result struct {
	Items     []scoring.Item
	Score     int
	Dress     float64
	Separates float64
}

func Best(positions []Position, st scoring.Stage, sk scoring.Skills) Result {
	var outfit []scoring.Item
	var dress, top, bottom *pick
	var accessories []accessory
	grouped := map[uint8][]accessory{}

	for _, p := range positions {
		if len(p.Items) == 0 {
			continue
		}
		if p.Group != 0 && p.Items[0].Slot == scoring.Accessory {
			grouped[p.Group] = append(grouped[p.Group], weighed(p, st, sk))
			continue
		}
		switch p.Items[0].Slot {
		case scoring.Accessory:
			accessories = append(accessories, weighed(p, st, sk))
		case scoring.Dress:
			dress = better(dress, bestIn(p.Items, st, sk, 1))
		case scoring.Top:
			top = better(top, bestIn(p.Items, st, sk, 1))
		case scoring.Bottom:
			bottom = better(bottom, bestIn(p.Items, st, sk, 1))
		default:
			if b := bestIn(p.Items, st, sk, 1); b != nil {
				outfit = append(outfit, b.item)
			}
		}
	}
	if dress != nil && value(dress) >= value(top)+value(bottom) {
		outfit = appendPick(outfit, dress)
	} else {
		outfit = appendPick(outfit, top)
		outfit = appendPick(outfit, bottom)
	}

	var best []scoring.Item
	bestScore := -1
	for _, filled := range groupChoices(grouped) {
		pool := append(slices.Clone(accessories), filled...)
		items := append(slices.Clone(outfit), bestAccessories(pool)...)
		items = enforceLimits(dedupe(items), st, sk)
		if score := scoring.Score(items, st, sk); score > bestScore {
			best, bestScore = items, score
		}
	}

	return Result{Items: best, Score: bestScore,
		Dress: value(dress), Separates: value(top) + value(bottom)}
}

func BestPlaced(positions []Position, st scoring.Stage) (Result, scoring.Placement) {
	return BestPlacedFrom(positions, st, Best(positions, st, nil))
}

func BestPlacedAt(positions []Position, st scoring.Stage, l scoring.Levels) (Result, scoring.Placement) {
	return Space{branches: [][]Position{positions}}.BestPlacedAt(st, l)
}

func BestPlacedFrom(positions []Position, st scoring.Stage, unskilled Result) (Result, scoring.Placement) {
	return Space{branches: [][]Position{positions}}.BestPlacedFrom(st, unskilled)
}

func groupChoices(grouped map[uint8][]accessory) [][]accessory {
	keys := make([]uint8, 0, len(grouped))
	for g := range grouped {
		keys = append(keys, g)
	}
	slices.Sort(keys)

	choices := [][]accessory{nil}
	for _, g := range keys {
		var shared, whole []accessory
		for _, p := range grouped[g] {
			if p.Exclusive {
				whole = append(whole, p)
			} else {
				shared = append(shared, p)
			}
		}
		options := [][]accessory{shared}
		for _, w := range whole {
			options = append(options, []accessory{w})
		}
		options = keepRequired(options, grouped[g])
		var next [][]accessory
		for _, prefix := range choices {
			for _, option := range options {
				next = append(next, append(slices.Clone(prefix), option...))
			}
		}
		choices = next
	}
	return choices
}

func keepRequired(options [][]accessory, members []accessory) [][]accessory {
	required := 0
	for _, p := range members {
		if p.Required {
			required++
		}
	}
	if required == 0 {
		return options
	}
	kept := func(option []accessory) int {
		n := 0
		for _, p := range option {
			if p.Required {
				n++
			}
		}
		return n
	}
	most := 0
	for _, o := range options {
		most = max(most, kept(o))
	}
	var out [][]accessory
	for _, o := range options {
		if kept(o) == most {
			out = append(out, o)
		}
	}
	return out
}

func dedupe(outfit []scoring.Item) []scoring.Item {
	seen := make(map[int]bool, len(outfit))
	out := outfit[:0:0]
	for _, it := range outfit {
		if seen[it.ID] {
			continue
		}
		seen[it.ID] = true
		out = append(out, it)
	}
	return out
}

func enforceLimits(outfit []scoring.Item, st scoring.Stage, sk scoring.Skills) []scoring.Item {
	bySlot := map[scoring.Slot][]scoring.Item{}
	for _, it := range outfit {
		bySlot[it.Slot] = append(bySlot[it.Slot], it)
	}
	over := false
	for slot, items := range bySlot {
		if len(items) > scoring.SlotLimit(slot) {
			over = true
			break
		}
	}
	if !over {
		return outfit
	}
	kept := make(map[int]bool, len(outfit))
	for slot, items := range bySlot {
		limit := scoring.SlotLimit(slot)
		if len(items) > limit {
			slices.SortStableFunc(items, func(a, b scoring.Item) int {
				return cmpDesc(worth(a, st, sk, 1), worth(b, st, sk, 1))
			})
			items = items[:limit]
		}
		for _, it := range items {
			kept[it.ID] = true
		}
	}
	out := make([]scoring.Item, 0, len(outfit))
	for _, it := range outfit {
		if kept[it.ID] {
			kept[it.ID] = false
			out = append(out, it)
		}
	}
	return out
}

func bestAccessories(positions []accessory) []scoring.Item {
	var chosen []scoring.Item
	best := math.Inf(-1)

	picks := make([]pick, len(positions))
	ratio := 0.0
	limit := scoring.SlotLimit(scoring.Accessory)
	for worn := range min(len(positions), limit) + 1 {
		if r := scoring.AccessoryPenalty(worn); worn == 0 || r != ratio {
			ratio = r
			for i, p := range positions {
				picks[i] = p.best(ratio)
			}
		}
		var required, optional []pick
		for i, p := range positions {
			if p.Required {
				required = append(required, picks[i])
			} else {
				optional = append(optional, picks[i])
			}
		}
		if len(required) > limit {
			slices.SortStableFunc(required, func(a, b pick) int { return cmpDesc(a.value, b.value) })
			required = required[:limit]
		}
		if worn < len(required) {
			continue
		}
		slices.SortStableFunc(optional, func(a, b pick) int {
			return cmpDesc(a.value, b.value)
		})

		total := 0.0
		items := make([]scoring.Item, 0, worn)
		for _, p := range required {
			total += p.value
			items = append(items, p.item)
		}
		for _, p := range optional[:worn-len(required)] {
			total += p.value
			items = append(items, p.item)
		}
		if total > best {
			best, chosen = total, items
		}
	}
	return chosen
}

func Ranked(items []scoring.Item, st scoring.Stage, sk scoring.Skills) []scoring.Item {
	type rank struct {
		worth float64
		at    int
	}
	order := make([]rank, len(items))
	for i, it := range items {
		order[i] = rank{worth(it, st, sk, 1), i}
	}
	slices.SortFunc(order, func(a, b rank) int {
		if c := cmpDesc(a.worth, b.worth); c != 0 {
			return c
		}
		return cmp.Compare(a.at, b.at)
	})
	ranked := make([]scoring.Item, len(items))
	for i, r := range order {
		ranked[i] = items[r.at]
	}
	return ranked
}

func Exclude(positions []Position, ids ...int) []Position {
	out := make([]Position, 0, len(positions))
	for _, p := range positions {
		kept := make([]scoring.Item, 0, len(p.Items))
		for _, it := range p.Items {
			if !slices.Contains(ids, it.ID) {
				kept = append(kept, it)
			}
		}
		q := p
		q.Items = kept
		out = append(out, q)
	}
	return out
}

type pick struct {
	item  scoring.Item
	value float64
}

type accessory struct {
	Position
	worths []contribution
}

type contribution struct{ scaled, fixed float64 }

func weighed(p Position, st scoring.Stage, sk scoring.Skills) accessory {
	worths := make([]contribution, len(p.Items))
	for i, it := range p.Items {
		worths[i].scaled, worths[i].fixed = scoring.Contribution(it, st, sk)
	}
	return accessory{p, worths}
}

func (a accessory) best(ratio float64) pick {
	at, top := 0, 0.0
	for i, c := range a.worths {
		if v := ratio*c.scaled + c.fixed; i == 0 || v > top {
			at, top = i, v
		}
	}
	return pick{a.Items[at], top}
}

func worth(it scoring.Item, st scoring.Stage, sk scoring.Skills, ratio float64) float64 {
	scaled, fixed := scoring.Contribution(it, st, sk)
	return ratio*scaled + fixed
}

func bestIn(items []scoring.Item, st scoring.Stage, sk scoring.Skills, ratio float64) *pick {
	at, top := -1, 0.0
	for i, it := range items {
		if v := worth(it, st, sk, ratio); at < 0 || v > top {
			at, top = i, v
		}
	}
	if at < 0 {
		return nil
	}
	return &pick{items[at], top}
}

func better(a, b *pick) *pick {
	if a == nil || (b != nil && b.value > a.value) {
		return b
	}
	return a
}

func value(p *pick) float64 {
	if p == nil {
		return 0
	}
	return p.value
}

func appendPick(items []scoring.Item, p *pick) []scoring.Item {
	if p == nil {
		return items
	}
	return append(items, p.item)
}

func cmpDesc(a, b float64) int {
	switch {
	case a > b:
		return -1
	case a < b:
		return 1
	default:
		return 0
	}
}
