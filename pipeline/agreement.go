package pipeline

import (
	"fmt"
	"slices"
	"strings"
)

type AgreementStats struct {
	Pairs, Sides int
	Items, Heavy []int
}

func CorrectByAgreement(entries []Entry, given map[int][5]string, packed map[int]Entry, calc map[int]CalcItem) AgreementStats {
	var stats AgreementStats
	for i := range entries {
		e := &entries[i]
		first, ok := given[e.Item.ID]
		if !ok {
			continue
		}
		table, inTable := packed[e.Item.ID]
		nc, inCalc := calc[e.Item.ID]
		if !inTable || !inCalc {
			continue
		}
		changed := 0
		for p := range 5 {
			letter, attr := strings.ToUpper(table.Grades[p]), table.Item.Attrs[p]
			if first[p] == "" || letter == "" || nc.Row[p].Attr != attr || SubgradeLetter(nc.Row[p].Grade) != letter {
				continue
			}
			if e.Item.Attrs[p] == attr && strings.ToUpper(e.Grades[p]) == letter {
				continue
			}
			if e.Item.Attrs[p] != attr {
				stats.Sides++
			}
			e.Item.Attrs[p], e.Grades[p] = attr, letter
			e.Item.Stats[p] = Stat(letter, e.Item.Slot)
			changed++
		}
		if changed == 0 {
			continue
		}
		stats.Pairs += changed
		stats.Items = append(stats.Items, e.Item.ID)
		if changed >= 3 {
			stats.Heavy = append(stats.Heavy, e.Item.ID)
		}
	}
	slices.Sort(stats.Items)
	slices.Sort(stats.Heavy)
	return stats
}

func TableRows(entries []Entry) map[int]Entry {
	return rowsByID(entries, func(a, b Entry) bool { return a.Grades == b.Grades && a.Item.Attrs == b.Item.Attrs })
}

func CheckAgreement(stats AgreementStats, want Coverage) Violations {
	if stats.Pairs > want.MaxAgreedGrades {
		return Violations{fmt.Sprintf(
			"the packed table and Nikki Calc agree against %d grades the item pages give; the committed ceiling is %d, so a source may have been misread",
			stats.Pairs, want.MaxAgreedGrades)}
	}
	return nil
}
