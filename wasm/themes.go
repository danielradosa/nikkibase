//go:build js && wasm

package main

import (
	"slices"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
	"github.com/danielradosa/nikkibase/core/themes"
)

func (e *engine) split(_ js.Value, args []js.Value) any {
	if e.catalogue == nil {
		return fail("catalogue not loaded")
	}
	list := arg(args, 0)
	if !isArray(list) || list.Length() < 2 {
		return fail("the themes must be a list of two or more")
	}
	v := e.view(!(len(args) > 1 && args[1].String() == "all"))
	n := list.Length()
	stages := make([]scoring.Stage, n)
	required := make([][][]int, n)
	skills := make([]skillRequest, n)
	for k := range n {
		t := list.Index(k)
		if !isObject(t) || !isArray(t.Get("weights")) || !isArray(t.Get("attrs")) {
			return fail("each theme needs weights and attrs")
		}
		stages[k] = stageArg(t.Get("weights"), t.Get("attrs"), t.Get("tags"))
		required[k] = requireArg(t.Get("require"))
		var ok bool
		if skills[k], ok = skillsArg(t.Get("skills")); !ok {
			return fail("skill levels run from 0 to 9")
		}
	}
	solve := func(k int, exclude map[int]bool) optimizer.Result {
		r, _, _ := skills[k].solve(optimizer.Require(themes.Without(v.positions, exclude), v.posOf, required[k]), stages[k])
		return r
	}
	refs := make([]int, n)
	alone := make([]optimizer.Result, n)
	for k := range n {
		alone[k] = solve(k, nil)
		refs[k] = alone[k].Score
	}
	others := themes.Others(themes.Settle(themes.Split(n, refs, solve), solve))
	var b strings.Builder
	b.WriteString(`{"exclude":[`)
	for k, ids := range others {
		if k > 0 {
			b.WriteByte(',')
		}
		writeInts(&b, ids)
	}
	b.WriteString(`],"taken":[`)
	for k, ids := range others {
		if k > 0 {
			b.WriteByte(',')
		}
		taken := 0
		for _, it := range alone[k].Items {
			if slices.Contains(ids, it.ID) {
				taken++
			}
		}
		b.WriteString(strconv.Itoa(taken))
	}
	b.WriteString(`]}`)
	return result(b.String())
}
