package optimizer

import (
	"cmp"
	"maps"
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
	for _, filled := range GroupChoices(nil, grouped, func(a accessory) bool { return a.Exclusive }, func(a accessory) bool { return a.Required }) {
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

func GroupChoices[T any](seed []T, grouped map[uint8][]T, exclusive, required func(T) bool) [][]T {
	choices := [][]T{seed}
	for _, g := range slices.Sorted(maps.Keys(grouped)) {
		var shared []T
		var whole [][]T
		for _, x := range grouped[g] {
			if exclusive(x) {
				whole = append(whole, []T{x})
			} else {
				shared = append(shared, x)
			}
		}
		options := keepRequired(append([][]T{shared}, whole...), required)
		var next [][]T
		for _, prefix := range choices {
			for _, o := range options {
				next = append(next, append(slices.Clone(prefix), o...))
			}
		}
		choices = next
	}
	return choices
}

func keepRequired[T any](options [][]T, required func(T) bool) [][]T {
	count := func(o []T) int {
		n := 0
		for _, x := range o {
			if required(x) {
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
	var kept [][]T
	for _, o := range options {
		if count(o) == most {
			kept = append(kept, o)
		}
	}
	return kept
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
				return cmp.Compare(worth(b, st, sk, 1), worth(a, st, sk, 1))
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
			slices.SortStableFunc(required, func(a, b pick) int { return cmp.Compare(b.value, a.value) })
			required = required[:limit]
		}
		if worn < len(required) {
			continue
		}
		slices.SortStableFunc(optional, func(a, b pick) int { return cmp.Compare(b.value, a.value) })

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
		if c := cmp.Compare(b.worth, a.worth); c != 0 {
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
