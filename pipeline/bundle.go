package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/danielradosa/nikkibase/core/catalogue"
	"github.com/danielradosa/nikkibase/core/scoring"
)

func WriteCatalogue(entries []Entry) ([]byte, error) {
	items := make([]catalogue.Item, 0, len(entries))
	for _, e := range entries {
		place, err := ResolvePosition(e.Position, e.Item.Slot)
		if err != nil {
			return nil, fmt.Errorf("catalogue: item %d (%s): %w", e.Item.ID, e.Name, err)
		}
		index, ok := PositionIndex(place.Name)
		if !ok {
			return nil, fmt.Errorf("catalogue: item %d (%s): %q has no index", e.Item.ID, e.Name, place.Name)
		}
		it := catalogue.Item{
			ID: int32(e.Item.ID), Slot: uint8(e.Item.Slot), Position: uint16(index),
			Group:     groupByte(place),
			FlatBonus: int32(e.Item.FlatBonus),
		}
		for p := range 5 {
			it.Attrs[p] = e.Item.Attrs[p]
			it.Stats[p] = int32(e.Item.Stats[p])
		}
		for _, t := range e.Item.Tags {
			it.Tags = append(it.Tags, int32(t))
		}
		items = append(items, it)
	}
	return catalogue.Write(items), nil
}

type ItemNames struct {
	Calc  map[int]string
	Shown map[int]string
}

func marshal(v any) []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
	return bytes.TrimSuffix(b.Bytes(), []byte("\n"))
}

func WriteItems(entries []Entry, names ItemNames, rarity map[int]int, suits map[int]string) []byte {
	rows := make([][]any, 0, len(entries))
	seen := make(map[int]bool, len(entries))
	for _, e := range entries {
		seen[e.Item.ID] = true
		row := []any{e.Item.ID, displayName(e, names), e.Item.Slot}
		for _, a := range e.Item.Attrs {
			row = append(row, a)
		}
		for _, g := range e.Grades {
			row = append(row, strings.ToUpper(g))
		}
		rows = append(rows, append(row, e.Rarity, e.Suit))
	}
	for _, id := range slices.Sorted(maps.Keys(names.Calc)) {
		if seen[id] {
			continue
		}
		name := gameSpelling(names.Calc[id])
		if shown, ok := names.Shown[id]; ok {
			name = gameSpelling(shown)
		}
		rows = append(rows, []any{id, name, SlotOfID(id), 0, 0, 0, 0, 0, "", "", "", "", "", rarity[id], suits[id]})
	}
	return marshal(struct {
		Items [][]any `json:"items"`
	}{rows})
}

