package pipeline

import (
	"fmt"
	"slices"
	"strings"

	"github.com/danielradosa/nikkibase/core/scoring"
)

type CalcItem struct {
	Place  int
	Row    [5]Subgrade
	Tags   []int
	Recipe []Ingredient
}

type CalcStats struct {
	Candidates, Graded, Tagged, UnknownTags int
	NoRecord, Incomplete                    int
	Unplaced                                []int
}

var calcPlaces = map[int]string{
	0: "hair", 1: "hair",
	2: "dress", 3: "dress", 4: "dress", 40: "dress", 53: "dress", 55: "dress", 61: "dress",
	5: "coat", 6: "coat", 47: "coat", 51: "coat", 62: "coat", 64: "coat",
	7: "top", 8: "top", 9: "top", 63: "top",
	10: "bottom", 11: "bottom", 12: "bottom", 56: "bottom",
	13: "hosiery_leglet", 14: "hosiery_socks",
	15: "shoes", 16: "shoes",
	17: "makeup",
	18: "accessory_hair_ornament",
	19: "accessory_veil", 30: "accessory_veil",
	42: "accessory_hairpin",
	43: "accessory_ears",
	20: "accessory_earrings",
	21: "accessory_scarf",
	22: "accessory_necklace", 23: "accessory_necklace",
	24: "accessory_hand_right", 38: "accessory_hand_right",
	25: "accessory_hand_left", 39: "accessory_hand_left",
	26: "accessory_hand_both", 44: "accessory_hand_both", 45: "accessory_hand_both", 52: "accessory_hand_both",
	66: "accessory_hand_both", 67: "accessory_hand_both",
	27: "accessory_handheld_right",
	28: "accessory_handheld_left",
	57: "accessory_handheld_both", 65: "accessory_handheld_both",
	29: "accessory_waist",
	31: "accessory_face",
	32: "accessory_brooch", 68: "accessory_brooch",
	33: "accessory_tattoo",
	34: "accessory_wings", 41: "accessory_wings",
	35: "accessory_tail", 46: "accessory_tail",
	36: "accessory_foreground",
	37: "accessory_background", 54: "accessory_background",
	48: "accessory_headwear",
	49: "accessory_skin",
	50: "accessory_ground",
	60: "spirit",
}

var calcTags = [...]string{
	"Sports", "POP", "Homewear", "Chinese Classical", "Chic", "Preppy", "Unisex", "Fairy", "European", "Workwear",
	"Pajamas", "Dancer", "Britain", "Musician", "Dryad", "Goddess", "Shower", "Animal", "Rain", "Swimsuit",
	"Floral", "Sun Care", "Apron", "Paramedics", "Evening Gown", "Bunny", "Gothic", "Lady", "Maiden", "Winter",
	"Bohemia", "Swordsman", "Modern China", "Kimono", "Lolita", "Wedding", "Republic of China", "Cheongsam",
	"Multicultural", "Traditional", "Army", "Sailor", "Future", "Street",
}

func CalcEntries(items map[int]CalcItem, names map[int]string, have map[int]bool) ([]Entry, CalcStats) {
	var stats CalcStats
	ids := make([]int, 0, len(names))
	for id := range names {
		if !have[id] {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	var out []Entry
	for _, id := range ids {
		stats.Candidates++
		r, ok := items[id]
		if !ok {
			stats.NoRecord++
			continue
		}
		if slices.ContainsFunc(r.Row[:], func(s Subgrade) bool { return s.Grade == "" }) {
			stats.Incomplete++
			continue
		}
		slot := SlotOfID(id)
		place, known := calcPlaces[r.Place]
		sub, err := ResolvePosition(place, slot)
		if !known || err != nil {
			stats.Unplaced = append(stats.Unplaced, id)
			continue
		}
		e := Entry{Item: scoring.Item{ID: id, Slot: slot}, Name: strings.TrimSpace(names[id]), Position: sub.Name}
		for p, s := range r.Row {
			e.Grades[p] = SubgradeLetter(s.Grade)
			e.Item.Attrs[p] = s.Attr
			e.Item.Stats[p] = Stat(e.Grades[p], slot)
		}
		for _, t := range r.Tags {
			if t < 0 || t >= len(calcTags) {
				stats.UnknownTags++
				continue
			}
			tag, ok := TagID(calcTags[t])
			if !ok {
				stats.UnknownTags++
				continue
			}
			if !slices.Contains(e.Item.Tags, tag) {
				e.Item.Tags = append(e.Item.Tags, tag)
			}
		}
		if len(e.Item.Tags) > 0 {
			stats.Tagged++
		}
		out = append(out, e)
	}
	stats.Graded = len(out)
	return out, stats
}

func CheckCalcGrades(stats CalcStats, want Coverage) Violations {
	if stats.Graded > want.MaxCalcGradedItems {
		return Violations{fmt.Sprintf(
			"Nikki Calc grades %d items no other source grades; the committed ceiling is %d, so another source may have stopped grading items it used to",
			stats.Graded, want.MaxCalcGradedItems)}
	}
	return nil
}

func ApplyCalcRecipes(acq map[int][]Acquisition, cat AcquisitionCatalogue, items map[int]CalcItem) int {
	bare := func(a Acquisition) bool { return a.Kind == "craft" && a.Text == "Crafting" && len(a.From) == 0 }
	given := func(a Acquisition) bool { return a.Kind == "craft" && len(a.From) > 0 }
	detailed := 0
	for _, id := range sortedIDs(acq) {
		list, recipe := acq[id], items[id].Recipe
		if len(recipe) == 0 || !slices.ContainsFunc(list, bare) || slices.ContainsFunc(list, given) {
			continue
		}
		named := make([]namedIngredient, 0, len(recipe))
		for _, in := range recipe {
			if name := cat.shown(in.ID); name != "" && !hasHan(name) {
				named = append(named, namedIngredient{name: name, id: in.ID, qty: in.Qty})
			}
		}
		if len(named) != len(recipe) {
			continue
		}
		craft := Acquisition{Kind: "craft", Text: "Craft: " + ingredientList(named), From: ingredients(named)}
		out := make([]Acquisition, 0, len(list))
		for _, a := range list {
			if bare(a) {
				a = craft
			}
			out = appendAcquisition(out, a)
		}
		acq[id] = out
		detailed++
	}
	return detailed
}
