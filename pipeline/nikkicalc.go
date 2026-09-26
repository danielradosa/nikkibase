package pipeline

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/danielradosa/nikkibase/core/scoring"
)

var nikkicalcSlot = map[byte]scoring.Slot{
	'H': scoring.Hair, 'D': scoring.Dress, 'C': scoring.Coat, 'T': scoring.Top,
	'B': scoring.Bottom, 'P': scoring.Hosiery, 'S': scoring.Shoes,
	'A': scoring.Accessory, 'M': scoring.Makeup,
}

var nikkicalcKey = regexp.MustCompile(`^([A-Z])(\d+)$`)

const (
	nikkicalcFields = 5
	nikkicalcRarity = 3
)

func NikkicalcID(key string) (int, bool) {
	m := nikkicalcKey.FindStringSubmatch(key)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[2])
	if err != nil {
		return 0, false
	}
	switch m[1][0] {
	case 'Z':
		return 170000 + n, true
	case 'X':
		return 880000 + n, true
	}
	slot, ok := nikkicalcSlot[m[1][0]]
	if !ok {
		return 0, false
	}
	return idPrefix[slot]*10000 + n, true
}

func ParseNikkicalc(items, ids []byte) (map[int]string, map[int]int, error) {
	var keys map[string]int
	if err := json.Unmarshal(ids, &keys); err != nil {
		return nil, nil, fmt.Errorf("pipeline: reading item keys: %w", err)
	}
	var flat []json.RawMessage
	if err := json.Unmarshal(items, &flat); err != nil {
		return nil, nil, fmt.Errorf("pipeline: reading item names: %w", err)
	}
	if len(flat)%nikkicalcFields != 0 {
		return nil, nil, fmt.Errorf("pipeline: item array is %d values, not a multiple of %d",
			len(flat), nikkicalcFields)
	}
	type record struct {
		name   string
		rarity int
	}
	byIndex := make(map[int]record, len(flat)/nikkicalcFields)
	for i := 0; i < len(flat); i += nikkicalcFields {
		var index int
		if err := json.Unmarshal(flat[i+1], &index); err != nil {
			continue
		}
		var code int
		if err := json.Unmarshal(flat[i+nikkicalcRarity], &code); err != nil || rarityOfCode(code) == 0 {
			return nil, nil, fmt.Errorf("pipeline: item %s has rarity code %s, which names no rarity",
				flat[i], flat[i+nikkicalcRarity])
		}
		rec := record{rarity: rarityOfCode(code)}
		_ = json.Unmarshal(flat[i+nikkicalcFields-1], &rec.name)
		byIndex[index] = rec
	}
	names := make(map[int]string, len(keys))
	rarity := make(map[int]int, len(keys))
	for key, index := range keys {
		id, ok := NikkicalcID(key)
		if !ok {
			continue
		}
		rec, ok := byIndex[index]
		if !ok {
			continue
		}
		rarity[id] = rec.rarity
		if rec.name != "" {
			names[id] = rec.name
		}
	}
	return names, rarity, nil
}

func rarityOfCode(code int) int {
	switch {
	case code >= 1 && code <= 6:
		return code
	case code >= 7 && code <= 12:
		return code - 6
	}
	return 0
}

func FillRarity(entries []Entry, extra map[int]string, rarity map[int]int) int {
	listed := make(map[int]bool, len(entries))
	filled := 0
	for i := range entries {
		listed[entries[i].Item.ID] = true
		if r, ok := rarity[entries[i].Item.ID]; ok && entries[i].Rarity == 0 {
			entries[i].Rarity = r
			filled++
		}
	}
	for id := range extra {
		if _, ok := rarity[id]; ok && !listed[id] {
			filled++
		}
	}
	return filled
}

func SlotOfID(id int) scoring.Slot {
	switch {
	case id >= 880000:
		return scoring.Spirit
	case id >= 170000:
		return scoring.Accessory
	}
	for slot, prefix := range idPrefix {
		if id/10000 == prefix {
			return slot
		}
	}
	return scoring.Accessory
}
