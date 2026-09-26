package optimizer

import (
	"slices"

	"github.com/danielradosa/nikkibase/core/scoring"
)

type Space struct {
	branches [][]Position
	sets     [][]int
}

type option struct {
	at  int
	ids []int
}

func Require(positions []Position, posOf map[int]int, sets [][]int) Space {
	var kept [][]option
	for _, set := range sets {
		options := optionsFor(positions, posOf, set)
		if len(options) == 0 {
			continue
		}
		if len(branches(positions, append(slices.Clone(kept), options))) > 0 {
			kept = append(kept, options)
		}
	}
	return Space{branches: branches(positions, kept), sets: sets}
}

func (s Space) Best(st scoring.Stage, sk scoring.Skills) Result {
	var best Result
	fewest := -1
	for _, positions := range s.branches {
		r := Best(positions, st, sk)
		unmet := len(Unmet(r.Items, s.sets))
		if fewest < 0 || unmet < fewest || (unmet == fewest && r.Score > best.Score) {
			best, fewest = r, unmet
		}
	}
	return best
}

func (s Space) Branches() [][]Position {
	return s.branches
}

func (s Space) BestPlaced(st scoring.Stage) (Result, scoring.Placement) {
	return s.BestPlacedAt(st, scoring.MaxLevels)
}

func (s Space) BestPlacedAt(st scoring.Stage, l scoring.Levels) (Result, scoring.Placement) {
	return s.BestPlacedFromAt(st, s.Best(st, nil), l)
}

func (s Space) BestPlacedFrom(st scoring.Stage, unskilled Result) (Result, scoring.Placement) {
	return s.BestPlacedFromAt(st, unskilled, scoring.MaxLevels)
}

func (s Space) BestPlacedFromAt(st scoring.Stage, unskilled Result, l scoring.Levels) (Result, scoring.Placement) {
	placement := scoring.Place(unskilled.Items, st)
	sk := placement.SkillsAt(l)
	placed := s.Best(st, sk)
	if rescored := scoring.Score(unskilled.Items, st, sk); rescored > placed.Score {
		unskilled.Score = rescored
		return unskilled, placement
	}
	return placed, placement
}

func Unmet(outfit []scoring.Item, sets [][]int) [][]int {
	var unmet [][]int
	for _, set := range sets {
		if !slices.ContainsFunc(outfit, func(it scoring.Item) bool { return slices.Contains(set, it.ID) }) {
			unmet = append(unmet, set)
		}
	}
	return unmet
}

func optionsFor(positions []Position, posOf map[int]int, set []int) []option {
	var options []option
	for _, id := range set {
		at, ok := posOf[id]
		if !ok || at < 0 || at >= len(positions) ||
			!slices.ContainsFunc(positions[at].Items, func(it scoring.Item) bool { return it.ID == id }) {
			continue
		}
		if i := slices.IndexFunc(options, func(o option) bool { return o.at == at }); i >= 0 {
			options[i].ids = append(options[i].ids, id)
		} else {
			options = append(options, option{at, []int{id}})
		}
	}
	return options
}

func branches(positions []Position, choices [][]option) [][]Position {
	var out [][]Position
	picks := make([]option, len(choices))
	var walk func(i int)
	walk = func(i int) {
		if i == len(choices) {
			if narrowed, ok := narrow(positions, picks); ok {
				out = append(out, narrowed)
			}
			return
		}
		for _, o := range choices[i] {
			picks[i] = o
			walk(i + 1)
		}
	}
	walk(0)
	return out
}

func narrow(positions []Position, picks []option) ([]Position, bool) {
	out := slices.Clone(positions)
	var picked []int
	for _, o := range picks {
		p := out[o.at]
		kept := make([]scoring.Item, 0, len(o.ids))
		for _, it := range p.Items {
			if slices.Contains(o.ids, it.ID) {
				kept = append(kept, it)
			}
		}
		if len(kept) == 0 {
			return nil, false
		}
		p.Items, p.Required = kept, true
		out[o.at] = p
		if !slices.Contains(picked, o.at) {
			picked = append(picked, o.at)
		}
	}

	accessories := 0
	for k, i := range picked {
		if out[i].Items[0].Slot == scoring.Accessory {
			accessories++
		}
		for _, j := range picked[k+1:] {
			if clash(out[i], out[j]) {
				return nil, false
			}
		}
	}
	if accessories > scoring.SlotLimit(scoring.Accessory) {
		return nil, false
	}

	for _, i := range picked {
		for j := range out {
			if len(out[j].Items) > 0 && torsoClash(out[i].Items[0].Slot, out[j].Items[0].Slot) {
				out[j].Items = nil
			}
		}
	}
	return out, true
}

func clash(a, b Position) bool {
	if a.Group != 0 && a.Group == b.Group && (a.Exclusive || b.Exclusive) {
		return true
	}
	return torsoClash(a.Items[0].Slot, b.Items[0].Slot)
}

func torsoClash(a, b scoring.Slot) bool {
	separates := func(s scoring.Slot) bool { return s == scoring.Top || s == scoring.Bottom }
	return a == scoring.Dress && separates(b) || b == scoring.Dress && separates(a)
}
