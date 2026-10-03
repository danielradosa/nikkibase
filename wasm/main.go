//go:build js && wasm

package main

import (
	"slices"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/danielradosa/nikkibase/core/catalogue"
	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
	"github.com/danielradosa/nikkibase/core/themes"
	"github.com/danielradosa/nikkibase/core/wardrobe"
	"github.com/danielradosa/nikkibase/core/worth"
)

type engine struct {
	catalogue *catalogue.Catalogue
	keystream *wardrobe.Keystream
	owned     []int
	views     map[bool]*view
	worth     *worth.Session
	session   int
}

type view struct {
	positions []optimizer.Position
	posOf     map[int]int
	placeOf   map[int]uint16
	places    []uint16
}

func main() {
	e := &engine{}
	js.Global().Set("nikkibase", js.ValueOf(map[string]any{
		"loadCatalogue":    js.FuncOf(e.loadCatalogue),
		"loadKeystream":    js.FuncOf(e.loadKeystream),
		"decodeWardrobe":   js.FuncOf(e.decodeWardrobe),
		"importSelections": js.FuncOf(e.importSelections),
		"setWardrobe":      js.FuncOf(e.setWardrobe),
		"best":             js.FuncOf(e.best),
		"worthStart":       js.FuncOf(e.worthStart),
		"worthRun":         js.FuncOf(e.worthRun),
		"worthRank":        js.FuncOf(e.worthRank),
		"worthBases":       js.FuncOf(e.worthBases),
		"split":            js.FuncOf(e.split),
		"places":           js.FuncOf(e.places),
	}))
	select {}
}

func (e *engine) loadCatalogue(_ js.Value, args []js.Value) any {
	c, err := catalogue.Read(bytesFrom(args[0]))
	if err != nil {
		return fail(err.Error())
	}
	e.catalogue = c
	e.views = nil
	e.worth = nil
	return result(`{"items":` + strconv.Itoa(len(c.IDs)) + `}`)
}

func (e *engine) places(_ js.Value, _ []js.Value) any {
	if e.catalogue == nil {
		return fail("catalogue not loaded")
	}
	var b strings.Builder
	b.WriteString(`{"ids":`)
	writeInts(&b, e.catalogue.IDs)
	b.WriteString(`,"places":`)
	writeInts(&b, e.catalogue.Positions)
	b.WriteByte('}')
	return result(b.String())
}

func (e *engine) loadKeystream(_ js.Value, args []js.Value) any {
	ks, err := wardrobe.ParseKeystream(bytesFrom(args[0]))
	if err != nil {
		return fail(err.Error())
	}
	e.keystream = ks
	return result(`{"length":` + strconv.Itoa(ks.Len()) + `}`)
}

func (e *engine) decodeWardrobe(_ js.Value, args []js.Value) any {
	if e.keystream == nil {
		return fail("keystream not loaded")
	}
	w, err := wardrobe.Decode([]byte(args[0].String()), e.keystream)
	if err != nil {
		return fail(err.Error())
	}
	return e.adopt(w)
}

func (e *engine) importSelections(_ js.Value, args []js.Value) any {
	w, err := wardrobe.DecodeSelections([]byte(args[0].String()))
	if err != nil {
		return fail(err.Error())
	}
	return e.adopt(w)
}

func (e *engine) adopt(w *wardrobe.Wardrobe) any {
	e.owned = w.Items
	delete(e.views, true)
	e.worth = nil
	var ids strings.Builder
	writeInts(&ids, w.Items)
	return result(`{"items":` + strconv.Itoa(len(w.Items)) +
		`,"unresolved":` + strconv.Itoa(w.Unresolved) +
		`,"known":` + strconv.Itoa(e.known()) +
		`,"ids":` + ids.String() + `}`)
}

func (e *engine) setWardrobe(_ js.Value, args []js.Value) any {
	ids := args[0]
	owned := make([]int, 0, ids.Length())
	for i := range ids.Length() {
		owned = append(owned, ids.Index(i).Int())
	}
	e.owned = owned
	delete(e.views, true)
	e.worth = nil
	return result(`{"items":` + strconv.Itoa(len(owned)) +
		`,"unresolved":0,"known":` + strconv.Itoa(e.known()) + `}`)
}

func stageArg(weights, attrs, tags js.Value) scoring.Stage {
	var st scoring.Stage
	for p := range 5 {
		st.Weights[p] = weights.Index(p).Float()
		st.Attrs[p] = int8(attrs.Index(p).Int())
	}
	if !tags.IsNull() && !tags.IsUndefined() {
		st.Tags = map[int]int{}
		entries := js.Global().Get("Object").Call("entries", tags)
		for i := range entries.Length() {
			pair := entries.Index(i)
			id, err := strconv.Atoi(pair.Index(0).String())
			if err != nil {
				continue
			}
			st.Tags[id] = pair.Index(1).Int()
		}
	}
	return st
}

