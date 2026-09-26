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

func LayerSuits(ids []int, wiki WikiAcquisition, packed map[int]PackedSuit) (map[int]string, SuitStats) {
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
