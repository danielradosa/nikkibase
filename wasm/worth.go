//go:build js && wasm

package main

import (
	"math"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/danielradosa/nikkibase/core/scoring"
	"github.com/danielradosa/nikkibase/core/worth"
)

const rankLimit = 1000

func (e *engine) worthStart(_ js.Value, args []js.Value) any {
	if e.catalogue == nil {
		return fail("catalogue not loaded")
	}
	if len(args) == 0 {
		return fail("the stages must be a list")
	}
	versions, msg := worthVersions(args[0])
	if msg != "" {
		return fail(msg)
	}
	var settings worth.Settings
	if len(args) > 1 && !args[1].IsNull() && !args[1].IsUndefined() {
		s := args[1]
		if !isObject(s) {
			return fail("skills must be null or {auto: true}")
		}
		if s.Get("auto").Equal(js.ValueOf(true)) {
			l, ok := worthLevels(s.Get("levels"))
			if !ok {
				return fail("skill levels run from 0 to 9")
			}
			settings = worth.Settings{Skills: true, Levels: l}
		}
	}
	suits, msg := worthSuits(arg(args, 2))
	if msg != "" {
		return fail(msg)
	}
	settings.Suits = suits
	v := e.view(false)
	e.worth = worth.NewSession(v.positions, v.posOf, e.pool(true), versions, settings)
	if e.session == 0 {
		e.session = int(js.Global().Get("Date").Call("now").Float())
	}
	e.session++
	return result(`{"session":` + strconv.Itoa(e.session) + `,"total":` + strconv.Itoa(len(versions)) + `}`)
}

func (e *engine) worthRun(_ js.Value, args []js.Value) any {
	if !e.live(args) {
		return fail("no session")
	}
	count, ok := whole(arg(args, 1))
	if !ok || count < 0 {
		return fail("the count must be a whole number")
	}
	done, total := e.worth.Run(count)
	return result(`{"done":` + strconv.Itoa(done) + `,"total":` + strconv.Itoa(total) + `}`)
}