func requireArg(v js.Value) [][]int {
	if v.IsNull() || v.IsUndefined() {
		return nil
	}
	var required [][]int
	for i := range v.Length() {
		set := v.Index(i)
		ids := make([]int, 0, set.Length())
		for j := range set.Length() {
			ids = append(ids, set.Index(j).Int())
		}
		required = append(required, ids)
	}
	return required
}

func idsArg(v js.Value) map[int]bool {
	if v.IsNull() || v.IsUndefined() {
		return nil
	}
	out := make(map[int]bool, v.Length())
	for i := range v.Length() {
		out[v.Index(i).Int()] = true
	}
	return out
}

type skillRequest struct {
	chosen, auto bool
	levels       scoring.Levels
	placement    scoring.Placement
}

func skillsArg(v js.Value) (skillRequest, bool) {
	req := skillRequest{levels: scoring.MaxLevels}
	if v.IsNull() || v.IsUndefined() {
		return req, true
	}
	req.chosen = true
	var ok bool
	if req.levels, ok = worthLevels(v.Get("levels")); !ok {
		return req, false
	}
	req.auto = v.Get("auto").Truthy()
	if !req.auto {
		req.placement = scoring.Placement{CharmSmile: attrCode(v.Get("charmSmile")), Smile: attrCode(v.Get("smile"))}
		if req.placement.Smile == req.placement.CharmSmile || req.levels.Smile == 0 {
			req.placement.Smile = -1
		}
	}
	return req, true
}

func (req skillRequest) solve(space optimizer.Space, st scoring.Stage) (optimizer.Result, *scoring.Placement, scoring.Skills) {
	switch {
	case !req.chosen || req.levels.None():
		return space.Best(st, scoring.Skills{}), nil, scoring.Skills{}
	case req.auto:
		outfit, p := space.BestPlacedAt(st, req.levels)
		if req.levels.Smile == 0 {
			p.Smile = -1
		}
		return outfit, &p, p.SkillsAt(req.levels)
	default:
		p := req.placement
		skills := p.SkillsAt(req.levels)
		return space.Best(st, skills), &p, skills
	}
}

