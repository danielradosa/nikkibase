package pipeline

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/danielradosa/nikkibase/core/scoring"
)

var (
	packedArray  = regexp.MustCompile(`(?s)var codewardrobe = \[(.*?)\n\];`)
	packedRow    = regexp.MustCompile(`'((?:[^'\\]|\\.)*)'`)
	packedTable  = `var %s = \[(.*?)\];`
	packedString = regexp.MustCompile(`'((?:[^'\\]|\\.)*)'`)
	packedBonus  = regexp.MustCompile(`^[^+]+\+(\d+)$`)
)

func letter2num(c byte) int {
	switch {
	case c == '-':
		return 62
	case c == '_':
		return 63
	case c <= '9':
		return int(c) - 48
	case c <= 'Z':
		return int(c) - 55
	default:
		return int(c) - 61
	}
}

func code2num(s string) int {
	n := 0
	for i := range len(s) {
		n = 64*n + letter2num(s[i])
	}
	return n
}

var num2stat = [12]struct {
	grade  string
	second bool
}{
	{"C", false}, {"B", false}, {"A", false}, {"S", false}, {"SS", false}, {"SSS", false},
	{"C", true}, {"B", true}, {"A", true}, {"S", true}, {"SS", true}, {"SSS", true},
}

var packedCategory = map[int]scoring.Slot{
	0: scoring.Hair, 1: scoring.Dress, 2: scoring.Coat, 3: scoring.Top,
	4: scoring.Bottom, 5: scoring.Hosiery, 6: scoring.Hosiery,
	7: scoring.Shoes, 32: scoring.Makeup, 33: scoring.Spirit,
}

type PackedStats struct {
	Rows       int
	Parsed     int
	UnknownAny int
	NotGlobal  int
	Bonuses    int
	UnknownTag int
}

func ParsePacked(src []byte, known map[int]bool) ([]Entry, PackedStats, error) {
	var stats PackedStats
	category, err := packedList(src, "category")
	if err != nil {
		return nil, stats, err
	}
	tags, err := packedList(src, "code2tag")
	if err != nil {
		return nil, stats, err
	}
	body := packedArray.FindSubmatch(src)
	if body == nil {
		return nil, stats, fmt.Errorf("pipeline: no codewardrobe array in the packed table")
	}

	var entries []Entry
	for _, m := range packedRow.FindAllSubmatch(body[1], -1) {
		stats.Rows++
		entry, ok := packedEntry(string(m[1]), category, tags, &stats)
		if !ok {
			continue
		}
		if len(known) > 0 && !known[entry.Item.ID] {
			stats.NotGlobal++
			continue
		}
		stats.Parsed++
		entries = append(entries, entry)
	}
	return entries, stats, nil
}

func packedList(src []byte, name string) ([]string, error) {
	re, err := regexp.Compile(fmt.Sprintf(packedTable, name))
	if err != nil {
		return nil, err
	}
	m := re.FindSubmatch(src)
	if m == nil {
		return nil, fmt.Errorf("pipeline: packed table has no %s array", name)
	}
	var out []string
	for _, s := range packedString.FindAllSubmatch(m[1], -1) {
		out = append(out, string(s[1]))
	}
	return out, nil
}

func packedEntry(row string, category, tagNames []string, stats *PackedStats) (Entry, bool) {
	w := strings.Split(row, "|")
	if len(w) < 5 || len(w[1]) < 2 {
		stats.UnknownAny++
		return Entry{}, false
	}
	ci := code2num(w[1][:1])
	id, slot, ok := packedRowID(w[1])
	if !ok {
		stats.UnknownAny++
		return Entry{}, false
	}

	it := scoring.Item{ID: id, Slot: slot}
	var letters [5]string
	code := strings.TrimPrefix(w[2], "*")
	num := code2num(code)
	for p := range 5 {
		s := num2stat[num%12]
		num /= 12
		first, second := packedCodes(p)
		attr := first
		if s.second {
			attr = second
		}
		it.Attrs[p], it.Stats[p], letters[p] = int8(attr), Stat(s.grade, slot), s.grade
	}

	if m := packedBonus.FindStringSubmatch(w[3]); m != nil {
		if v, err := strconv.Atoi(m[1]); err == nil {
			it.FlatBonus = v
			stats.Bonuses++
		}
	} else {
		for i := range len(w[3]) {
			at := code2num(w[3][i : i+1])
			if at >= len(tagNames) {
				stats.UnknownTag++
				continue
			}
			style, ok := StyleFromStageTag(tagNames[at])
			if !ok {
				stats.UnknownTag++
				continue
			}
			tid, ok := TagID(style)
			if !ok {
				stats.UnknownTag++
				continue
			}
			it.Tags = append(it.Tags, tid)
		}
	}

	position := strings.TrimSpace(category[min(ci, len(category)-1)])
	return Entry{Item: it, Name: strings.TrimSpace(w[0]), Position: position, Grades: letters}, true
}

