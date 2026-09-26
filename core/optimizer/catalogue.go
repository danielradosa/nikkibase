package optimizer

import (
	"github.com/danielradosa/nikkibase/core/catalogue"
	"github.com/danielradosa/nikkibase/core/scoring"
)

func FromCatalogue(c *catalogue.Catalogue, owned func(id int32) bool) ([]Position, map[int]int, map[int]uint16) {
	kept := make([]int, 0, len(c.IDs))
	count := map[uint16]int{}
	groups := map[uint16]uint8{}
	var order []uint16
	tags := 0
	for i, id := range c.IDs {
		if owned != nil && !owned(id) {
			continue
		}
		kept = append(kept, i)
		place := c.Positions[i]
		if _, seen := count[place]; !seen {
			order = append(order, place)
		}
		count[place]++
		groups[place] = c.Groups[i]
		tags += int(c.TagOffset[i+1] - c.TagOffset[i])
	}

	items := make([]scoring.Item, len(kept))
	byPlace := make(map[uint16][]scoring.Item, len(order))
	start := 0
	for _, place := range order {
		byPlace[place] = items[start : start : start+count[place]]
		start += count[place]
	}
	tagPool := make([]int, 0, tags)
	for _, i := range kept {
		it := scoring.Item{ID: int(c.IDs[i]), Slot: scoring.Slot(c.Slots[i]), FlatBonus: int(c.FlatBonus[i])}
		for p := range 5 {
			it.Attrs[p] = c.Attrs[i*5+p]
			it.Stats[p] = int(c.Stats[i*5+p])
		}
		if from, to := c.TagOffset[i], c.TagOffset[i+1]; to > from {
			at := len(tagPool)
			for _, t := range c.Tags[from:to] {
				tagPool = append(tagPool, int(t))
			}
			it.Tags = tagPool[at:len(tagPool):len(tagPool)]
		}
		place := c.Positions[i]
		byPlace[place] = append(byPlace[place], it)
	}

	positions := make([]Position, 0, len(order))
	index := make(map[int]int, len(kept))
	placeOf := make(map[int]uint16, len(kept))
	for _, place := range order {
		for _, it := range byPlace[place] {
			index[it.ID] = len(positions)
			placeOf[it.ID] = place
		}
		group, whole := catalogue.SplitGroup(groups[place])
		positions = append(positions, Position{Items: byPlace[place], Group: group, Exclusive: whole})
	}
	return positions, index, placeOf
}