func (e *engine) worthRank(_ js.Value, args []js.Value) any {
	if !e.live(args) {
		return fail("no session")
	}
	f, places, msg := worthFilter(arg(args, 1))
	if msg != "" {
		return fail(msg)
	}
	limit, ok := whole(arg(args, 2))
	if !ok || limit < 0 || limit > rankLimit {
		return fail("the limit runs from 0 to " + strconv.Itoa(rankLimit))
	}
	v := e.view(false)
	placeOf := v.placeOf
	for _, place := range places {
		at := -1
		for k, p := range v.positions {
			if len(p.Items) > 0 && placeOf[p.Items[0].ID] == place {
				at = k
				break
			}
		}
		f.Positions = append(f.Positions, at)
	}
	r := e.worth.Rank(f, limit)

	var b strings.Builder
	b.WriteString(`{"rows":[`)
	for i, row := range r.Rows {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('{')
		if row.Suit != "" {
			b.WriteString(`"suit":`)
			writeString(&b, row.Suit)
			b.WriteByte(',')
		}
		b.WriteString(`"items":`)
		writeInts(&b, row.Items)
		b.WriteString(`,"places":[`)
		for j, id := range row.Items {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString(strconv.Itoa(int(placeOf[id])))
		}
		b.WriteString(`],"worth":`)
		b.WriteString(strconv.FormatFloat(row.Worth, 'f', 3, 64))
		b.WriteString(`,"stages":`)
		b.WriteString(strconv.Itoa(row.Stages))
		b.WriteString(`,"best":`)
		writeExample(&b, row.Best)
		b.WriteString(`,"examples":[`)
		for j, ex := range row.Examples {
			if j > 0 {
				b.WriteByte(',')
			}
			writeExample(&b, ex)
		}
		b.WriteString(`]}`)
	}
	b.WriteString(`],"needed":[`)
	for i, n := range r.Needed {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"key":`)
		writeString(&b, n.Key)
		b.WriteString(`,"missing":[`)
		for j, set := range n.Missing {
			if j > 0 {
				b.WriteByte(',')
			}
			writeInts(&b, set)
		}
		b.WriteString(`]}`)
	}
	b.WriteString(`]}`)
	return result(b.String())
}

func (e *engine) live(args []js.Value) bool {
	id, ok := whole(arg(args, 0))
	return ok && e.worth != nil && id == e.session
}

func arg(args []js.Value, i int) js.Value {
	if i < len(args) {
		return args[i]
	}
	return js.Undefined()
}

func worthVersions(list js.Value) ([]worth.Version, string) {
	if !isArray(list) {
		return nil, "the stages must be a list"
	}
	versions := make([]worth.Version, 0, list.Length())
	for i := range list.Length() {
		v, msg := worthVersion(list.Index(i))
		if msg != "" {
			return nil, "stage " + strconv.Itoa(i+1) + ": " + msg
		}
		versions = append(versions, v)
	}
	return versions, ""
}

func worthVersion(o js.Value) (worth.Version, string) {
	var v worth.Version
	if !isObject(o) {
		return v, "not an object"
	}
	key, okKey := text(o.Get("key"))
	mode, okMode := text(o.Get("mode"))
	if !okKey || !okMode {
		return v, "the key and the mode must be text"
	}
	v.Key, v.Mode = key, mode
	weights, attrs := o.Get("weights"), o.Get("attrs")
	if !isArray(weights) || weights.Length() != 5 || !isArray(attrs) || attrs.Length() != 5 {
		return v, "weights and attrs must be five numbers each"
	}
	for p := range 5 {
		w, ok := number(weights.Index(p))
		if !ok {
			return v, "weights and attrs must be five numbers each"
		}
		a, ok := whole(attrs.Index(p))
		if !ok || a < scoring.Gorgeous || a > scoring.Cool {
			return v, "attrs must be attribute codes from 0 to 9"
		}
		v.Stage.Weights[p], v.Stage.Attrs[p] = w, int8(a)
	}
	if tags := o.Get("tags"); !tags.IsNull() && !tags.IsUndefined() {
		if !isObject(tags) {
			return v, "tags must map tag numbers to points"
		}
		v.Stage.Tags = map[int]int{}
		entries := js.Global().Get("Object").Call("entries", tags)
		for i := range entries.Length() {
			pair := entries.Index(i)
			name, _ := text(pair.Index(0))
			id, err := strconv.Atoi(name)
			award, ok := whole(pair.Index(1))
			if err != nil || !ok {
				return v, "tags must map tag numbers to points"
			}
			v.Stage.Tags[id] = award
		}
	}
	if require := o.Get("require"); !require.IsNull() && !require.IsUndefined() {
		if !isArray(require) {
			return v, "require must be a list of item lists"
		}
		for i := range require.Length() {
			set := require.Index(i)
			if !isArray(set) || set.Length() == 0 {
				return v, "require must be a list of item lists"
			}
			ids := make([]int, 0, set.Length())
			for j := range set.Length() {
				id, ok := whole(set.Index(j))
				if !ok {
					return v, "require must be a list of item lists"
				}
				ids = append(ids, id)
			}
			v.Require = append(v.Require, ids)
		}
	}
	ideal, ok := whole(o.Get("ideal"))
	if !ok || ideal < 1 {
		return v, "ideal must be the best possible score, a whole number above 0"
	}
	v.Ideal = ideal
	return v, ""
}

func worthSuits(list js.Value) ([]worth.Suit, string) {
	if list.IsNull() || list.IsUndefined() {
		return nil, ""
	}
	if !isArray(list) {
		return nil, "suits must be a list"
	}
	suits := make([]worth.Suit, 0, list.Length())
	for i := range list.Length() {
		o := list.Index(i)
		at := "suit " + strconv.Itoa(i+1) + ": "
		if !isObject(o) {
			return nil, at + "not an object"
		}
		key, ok := text(o.Get("key"))
		if !ok {
			return nil, at + "the key must be text"
		}
		items := o.Get("items")
		if !isArray(items) {
			return nil, at + "items must be a list of item numbers"
		}
		ids := make([]int, 0, items.Length())
		for j := range items.Length() {
			id, ok := whole(items.Index(j))
			if !ok {
				return nil, at + "items must be a list of item numbers"
			}
			ids = append(ids, id)
		}
		suits = append(suits, worth.Suit{Key: key, Items: ids})
	}
	return suits, ""
}

func worthFilter(o js.Value) (worth.Filter, []uint16, string) {
	var f worth.Filter
	if !isObject(o) {
		return f, nil, "the filter must be an object"
	}
	var ok bool
	if f.Modes, ok = texts(o.Get("modes")); !ok {
		return f, nil, "modes must be a list of mode names"
	}
	if f.Skip, ok = texts(o.Get("skip")); !ok {
		return f, nil, "skip must be a list of stage keys"
	}
	if slots := o.Get("slots"); !slots.IsNull() && !slots.IsUndefined() {
		if !isArray(slots) {
			return f, nil, "slots must be a list of slot numbers from 0 to 9"
		}
		for i := range slots.Length() {
			s, ok := whole(slots.Index(i))
			if !ok || s < int(scoring.Hair) || s > int(scoring.Spirit) {
				return f, nil, "slots must be a list of slot numbers from 0 to 9"
			}
			f.Slots = append(f.Slots, scoring.Slot(s))
		}
	}
	if suits := o.Get("suits"); !suits.IsNull() && !suits.IsUndefined() {
		switch {
		case suits.Equal(js.ValueOf(true)):
			f.Suits = true
		case !suits.Equal(js.ValueOf(false)):
			return f, nil, "suits must be true or false"
		}
	}
	var places []uint16
	if list := o.Get("places"); !list.IsNull() && !list.IsUndefined() {
		if !isArray(list) {
			return f, nil, "places must be a list of place numbers"
		}
		for i := range list.Length() {
			p, ok := whole(list.Index(i))
			if !ok || p < 0 || p > math.MaxUint16 {
				return f, nil, "places must be a list of place numbers"
			}
			places = append(places, uint16(p))
		}
	}
	return f, places, ""
}

func texts(v js.Value) ([]string, bool) {
	if v.IsNull() || v.IsUndefined() {
		return nil, true
	}
	if !isArray(v) {
		return nil, false
	}
	out := make([]string, 0, v.Length())
	for i := range v.Length() {
		s, ok := text(v.Index(i))
		if !ok {
			return nil, false
		}
		out = append(out, s)
	}
	return out, true
}

func worthLevels(v js.Value) (scoring.Levels, bool) {
	l := scoring.MaxLevels
	if v.IsUndefined() || v.IsNull() {
		return l, true
	}
	if !isObject(v) {
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
		level, ok := whole(n)
		if !ok {
			return l, false
		}
		*f.to = level
	}
	return l, l.Valid()
}

var (
	arrayCheck = js.Global().Get("Array").Get("isArray")
	finite     = js.Global().Get("Number").Get("isFinite")
	objectTag  = js.Global().Get("Object").Get("prototype").Get("toString")
)

func isArray(v js.Value) bool {
	return arrayCheck.Invoke(v).Bool()
}

func tagOf(v js.Value) string {
	return objectTag.Call("call", v).String()
}

func isObject(v js.Value) bool {
	return tagOf(v) == "[object Object]"
}

func text(v js.Value) (string, bool) {
	if tagOf(v) != "[object String]" || v.Type() != js.TypeString {
		return "", false
	}
	return v.String(), true
}

func number(v js.Value) (float64, bool) {
	if !finite.Invoke(v).Bool() {
		return 0, false
	}
	return v.Float(), true
}

func whole(v js.Value) (int, bool) {
	f, ok := number(v)
	if !ok || f != math.Trunc(f) || math.Abs(f) > 1<<53 {
		return 0, false
	}
	return int(f), true
}

func writeInts(b *strings.Builder, ids []int) {
	b.WriteByte('[')
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(id))
	}
	b.WriteByte(']')
}

func writeExample(b *strings.Builder, ex worth.Example) {
	b.WriteString(`{"key":`)
	writeString(b, ex.Key)
	b.WriteString(`,"points":`)
	b.WriteString(strconv.Itoa(ex.Points))
	b.WriteString(`,"pct":`)
	b.WriteString(strconv.FormatFloat(ex.Pct, 'f', 3, 64))
	b.WriteByte('}')
}

func writeString(b *strings.Builder, s string) {
	const hex = "0123456789abcdef"
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '"' || c == '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		case c < 0x20:
			b.WriteString(`\u00`)
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&15])
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
}