func packedRowID(code string) (int, scoring.Slot, bool) {
	if len(code) < 2 {
		return 0, 0, false
	}
	ci := code2num(code[:1])
	slot, ok := packedCategory[ci]
	if !ok {
		if ci < 8 || ci > 31 {
			return 0, 0, false
		}
		slot = scoring.Accessory
	}
	id, ok := packedID(slot, code2num(code[1:]))
	return id, slot, ok
}

type PackedSource struct {
	Kind  string
	Value string
	ID    int
}

func ParsePackedSources(src []byte, known map[int]bool) (map[int][]PackedSource, error) {
	suits, err := packedList(src, "code2suit")
	if err != nil {
		return nil, err
	}
	codes, err := packedList(src, "code2src")
	if err != nil {
		return nil, err
	}
	body := packedArray.FindSubmatch(src)
	if body == nil {
		return nil, fmt.Errorf("pipeline: no codewardrobe array in the packed table")
	}
	rows := packedRows(body[1])
	out := map[int][]PackedSource{}
	for _, m := range packedRow.FindAllSubmatch(body[1], -1) {
		w := strings.Split(string(m[1]), "|")
		if len(w) < 7 {
			continue
		}
		id, slot, ok := packedRowID(w[1])
		if !ok || (len(known) > 0 && !known[id]) {
			continue
		}
		if _, seen := out[id]; seen {
			continue
		}
		list := []PackedSource{}
		for _, t := range strings.Split(w[6], "/") {
			if t == "" {
				continue
			}
			var s PackedSource
			switch {
			case t[0] == '*':
				n := code2num(t[1:])
				if n >= len(suits) {
					return nil, fmt.Errorf("pipeline: packed item %d names suit %d, and code2suit has %d", id, n, len(suits))
				}
				s = PackedSource{Kind: "suit", Value: suits[n]}
			case t[0] == '@' || t[0] == '!':
				from, ok := packedID(slot, code2num(t[1:]))
				if !ok {
					continue
				}
				s = PackedSource{Kind: "customize", ID: from}
				if t[0] == '!' {
					s.Kind = "evolve"
				}
				if rows.crossWired(id, from) {
					s.ID = 0
				}
			case t[0] == '~':
				s = PackedSource{Kind: "dream", Value: t[1:]}
			case packedIsCode(t):
				n := code2num(t)
				if n >= len(codes) {
					return nil, fmt.Errorf("pipeline: packed item %d names source %d, and code2src has %d", id, n, len(codes))
				}
				s = PackedSource{Kind: "code", Value: codes[n]}
			default:
				s = PackedSource{Kind: "text", Value: t}
			}
			list = append(list, s)
		}
		out[id] = list
	}
	return out, nil
}

type packedRowInfo struct {
	name     string
	category byte
	slot     scoring.Slot
}

type packedRowSet struct {
	rows     map[int]packedRowInfo
	families map[string]int
}

func packedRows(body []byte) packedRowSet {
	set := packedRowSet{rows: map[int]packedRowInfo{}, families: map[string]int{}}
	for _, m := range packedRow.FindAllSubmatch(body, -1) {
		w := strings.Split(string(m[1]), "|")
		if len(w) < 2 {
			continue
		}
		id, slot, ok := packedRowID(w[1])
		if !ok {
			continue
		}
		if _, seen := set.rows[id]; seen {
			continue
		}
		name := strings.TrimSpace(w[0])
		set.rows[id] = packedRowInfo{name: name, category: w[1][0], slot: slot}
		set.families[packedFamily(slot, name)]++
	}
	return set
}

func packedStem(name string) string {
	stem, _, _ := strings.Cut(name, "·")
	return strings.TrimSpace(stem)
}

func packedFamily(slot scoring.Slot, name string) string {
	return strconv.Itoa(int(slot)) + "|" + packedStem(name)
}

func (s packedRowSet) crossWired(id, base int) bool {
	item, ok := s.rows[id]
	from, found := s.rows[base]
	if !ok || !found || item.category == from.category || packedStem(item.name) == packedStem(from.name) {
		return false
	}
	return strings.Contains(item.name, "·") && s.families[packedFamily(item.slot, item.name)] > 1
}

func packedIsCode(s string) bool {
	if s == "" || s[0] >= 0x80 {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !(c == '!' || c == '*' || c == '-' || c == '_' || (c >= '0' && c <= '9') || (c >= '@' && c <= 'Z') || (c >= 'a' && c <= 'z')) {
			return false
		}
	}
	return true
}

func packedCodes(p int) (first, second int) {
	if p == 4 {
		return scoring.Cool, scoring.Warm
	}
	return p * 2, p*2 + 1
}

func packedID(slot scoring.Slot, n int) (int, bool) {
	switch slot {
	case scoring.Spirit:
		return 880000 + n, true
	case scoring.Accessory:
		if n >= 10000 {
			return 170000 + n, true
		}
		return idPrefix[scoring.Accessory]*10000 + n, true
	}
	p, ok := idPrefix[slot]
	if !ok {
		return 0, false
	}
	return p*10000 + n, true
}
