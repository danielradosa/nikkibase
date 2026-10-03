package themes

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"

	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
)

var themeKey = regexp.MustCompile(`^(Story/(?:II-|III-)?\d+-\d+)-(\d+)(#.+)?$`)

func Groups(keys []string) [][]int {
	type member struct{ at, theme int }
	members := map[string][]member{}
	var order []string
	for at, key := range keys {
		m := themeKey.FindStringSubmatch(key)
		if m == nil {
			continue
		}
		id := m[1] + m[3]
		if _, ok := members[id]; !ok {
			order = append(order, id)
		}
		theme, _ := strconv.Atoi(m[2])
		members[id] = append(members[id], member{at, theme})
	}
	var out [][]int
	for _, id := range order {
		ms := members[id]
		if len(ms) < 2 {
			continue
		}
		slices.SortStableFunc(ms, func(a, b member) int { return a.theme - b.theme })
		group := make([]int, len(ms))
		for i, m := range ms {
			group[i] = m.at
		}
		out = append(out, group)
	}
	return out
}

func Without(positions []optimizer.Position, exclude map[int]bool) []optimizer.Position {
	if len(exclude) == 0 {
		return positions
	}
	out := make([]optimizer.Position, len(positions))
	for at, p := range positions {
		q := p
		q.Items = nil
		for _, it := range p.Items {
			if !exclude[it.ID] {
				q.Items = append(q.Items, it)
			}
		}
		out[at] = q
	}
	return out
}

func Split(n int, refs []int, solve func(theme int, exclude map[int]bool) optimizer.Result) []optimizer.Result {
	seen := map[string]optimizer.Result{}
	inner := solve
	solve = func(theme int, exclude map[int]bool) optimizer.Result {
		ids := make([]int, 0, len(exclude))
		for id, on := range exclude {
			if on {
				ids = append(ids, id)
			}
		}
		slices.Sort(ids)
		key := fmt.Sprint(theme, ids)
		if r, ok := seen[key]; ok {
			return r
		}
		r := inner(theme, exclude)
		seen[key] = r
		return r
	}
	var best []optimizer.Result
	bestMin, bestSum := -1.0, -1.0
	for _, order := range orders(n) {
		results := make([]optimizer.Result, n)
		used := map[int]bool{}
		for _, t := range order {
			results[t] = solve(t, used)
			for _, it := range results[t].Items {
				used[it.ID] = true
			}
		}
		for range 50 {
			next, ok := improve(results, refs, solve)
			if !ok {
				break
			}
			results = next
		}
		weakest, sum := shares(results, refs)
		if better(weakest, sum, bestMin, bestSum) {
			best, bestMin, bestSum = results, weakest, sum
		}
	}
	return best
}

func better(weakest, sum, thanWeakest, thanSum float64) bool {
	return weakest > thanWeakest || weakest == thanWeakest && sum > thanSum
}

func improve(cur []optimizer.Result, refs []int, solve func(int, map[int]bool) optimizer.Result) ([]optimizer.Result, bool) {
	curMin, curSum := shares(cur, refs)
	w := weakestTheme(cur, refs)
	var best []optimizer.Result
	bestMin, bestSum := curMin, curSum
	for o := range cur {
		if o == w {
			continue
		}
		rest := usedBy(cur, w, o)
		wants := solve(w, rest)
		var contested []int
		for _, it := range wants.Items {
			if slices.ContainsFunc(cur[o].Items, func(x scoring.Item) bool { return x.ID == it.ID }) {
				contested = append(contested, it.ID)
			}
		}
		if len(contested) == 0 {
			continue
		}
		moves := [][]int{contested}
		if len(contested) > 1 {
			for _, id := range contested {
				moves = append(moves, []int{id})
			}
		}
		for _, give := range moves {
			for _, keepOut := range []bool{false, true} {
				lost := maps.Clone(rest)
				for _, id := range give {
					lost[id] = true
				}
				if keepOut {
					for _, it := range cur[w].Items {
						lost[it.ID] = true
					}
				}
				next := slices.Clone(cur)
				next[w] = solve(w, plus(rest, solve(o, lost)))
				next[o] = solve(o, plus(rest, next[w]))
				weakest, sum := shares(next, refs)
				if better(weakest, sum, bestMin, bestSum) {
					best, bestMin, bestSum = next, weakest, sum
				}
			}
		}
	}
	return best, best != nil
}

func plus(used map[int]bool, r optimizer.Result) map[int]bool {
	out := maps.Clone(used)
	for _, it := range r.Items {
		out[it.ID] = true
	}
	return out
}

func weakestTheme(results []optimizer.Result, refs []int) int {
	w, wShare := 0, 2.0
	for t, r := range results {
		s := 1.0
		if refs[t] > 0 {
			s = float64(r.Score) / float64(refs[t])
		}
		if s < wShare {
			w, wShare = t, s
		}
	}
	return w
}

func usedBy(results []optimizer.Result, skip ...int) map[int]bool {
	used := map[int]bool{}
	for t, r := range results {
		if slices.Contains(skip, t) {
			continue
		}
		for _, it := range r.Items {
			used[it.ID] = true
		}
	}
	return used
}

func shares(results []optimizer.Result, refs []int) (weakest, sum float64) {
	weakest = 2
	for t, r := range results {
		s := 1.0
		if refs[t] > 0 {
			s = float64(r.Score) / float64(refs[t])
		}
		weakest = min(weakest, s)
		sum += s
	}
	return weakest, sum
}

func Others(results []optimizer.Result) [][]int {
	out := make([][]int, len(results))
	for t := range results {
		ids := []int{}
		for o, r := range results {
			if o == t {
				continue
			}
			for _, it := range r.Items {
				ids = append(ids, it.ID)
			}
		}
		slices.Sort(ids)
		out[t] = ids
	}
	return out
}

func orders(n int) [][]int {
	if n <= 1 {
		return [][]int{make([]int, n)}
	}
	var out [][]int
	for _, rest := range orders(n - 1) {
		for i := n - 1; i >= 0; i-- {
			out = append(out, slices.Insert(slices.Clone(rest), i, n-1))
		}
	}
	slices.SortFunc(out, slices.Compare)
	return out
}