func (e *engine) best(_ js.Value, args []js.Value) any {
	if e.catalogue == nil {
		return fail("catalogue not loaded")
	}
	st := stageArg(args[0], args[1], arg(args, 3))
	ownedOnly := !(len(args) > 4 && args[4].String() == "all")
	required := requireArg(arg(args, 5))
	v := e.view(ownedOnly)
	positions, posOf, placeOf := themes.Without(v.positions, idsArg(arg(args, 6))), v.posOf, v.placeOf
	space := optimizer.Require(positions, posOf, required)

	req, ok := skillsArg(arg(args, 2))
	if !ok {
		return fail("skill levels run from 0 to 9")
	}
	levels := req.levels
	outfit, placement, skills := req.solve(space, st)
	missing := optimizer.Unmet(outfit.Items, required)
	var met [][]int
	for _, set := range required {
		if len(optimizer.Unmet(outfit.Items, [][]int{set})) == 0 {
			met = append(met, set)
		}
	}

	worn := 0
	for _, it := range outfit.Items {
		if it.Slot == scoring.Accessory {
			worn++
		}
	}
	ratio := scoring.AccessoryPenalty(worn)

	var b strings.Builder
	b.WriteString(`{"score":`)
	b.WriteString(strconv.Itoa(outfit.Score))
	b.WriteString(`,"dress":`)
	b.WriteString(strconv.Itoa(int(outfit.Dress)))
	b.WriteString(`,"separates":`)
	b.WriteString(strconv.Itoa(int(outfit.Separates)))
	if placement != nil {
		b.WriteString(`,"skills":{"charmSmile":`)
		b.WriteString(strconv.Itoa(placement.CharmSmile))
		b.WriteString(`,"smile":`)
		b.WriteString(strconv.Itoa(placement.Smile))
		if levels != scoring.MaxLevels {
			b.WriteString(`,"levels":{"charming":`)
			b.WriteString(strconv.Itoa(levels.Charming))
			b.WriteString(`,"smile":`)
			b.WriteString(strconv.Itoa(levels.Smile))
			b.WriteByte('}')
		}
		b.WriteByte('}')
	}
	b.WriteString(`,"items":[`)
	for i, it := range outfit.Items {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"id":`)
		b.WriteString(strconv.Itoa(it.ID))
		b.WriteString(`,"slot":`)
		b.WriteString(strconv.Itoa(int(it.Slot)))
		b.WriteString(`,"pos":`)
		b.WriteString(strconv.Itoa(int(placeOf[it.ID])))
		alts, more := alternatives(positions, posOf, outfit.Items, it, met, st, skills, ratio, altLimit)
		b.WriteString(`,"alts":[`)
		for j, alt := range alts {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString(`{"id":`)
			b.WriteString(strconv.Itoa(alt.id))
			b.WriteString(`,"delta":`)
			b.WriteString(strconv.Itoa(alt.delta))
			b.WriteByte('}')
		}
		b.WriteString(`],"moreAlts":`)
		b.WriteString(strconv.FormatBool(more))
		b.WriteByte('}')
	}
	b.WriteString(`],"ownedPlaces":`)
	writeInts(&b, v.places)
	if len(missing) > 0 {
		b.WriteString(`,"missing":[`)
		for i, set := range missing {
			if i > 0 {
				b.WriteByte(',')
			}
			writeInts(&b, set)
		}
		b.WriteByte(']')
	}
	b.WriteByte('}')
	return result(b.String())
}

func attrCode(v js.Value) int {
	if v.IsUndefined() || v.IsNull() {
		return -1
	}
	return v.Int()
}

func (e *engine) known() int {
	if e.catalogue == nil {
		return 0
	}
	owns, n := e.pool(true), 0
	for _, id := range e.catalogue.IDs {
		if owns(id) {
			n++
		}
	}
	return n
}

func (e *engine) view(ownedOnly bool) *view {
	if v, ok := e.views[ownedOnly]; ok {
		return v
	}
	owns := e.pool(ownedOnly)
	v := &view{places: placesHeld(e.catalogue, owns)}
	v.positions, v.posOf, v.placeOf = optimizer.FromCatalogue(e.catalogue, owns)
	if e.views == nil {
		e.views = map[bool]*view{}
	}
	e.views[ownedOnly] = v
	return v
}

func (e *engine) pool(ownedOnly bool) func(id int32) bool {
	if !ownedOnly {
		return nil
	}
	have := make(map[int32]bool, len(e.owned))
	for _, id := range e.owned {
		have[int32(id)] = true
	}
	return func(id int32) bool { return have[id] }
}

func placesHeld(c *catalogue.Catalogue, owns func(id int32) bool) []uint16 {
	seen := map[uint16]bool{}
	var places []uint16
	for i, id := range c.IDs {
		if owns != nil && !owns(id) {
			continue
		}
		if place := c.Positions[i]; !seen[place] {
			seen[place] = true
			places = append(places, place)
		}
	}
	slices.Sort(places)
	return places
}

const altLimit = 5

type alternative struct {
	id    int
	delta int
}

func alternatives(positions []optimizer.Position, posOf map[int]int, worn []scoring.Item, chosen scoring.Item,
	met [][]int, st scoring.Stage, sk scoring.Skills, ratio float64, limit int) ([]alternative, bool) {
	idx, ok := posOf[chosen.ID]
	if !ok {
		return nil, false
	}
	rate := func(it scoring.Item) float64 {
		scaled, fixed := scoring.Contribution(it, st, sk)
		if it.Slot == scoring.Accessory {
			return ratio*scaled + fixed
		}
		return scaled + fixed
	}
	base := rate(chosen)

	var out []alternative
	for _, it := range optimizer.Ranked(positions[idx].Items, st, sk) {
		if it.ID == chosen.ID || !keepsRules(worn, chosen, it, met) {
			continue
		}
		if len(out) == limit {
			return out, true
		}
		out = append(out, alternative{it.ID, int(rate(it) - base)})
	}
	return out, false
}

func keepsRules(worn []scoring.Item, chosen, alt scoring.Item, met [][]int) bool {
	if len(met) == 0 {
		return true
	}
	swapped := make([]scoring.Item, len(worn))
	for i, it := range worn {
		if it.ID == chosen.ID {
			it = alt
		}
		swapped[i] = it
	}
	return len(optimizer.Unmet(swapped, met)) == 0
}

func bytesFrom(v js.Value) []byte {
	b := make([]byte, v.Get("byteLength").Int())
	js.CopyBytesToGo(b, v)
	return b
}

func result(json string) any { return map[string]any{"ok": true, "json": json} }
func fail(msg string) any    { return map[string]any{"ok": false, "error": msg} }