func WriteStages(stages []Stage) []byte {
	var b strings.Builder
	b.WriteByte('[')
	for i, s := range stages {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"name":`)
		b.WriteString(quote(DisplayName(s)))
		if s.Mode != "" {
			b.WriteString(`,"mode":`)
			b.WriteString(quote(s.Mode))
		}
		b.WriteByte(',')
		writeScoring(&b, s.Stage)
		if s.Rules != nil {
			writeRules(&b, *s.Rules)
		}
		if len(s.Variants) > 0 {
			b.WriteString(`,"variants":{`)
			for j, name := range variantNames(s) {
				if j > 0 {
					b.WriteByte(',')
				}
				b.WriteString(quote(name))
				b.WriteString(":{")
				writeScoring(&b, s.Variants[name])
				if r, ok := s.VariantRules[name]; ok {
					writeRules(&b, r)
				}
				b.WriteByte('}')
			}
			b.WriteByte('}')
		}
		b.WriteString("}")
	}
	b.WriteByte(']')
	return []byte(b.String())
}

func writeRules(b *strings.Builder, r StageRules) {
	b.WriteString(`,"rules":{"styles":[`)
	for j, style := range r.Styles {
		if j > 0 {
			b.WriteByte(',')
		}
		b.WriteString(quote(style))
	}
	b.WriteByte(']')
	if len(r.Require) > 0 {
		b.WriteString(`,"require":[`)
		for j, set := range r.Require {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteByte('[')
			for k, id := range set {
				if k > 0 {
					b.WriteByte(',')
				}
				b.WriteString(strconv.Itoa(id))
			}
			b.WriteByte(']')
		}
		b.WriteByte(']')
	}
	b.WriteByte('}')
}

func writeScoring(b *strings.Builder, st scoring.Stage) {
	b.WriteString(`"weights":[`)
	for p, w := range st.Weights {
		if p > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(w, 'g', -1, 64))
	}
	b.WriteString(`],"attrs":[`)
	for p, a := range st.Attrs {
		if p > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(int(a)))
	}
	b.WriteByte(']')
	if len(st.Tags) > 0 {
		b.WriteString(`,"tags":{`)
		for j, id := range slices.Sorted(maps.Keys(st.Tags)) {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString(quote(strconv.Itoa(id)))
			b.WriteByte(':')
			b.WriteString(strconv.Itoa(st.Tags[id]))
		}
		b.WriteByte('}')
	}
}

func WriteTags() []byte {
	return marshal(TagNames())
}

func displayName(e Entry, names ItemNames) string {
	if name, ok := names.Shown[e.Item.ID]; ok {
		return gameSpelling(name)
	}
	return plainName(e, names)
}

func plainName(e Entry, names ItemNames) string {
	name := e.Name
	if hasHan(name) {
		if english, ok := names.Calc[e.Item.ID]; ok && english != "" && !hasHan(english) {
			name = english
		}
	}
	return gameName(gameSpelling(name), names.Calc[e.Item.ID])
}

func sourceName(e Entry, names ItemNames) string {
	spellings := []string{names.Calc[e.Item.ID]}
	if !hasHan(e.Name) {
		return NormalizeName(e.Name, spellings...)
	}
	if english, ok := names.Calc[e.Item.ID]; ok && english != "" && !hasHan(english) {
		return calcName(english)
	}
	return e.Name
}

func calcName(name string) string {
	return NormalizeName(name, name)
}

func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n', '\r', '\t':
			b.WriteString(`\u00`)
			b.WriteString(strconv.FormatInt(int64(r), 16))
		default:
			if r < 0x20 {
				continue
			}
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func WriteAcquisition(version string, acq map[int][]Acquisition) []byte {
	var b strings.Builder
	b.WriteString(`{"version":`)
	b.WriteString(quote(version))
	b.WriteString(`,"items":{`)
	first := true
	for _, id := range slices.Sorted(maps.Keys(acq)) {
		if len(acq[id]) == 0 {
			continue
		}
		if !first {
			b.WriteByte(',')
		}
		first = false
		b.WriteString(quote(strconv.Itoa(id)))
		b.WriteString(":[")
		for i, a := range acq[id] {
			if i > 0 {
				b.WriteByte(',')
			}
			writeAcquisition(&b, a)
		}
		b.WriteByte(']')
	}
	b.WriteString("}}")
	return []byte(b.String())
}

func writeAcquisition(b *strings.Builder, a Acquisition) {
	b.WriteString(`{"k":`)
	b.WriteString(quote(a.Kind))
	b.WriteString(`,"t":`)
	b.WriteString(quote(a.Text))
	if len(a.From) > 0 {
		b.WriteString(`,"from":[`)
		for i, in := range a.From {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteByte('[')
			b.WriteString(strconv.Itoa(in.ID))
			b.WriteByte(',')
			b.WriteString(strconv.Itoa(in.Qty))
			b.WriteByte(']')
		}
		b.WriteByte(']')
	}
	if len(a.Cost) > 0 {
		b.WriteString(`,"cost":[`)
		for i, c := range a.Cost {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteByte('[')
			b.WriteString(strconv.Itoa(c.Amount))
			b.WriteByte(',')
			b.WriteString(quote(c.Unit))
			b.WriteByte(']')
		}
		b.WriteByte(']')
	}
	for _, f := range []struct{ key, value string }{
		{"recipe", a.Recipe}, {"stage", a.Stage}, {"level", a.Level}, {"last", a.Last},
	} {
		if f.value != "" {
			b.WriteString(`,"`)
			b.WriteString(f.key)
			b.WriteString(`":`)
			b.WriteString(quote(f.value))
		}
	}
	if a.Past {
		b.WriteString(`,"past":1`)
	}
	if a.CN {
		b.WriteString(`,"cn":1`)
	}
	b.WriteByte('}')
}

func WritePositions() []byte {
	type position struct {
		Name string       `json:"name"`
		Slot scoring.Slot `json:"slot"`
	}
	out := make([]position, 0, len(subSlots))
	for _, s := range subSlots {
		out = append(out, position{s.Display, s.Slot})
	}
	return marshal(out)
}

func groupByte(s SubSlot) uint8 {
	g := uint8(GroupIndex(s.Group))
	if g != 0 && s.Exclusive() {
		g |= catalogue.WholeGroup
	}
	return g
}
