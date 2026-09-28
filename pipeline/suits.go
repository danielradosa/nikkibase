package pipeline

import (
	"fmt"
	"strings"
)

type PackedSuit struct {
	Name string
	Base bool
}

func ParsePackedSuits(src []byte, known map[int]bool) (map[int]PackedSuit, error) {
	suits, err := packedList(src, "code2suit")
	if err != nil {
		return nil, err
	}
	body := packedArray.FindSubmatch(src)
	if body == nil {
		return nil, fmt.Errorf("pipeline: no codewardrobe array in the packed table")
	}
	out := map[int]PackedSuit{}
	seen := map[int]bool{}
	for _, m := range packedRow.FindAllSubmatch(body[1], -1) {
		w := strings.Split(string(m[1]), "|")
		if len(w) < 5 {
			continue
		}
		id, _, ok := packedRowID(w[1])
		if !ok || (len(known) > 0 && !known[id]) || seen[id] {
			continue
		}
		seen[id] = true
		code := w[4]
		if code == "" {
			continue
		}
		var s PackedSuit
		switch code[0] {
		case '!':
			s.Base = true
			code = code[1:]
		case '*', '@':
			code = code[1:]
		}
		n := code2num(code)
		if code == "" || n >= len(suits) {
			return nil, fmt.Errorf("pipeline: packed item %d names suit %q, and code2suit has %d", id, w[4], len(suits))
		}
		s.Name = strings.TrimSpace(suits[n])
		out[id] = s
	}
	return out, nil
}

type SuitStats struct {
	Wiki    int
	Late    int
	Members int
	Calc    int
	Packs   int
	Packed  int
	Bases   int
	Unnamed int
	None    int
	Suits   int
}

func suitName(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func LayerSuits(ids []int, wiki WikiAcquisition, packed map[int]PackedSuit, calc map[int][]string) (map[int]string, SuitStats) {
	var stats SuitStats
	out := make(map[int]string, len(wiki.SuitOf))
	for _, id := range ids {
		s := suitName(wiki.SuitOf[id])
		if wiki.Packs[s] {
			stats.Packs++
			s = ""
		}
		if s != "" {
			out[id] = s
			stats.Wiki++
			continue
		}
		if p, ok := packed[id]; ok && !p.Base {
			if s := suitName(wiki.ChineseSuits[p.Name]); s != "" && !wiki.Packs[s] {
				out[id] = s
				stats.Packed++
			}
		}
	}
	inCatalogue := make(map[int]bool, len(ids))
	for _, id := range ids {
		inCatalogue[id] = true
	}
	for _, part := range wiki.Unplaced {
		if id := placeLate(out, part, inCatalogue); id != 0 {
			out[id] = suitName(part.Suit)
			stats.Late++
		}
	}
	stats.Members = placeByMembers(out, ids, packed)
	stats.Calc = placeByCalc(out, ids, wiki.Packs, packed, calc)
	for _, id := range ids {
		if out[id] != "" {
			continue
		}
		p, ok := packed[id]
		switch {
		case !ok:
			stats.None++
		case p.Base:
			stats.Bases++
		default:
			stats.Unnamed++
		}
	}
	names := map[string]bool{}
	for _, s := range out {
		names[s] = true
	}
	stats.Suits = len(names)
	return out, stats
}

func placeByMembers(out map[int]string, ids []int, packed map[int]PackedSuit) int {
	held := map[string]map[string]int{}
	for _, id := range ids {
		p, ok := packed[id]
		if !ok || p.Base || out[id] == "" {
			continue
		}
		if held[p.Name] == nil {
			held[p.Name] = map[string]int{}
		}
		held[p.Name][out[id]]++
	}
	placed := 0
	for _, id := range ids {
		p, ok := packed[id]
		if !ok || p.Base || out[id] != "" || len(held[p.Name]) != 1 {
			continue
		}
		for suit, n := range held[p.Name] {
			if n >= 2 {
				out[id] = suit
				placed++
			}
		}
	}
	return placed
}

func placeByCalc(out map[int]string, ids []int, packs map[string]bool, packed map[int]PackedSuit, calc map[int][]string) int {
	if len(calc) == 0 {
		return 0
	}
	keys := map[string]string{}
	for _, s := range out {
		keys[foldSuit(s)] = s
	}
	pack := map[string]bool{}
	for s := range packs {
		pack[foldSuit(s)] = true
	}
	given := map[string]map[string]bool{}
	holds := map[string]map[string]bool{}
	for _, id := range ids {
		p, ok := packed[id]
		if !ok || p.Base {
			continue
		}
		if out[id] != "" {
			if holds[p.Name] == nil {
				holds[p.Name] = map[string]bool{}
			}
			holds[p.Name][out[id]] = true
		}
		if names := calc[id]; len(names) == 1 {
			if given[p.Name] == nil {
				given[p.Name] = map[string]bool{}
			}
			given[p.Name][names[0]] = true
		}
	}
	named := map[string]string{}
	claims := map[string]int{}
	for chinese, names := range given {
		if len(names) != 1 {
			continue
		}
		for name := range names {
			key, clash := keys[foldSuit(name)]
			held := holds[chinese]
			switch {
			case pack[foldSuit(name)] || len(held) > 1:
			case len(held) == 1 && held[key]:
				named[chinese] = key
			case len(held) == 0 && !clash:
				named[chinese] = name
				claims[foldSuit(name)]++
			}
		}
	}
	placed := 0
	for _, id := range ids {
		p, ok := packed[id]
		if !ok || p.Base || out[id] != "" {
			continue
		}
		if name := named[p.Name]; name != "" && claims[foldSuit(name)] <= 1 {
			out[id] = name
			keys[foldSuit(name)] = name
			placed++
		}
	}
	for _, id := range ids {
		if _, inTable := packed[id]; inTable || out[id] != "" || len(calc[id]) != 1 {
			continue
		}
		name := calc[id][0]
		if pack[foldSuit(name)] {
			continue
		}
		if key, ok := keys[foldSuit(name)]; ok {
			name = key
		}
		out[id] = name
		keys[foldSuit(name)] = name
		placed++
	}
	return placed
}

func placeLate(out map[int]string, part SuitPart, inCatalogue map[int]bool) int {
	suit := suitName(part.Suit)
	free := 0
	for _, id := range part.IDs {
		if strings.EqualFold(out[id], suit) {
			return 0
		}
		if out[id] == "" && inCatalogue[id] {
			if free != 0 {
				return 0
			}
			free = id
		}
	}
	return free
}

func ApplySuits(entries []Entry, suits map[int]string) {
	for i := range entries {
		entries[i].Suit = suits[entries[i].Item.ID]
	}
}

func CheckSuits(suits map[int]string, want Coverage) Violations {
	var v Violations
	if len(suits) < want.SuitItems {
		v = append(v, fmt.Sprintf("%d items are in a suit, below the committed floor of %d", len(suits), want.SuitItems))
	}
	var foreign []string
	for _, id := range sortedIDs(suits) {
		if hasHan(suits[id]) {
			foreign = append(foreign, fmt.Sprintf("%d %q", id, suits[id]))
		}
	}
	if len(foreign) > 0 {
		v = append(v, fmt.Sprintf("%d suit names are not English: %s", len(foreign), truncate(foreign)))
	}
	return v
}
