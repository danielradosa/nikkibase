package pipeline

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/danielradosa/nikkibase/core/scoring"
)

var subgradeBase = map[string]float64{
	"C": 40.4, "C+": 54.6,
	"B-": 67.2, "B": 81.4, "B+": 92.4,
	"A-": 102.8, "A": 113.8, "A+": 125.3,
	"S-": 128.5, "S": 140, "S+": 155.1,
	"SS-": 168.1, "SS": 183.2, "SS+": 200.1,
	"SSS": 213.3,
}

type Subgrade struct {
	Attr  int8
	Grade string
}

type SubgradeStats struct {
	Files, Records, Unkeyed, NotInCatalogue  int
	Applied, OtherSide, OtherLetter, Missing int
}

func SubStat(grade string, slot scoring.Slot) int {
	base, ok := subgradeBase[grade]
	if !ok {
		return 0
	}
	return int(math.Round(base * scoring.SlotSize(slot)))
}

func SubgradeLetter(grade string) string {
	return strings.TrimRight(grade, "+-")
}

type subgradeRecord struct {
	Slot     int      `json:"slot"`
	Attrs    []int    `json:"attrs"`
	NiGrades []string `json:"niGrades"`
	Tags     []int    `json:"tags"`
	Recipe   []struct {
		ID  int `json:"id"`
		Num int `json:"num"`
	} `json:"recipe"`
}

func ParseCalcItems(batches [][]byte, keys []byte) (map[int]CalcItem, SubgradeStats, error) {
	var stats SubgradeStats
	var byKey map[string]int
	if err := json.Unmarshal(keys, &byKey); err != nil {
		return nil, stats, fmt.Errorf("pipeline: reading item keys: %w", err)
	}
	keyOf := make(map[int]string, len(byKey))
	for key, index := range byKey {
		if _, ok := NikkicalcID(key); !ok {
			continue
		}
		if other, dup := keyOf[index]; dup {
			pair := []string{key, other}
			slices.Sort(pair)
			return nil, stats, fmt.Errorf("pipeline: item keys %s and %s both point at index %d", pair[0], pair[1], index)
		}
		keyOf[index] = key
	}

	out := map[int]CalcItem{}
	seen := map[int]bool{}
	for f, raw := range batches {
		var records map[string]subgradeRecord
		if err := json.Unmarshal(raw, &records); err != nil {
			return nil, stats, fmt.Errorf("pipeline: reading item batch %d of %d: %w", f+1, len(batches), err)
		}
		stats.Files++
		indices := make([]int, 0, len(records))
		for k := range records {
			index, err := strconv.Atoi(k)
			if err != nil {
				return nil, stats, fmt.Errorf("pipeline: item batch %d: record %q is not an item index", f+1, k)
			}
			indices = append(indices, index)
		}
		slices.Sort(indices)
		for _, index := range indices {
			r := records[strconv.Itoa(index)]
			if seen[index] {
				return nil, stats, fmt.Errorf("pipeline: index %d appears in two item batches", index)
			}
			seen[index] = true
			stats.Records++
			key, keyed := keyOf[index]
			row, err := subgradeRow(r)
			if err != nil {
				return nil, stats, fmt.Errorf("pipeline: item batch record %d: %w", index, err)
			}
			if !keyed {
				stats.Unkeyed++
				continue
			}
			id, _ := NikkicalcID(key)
			if _, dup := out[id]; dup {
				return nil, stats, fmt.Errorf("pipeline: two item keys give ID %d", id)
			}
			out[id] = CalcItem{Place: r.Slot, Row: row, Tags: r.Tags, Recipe: calcRecipe(r, keyOf)}
		}
	}
	return out, stats, nil
}

func calcRecipe(r subgradeRecord, keyOf map[int]string) []Ingredient {
	var out []Ingredient
	for _, in := range r.Recipe {
		id, ok := NikkicalcID(keyOf[in.ID])
		if !ok || in.Num <= 0 {
			return nil
		}
		out = append(out, Ingredient{ID: id, Qty: in.Num})
	}
	return out
}

func subgradeRow(r subgradeRecord) ([5]Subgrade, error) {
	var row [5]Subgrade
	if len(r.Attrs) != len(row) || len(r.NiGrades) != len(row) {
		return row, fmt.Errorf("%d attributes and %d sub-grades, want %d of each", len(r.Attrs), len(r.NiGrades), len(row))
	}
	for p := range row {
		a, g := r.Attrs[p], r.NiGrades[p]
		if a < 0 || a/2 != p {
			return row, fmt.Errorf("pair %d holds attribute %d, which belongs to no side of it", p, a)
		}
		if _, ok := subgradeBase[g]; g != "" && !ok {
			return row, fmt.Errorf("pair %d holds unknown sub-grade %q", p, g)
		}
		row[p] = Subgrade{Attr: int8(a), Grade: g}
	}
	return row, nil
}

func ApplySubgrades(entries []Entry, items map[int]CalcItem, stats *SubgradeStats) {
	matched := 0
	for i := range entries {
		e := &entries[i]
		it, ok := items[e.Item.ID]
		if ok {
			matched++
		}
		for p, sub := range it.Row {
			letter := strings.ToUpper(e.Grades[p])
			switch {
			case letter == "":
			case sub.Grade == "":
				stats.Missing++
			case sub.Attr != e.Item.Attrs[p]:
				stats.OtherSide++
			case SubgradeLetter(sub.Grade) != letter:
				stats.OtherLetter++
			default:
				e.Item.Stats[p] = SubStat(sub.Grade, e.Item.Slot)
				e.Subgrades[p] = sub.Grade
				stats.Applied++
			}
		}
	}
	stats.NotInCatalogue = len(items) - matched
}
