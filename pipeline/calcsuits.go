package pipeline

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type CalcSuitStats struct {
	Suits, Ambiguous, Items, Unkeyed int
}

func ParseCalcSuits(table, keys []byte) (map[int][]string, CalcSuitStats, error) {
	var stats CalcSuitStats
	var doc struct {
		Names   []string `json:"names"`
		Offsets []int    `json:"offsets"`
		Counts  []int    `json:"counts"`
		Clothes []int    `json:"clothes"`
	}
	if err := json.Unmarshal(table, &doc); err != nil {
		return nil, stats, fmt.Errorf("pipeline: reading Nikki Calc's suits: %w", err)
	}
	if len(doc.Offsets) != len(doc.Names) || len(doc.Counts) != len(doc.Names) {
		return nil, stats, fmt.Errorf("pipeline: Nikki Calc's suits give %d names, %d offsets and %d counts",
			len(doc.Names), len(doc.Offsets), len(doc.Counts))
	}
	var byKey map[string]int
	if err := json.Unmarshal(keys, &byKey); err != nil {
		return nil, stats, fmt.Errorf("pipeline: reading item keys: %w", err)
	}
	idOf := make(map[int]int, len(byKey))
	for key, index := range byKey {
		if id, ok := NikkicalcID(key); ok {
			idOf[index] = id
		}
	}
	uses := map[string]int{}
	for _, n := range doc.Names {
		uses[suitName(n)]++
	}
	out := map[int][]string{}
	for k, raw := range doc.Names {
		name := suitName(raw)
		off, n := doc.Offsets[k], doc.Counts[k]
		if off < 0 || n < 0 || off+n > len(doc.Clothes) {
			return nil, stats, fmt.Errorf("pipeline: Nikki Calc's suit %q runs past its %d clothes", name, len(doc.Clothes))
		}
		if name == "" || uses[name] > 1 {
			stats.Ambiguous++
			continue
		}
		stats.Suits++
		for _, index := range doc.Clothes[off : off+n] {
			id, ok := idOf[index]
			if !ok {
				stats.Unkeyed++
				continue
			}
			if !slices.Contains(out[id], name) {
				out[id] = append(out[id], name)
			}
		}
	}
	stats.Items = len(out)
	return out, stats, nil
}

func foldSuit(s string) string {
	return strings.ToLower(suitName(shownSuit(suitName(s))))
}
