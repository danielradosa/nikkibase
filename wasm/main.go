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
	b.WriteString(`{"ids":[`)
	for i, id := range e.catalogue.IDs {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(int(id)))
	}
	b.WriteString(`],"places":[`)
	for i, place := range e.catalogue.Positions {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(int(place)))
	}
	b.WriteString(`]}`)
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
	for i, id := range w.Items {
		if i > 0 {
			ids.WriteByte(',')
		}
		ids.WriteString(strconv.Itoa(id))
	}
	return result(`{"items":` + strconv.Itoa(len(w.Items)) +
		`,"unresolved":` + strconv.Itoa(w.Unresolved) +
		`,"known":` + strconv.Itoa(e.known()) +
		`,"ids":[` + ids.String() + `]}`)
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

func (e *engine) best(_ js.Value, args []js.Value) any {
	if e.catalogue == nil {
		return fail("catalogue not loaded")
	}
	var st scoring.Stage
	weights, attrs := args[0], args[1]
	for p := range 5 {
		st.Weights[p] = weights.Index(p).Float()
		st.Attrs[p] = int8(attrs.Index(p).Int())
	}
	if len(args) > 3 && !args[3].IsNull() && !args[3].IsUndefined() {
		st.Tags = map[int]int{}
		entries := js.Global().Get("Object").Call("entries", args[3])
		for i := range entries.Length() {
			pair := entries.Index(i)
			id, err := strconv.Atoi(pair.Index(0).String())
			if err != nil {
				continue
			}
			st.Tags[id] = pair.Index(1).Int()
		}
	}

	ownedOnly := true
	if len(args) > 4 && args[4].String() == "all" {
		ownedOnly = false
	}
	var required [][]int
	if len(args) > 5 && !args[5].IsNull() && !args[5].IsUndefined() {
		for i := range args[5].Length() {
			set := args[5].Index(i)
			ids := make([]int, 0, set.Length())
			for j := range set.Length() {
				ids = append(ids, set.Index(j).Int())
			}
			required = append(required, ids)
		}
	}
	v := e.view(ownedOnly)
	positions, posOf, placeOf := v.positions, v.posOf, v.placeOf
	space := optimizer.Require(positions, posOf, required)

	var outfit optimizer.Result
	var placement *scoring.Placement
	skills := scoring.Skills{}
	chosen := len(args) > 2 && !args[2].IsNull() && !args[2].IsUndefined()
	levels := scoring.MaxLevels
	if chosen {
		var ok bool
		if levels, ok = skillLevels(args[2].Get("levels")); !ok {
			return fail("skill levels run from 0 to 9")
		}
	}
	switch {
	case !chosen || levels.None():
		outfit = space.Best(st, skills)
	case args[2].Get("auto").Truthy():
		var p scoring.Placement
		outfit, p = space.BestPlacedAt(st, levels)
		if levels.Smile == 0 {
			p.Smile = -1
		}
		placement, skills = &p, p.SkillsAt(levels)
	default:
		p := scoring.Placement{CharmSmile: attrCode(args[2].Get("charmSmile")), Smile: attrCode(args[2].Get("smile"))}
		if p.Smile == p.CharmSmile || levels.Smile == 0 {
			p.Smile = -1
		}
		placement, skills = &p, p.SkillsAt(levels)
		outfit = space.Best(st, skills)
	}
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
	b.WriteString(`],"ownedPlaces":[`)
	for i, place := range v.places {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(int(place)))
	}
	b.WriteByte(']')
	if len(missing) > 0 {
		b.WriteString(`,"missing":[`)
		for i, set := range missing {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteByte('[')
			for j, id := range set {
				if j > 0 {
					b.WriteByte(',')
				}
				b.WriteString(strconv.Itoa(id))
			}
			b.WriteByte(']')
		}
		b.WriteByte(']')
	}
	b.WriteByte('}')
	return result(b.String())
}

func skillLevels(v js.Value) (scoring.Levels, bool) {
	l := scoring.MaxLevels
	if v.IsUndefined() || v.IsNull() {
		return l, true
	}
	if v.Type() != js.TypeObject {
		return l, false
	}
	for _, f := range []struct {
		name string
		to   *int
	}{{"charming", &l.Charming}, {"smile", &l.Smile}} {
		n := v.Get(f.name)
		if n.IsUndefined() || n.IsNull() {
			continue
		}
		if n.Type() != js.TypeNumber || n.Float() != float64(int(n.Float())) {
			return l, false
		}
		*f.to = int(n.Float())
	}
	return l, l.Valid()
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
	have := make(map[int32]bool, len(e.owned))
	for _, id := range e.owned {
		have[int32(id)] = true
	}
	n := 0
	for _, id := range e.catalogue.IDs {
		if have[id] {
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
