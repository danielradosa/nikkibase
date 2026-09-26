package pipeline

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/danielradosa/nikkibase/core/scoring"
)

var wearablePlaces = map[scoring.Slot]int{
	scoring.Hair: 1, scoring.Dress: 1, scoring.Coat: 1, scoring.Top: 1,
	scoring.Bottom: 1, scoring.Hosiery: 2, scoring.Shoes: 1, scoring.Makeup: 1,
	scoring.Accessory: 24, scoring.Spirit: 1,
}

type Coverage struct {
	Items                int            `json:"items"`
	StagesByMode         map[string]int `json:"stagesByMode"`
	TaggedStages         int            `json:"taggedStages"`
	MaxOrphanBonus       int            `json:"maxOrphanBonus"`
	MaxUnwornTags        int            `json:"maxUnwornTags"`
	ValuedStages         int            `json:"valuedStages"`
	SubgradedCells       int            `json:"subgradedCells"`
	MaxSubgradeFallbacks int            `json:"maxSubgradeFallbacks"`
	AcquisitionItems     int            `json:"acquisitionItems"`
	MaxUnmatchedNames    int            `json:"maxUnmatchedNames"`
	SuitItems            int            `json:"suitItems"`
}

type Violations []string

func (v Violations) Error() string {
	return fmt.Sprintf("the bundle does not describe a game the player could be playing:\n  %s",
		strings.Join(v, "\n  "))
}

func CheckInvariants(entries []Entry, stages []Stage, stats StageStats, want Coverage,
	acknowledged map[string]Acknowledged, sub *SubgradeStats) error {
	var v Violations
	v = append(v, checkPositions(entries)...)
	v = append(v, checkIdentity(entries)...)
	v = append(v, checkStats(entries)...)
	v = append(v, checkTags(entries, stages, stats, want)...)
	v = append(v, checkCoverage(entries, stages, want)...)
	v = append(v, checkSubgrades(entries, sub, want)...)
	v = append(v, checkOutliers(stages, acknowledged)...)
	v = append(v, checkDisplayNames(stages)...)
	if len(v) == 0 {
		return nil
	}
	return v
}

func checkPositions(entries []Entry) Violations {
	places := map[scoring.Slot]map[string]int{}
	for _, e := range entries {
		slot := e.Item.Slot
		if places[slot] == nil {
			places[slot] = map[string]int{}
		}
		places[slot][e.Position]++
	}
	var v Violations
	for _, slot := range sortedSlots(places) {
		want, known := wearablePlaces[slot]
		got := len(places[slot])
		switch {
		case !known:
			v = append(v, fmt.Sprintf("slot %d is not a slot the game has", slot))
		case got > want:
			v = append(v, fmt.Sprintf("%s holds %d wearable places, and the game has %d: %s",
				SlotName(slot), got, want, namesOf(places[slot])))
		}
	}
	return v
}

func checkIdentity(entries []Entry) Violations {
	seen := make(map[int]string, len(entries))
	var dup, mis []string
	for _, e := range entries {
		id := e.Item.ID
		if first, ok := seen[id]; ok {
			dup = append(dup, fmt.Sprintf("%d (%s, %s)", id, first, e.Name))
		} else {
			seen[id] = e.Name
		}
		if want := SlotOfID(id); e.Item.Slot != want {
			mis = append(mis, fmt.Sprintf("%d (%s) is scored as %s, and its ID says %s",
				id, e.Name, SlotName(e.Item.Slot), SlotName(want)))
		}
	}
	var v Violations
	if len(dup) > 0 {
		sort.Strings(dup)
		v = append(v, fmt.Sprintf("%d item IDs each name two garments: %s", len(dup), truncate(dup)))
	}
	if len(mis) > 0 {
		sort.Strings(mis)
		v = append(v, fmt.Sprintf("%d items are scored in a slot their ID contradicts: %s", len(mis), truncate(mis)))
	}
	return v
}

func checkStats(entries []Entry) Violations {
	var bad []string
	for _, e := range entries {
		for p, got := range e.Item.Stats {
			grade, want := e.Grades[p], Stat(e.Grades[p], e.Item.Slot)
			if sub := e.Subgrades[p]; sub != "" {
				if SubgradeLetter(sub) != strings.ToUpper(grade) {
					bad = append(bad, fmt.Sprintf("%d (%s) holds sub-grade %q on pair %d, outside its grade %q",
						e.Item.ID, e.Name, sub, p, grade))
					break
				}
				grade, want = sub, SubStat(sub, e.Item.Slot)
			}
			if got != want {
				bad = append(bad, fmt.Sprintf("%d (%s) holds %d on pair %d, and grade %q on a %s is %d",
					e.Item.ID, e.Name, got, p, grade, SlotName(e.Item.Slot), want))
				break
			}
		}
	}
	if len(bad) == 0 {
		return nil
	}
	sort.Strings(bad)
	return Violations{fmt.Sprintf("%d items carry stats their grades and slot do not give: %s",
		len(bad), truncate(bad))}
}

func checkTags(entries []Entry, stages []Stage, stats StageStats, want Coverage) Violations {
	carried := map[int]bool{}
	for _, e := range entries {
		for _, t := range e.Item.Tags {
			carried[t] = true
		}
	}
	unworn := map[int]bool{}
	for _, s := range stages {
		for id := range s.Stage.Tags {
			if !carried[id] {
				unworn[id] = true
			}
		}
	}
	var v Violations
	if len(unworn) > want.MaxUnwornTags {
		names := make([]string, 0, len(unworn))
		for id := range unworn {
			names = append(names, TagName(id))
		}
		sort.Strings(names)
		v = append(v, fmt.Sprintf("%d stage tags are carried by no item, so they pay nothing, and the committed ceiling is %d: %s",
			len(unworn), want.MaxUnwornTags, strings.Join(names, ", ")))
	}
	if stats.UnknownTag > 0 {
		v = append(v, fmt.Sprintf("%d stage bonus tags matched no known style", stats.UnknownTag))
	}
	if stats.UnknownGrade > 0 {
		v = append(v, fmt.Sprintf("%d stage bonus grades are absent from the calibration table", stats.UnknownGrade))
	}
	if stats.OrphanBonus > want.MaxOrphanBonus {
		v = append(v, fmt.Sprintf("%d tag bonuses name a stage that was never parsed, and the committed ceiling is %d",
			stats.OrphanBonus, want.MaxOrphanBonus))
	}
	if stats.SideConflicts > 0 {
		v = append(v, fmt.Sprintf("%d stages put a weight on the other side in the second stage source, which cannot be settled by precision",
			stats.SideConflicts))
	}
	if want.ValuedStages > 0 && stats.Valued < want.ValuedStages {
		v = append(v, fmt.Sprintf("%d stages took their weights and awards from the second stage source, and the committed floor is %d",
			stats.Valued, want.ValuedStages))
	}
	if stats.Dropped > 0 {
		v = append(v, fmt.Sprintf("%d of the source's %d addBonusInfo calls were never parsed",
			stats.Dropped, stats.BonusCalls))
	}
	return v
}

func checkCoverage(entries []Entry, stages []Stage, want Coverage) Violations {
	var v Violations
	if want.Items > 0 && len(entries) < want.Items {
		v = append(v, fmt.Sprintf("the catalogue holds %d items, and the committed floor is %d",
			len(entries), want.Items))
	}
	byMode := map[string]int{}
	tagged := 0
	for _, s := range stages {
		byMode[s.Mode]++
		if len(s.Stage.Tags) > 0 {
			tagged++
		}
	}
	for _, mode := range sortedKeys(want.StagesByMode) {
		if byMode[mode] < want.StagesByMode[mode] {
			v = append(v, fmt.Sprintf("%s holds %d stages, and the committed floor is %d",
				mode, byMode[mode], want.StagesByMode[mode]))
		}
	}
	if want.TaggedStages > 0 && tagged < want.TaggedStages {
		v = append(v, fmt.Sprintf("%d stages carry a tag award, and the committed floor is %d",
			tagged, want.TaggedStages))
	}
	return v
}

func checkSubgrades(entries []Entry, sub *SubgradeStats, want Coverage) Violations {
	if sub == nil {
		return nil
	}
	cells := 0
	for _, e := range entries {
		for _, g := range e.Subgrades {
			if g != "" {
				cells++
			}
		}
	}
	var v Violations
	if cells < want.SubgradedCells {
		v = append(v, fmt.Sprintf("%d item stats come from a sub-grade, and the committed floor is %d",
			cells, want.SubgradedCells))
	}
	if fallback := sub.OtherSide + sub.OtherLetter + sub.Missing; fallback > want.MaxSubgradeFallbacks {
		v = append(v, fmt.Sprintf("%d graded item stats found no sub-grade to take (%d on the other side, %d under another letter, %d with none), and the committed ceiling is %d",
			fallback, sub.OtherSide, sub.OtherLetter, sub.Missing, want.MaxSubgradeFallbacks))
	}
	return v
}

func truncate(items []string) string {
	const show = 8
	if len(items) <= show {
		return strings.Join(items, "; ")
	}
	return strings.Join(items[:show], "; ") + fmt.Sprintf("; and %d more", len(items)-show)
}

func namesOf(places map[string]int) string {
	names := make([]string, 0, len(places))
	for p := range places {
		names = append(names, p)
	}
	sort.Strings(names)
	return truncate(names)
}

func sortedSlots(m map[scoring.Slot]map[string]int) []scoring.Slot {
	out := make([]scoring.Slot, 0, len(m))
	for s := range m {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func SlotName(s scoring.Slot) string {
	names := [...]string{"hair", "dress", "coat", "top", "bottom", "hosiery",
		"shoes", "makeup", "accessory", "spirit"}
	if int(s) < len(names) {
		return names[s]
	}
	return fmt.Sprintf("slot %d", s)
}

const outlierFactor = 5

var awardCeiling = 3 * gradeBase["SSS"]

func checkOutliers(stages []Stage, acknowledged map[string]Acknowledged) Violations {
	var weights []float64
	for _, s := range stages {
		if sum := weightSum(s); sum > 0 {
			weights = append(weights, sum)
		}
	}
	wMid := median(weights)
	var v Violations
	present := make(map[string]bool, len(stages))
	for _, s := range stages {
		present[s.Name] = true
		ack, acked := acknowledged[s.Name]
		label := s.Mode + " " + s.Name
		sum := weightSum(s)
		if acked && (ack.Covers == CoversWeights || ack.Covers == CoversTagStage) {
			if moved(sum, ack.Value) {
				v = append(v, fmt.Sprintf("%s is acknowledged with weights summing to %.0f, and they now sum to %.0f",
					label, ack.Value, sum))
			}
		} else {
			v = append(v, checkWeightSum(label, sum, wMid)...)
			for _, d := range variantNames(s) {
				v = append(v, checkWeightSum(variantLabel(label, d), weightSum(Stage{Stage: s.Variants[d]}), wMid)...)
			}
		}
		switch {
		case acked && ack.Covers == CoversAwards:
			if n := len(s.Stage.Tags); n != ack.Awards {
				v = append(v, fmt.Sprintf("%s is acknowledged with %d awards, and it pays %d", label, ack.Awards, n))
			}
			for _, id := range sortedTags(s.Stage.Tags) {
				if a := s.Stage.Tags[id]; moved(float64(a), ack.Value) {
					v = append(v, fmt.Sprintf("%s is acknowledged with awards of %.0f, and it pays %d for %s",
						label, ack.Value, a, TagName(id)))
				}
			}
			continue
		case acked && ack.Covers == CoversTagStage:
			if n := len(s.Stage.Tags); n != ack.Awards {
				v = append(v, fmt.Sprintf("%s is acknowledged as a tag stage with %d awards, and it pays %d", label, ack.Awards, n))
			}
			continue
		}
		v = append(v, checkAwards(label, s.Stage)...)
		for _, d := range variantNames(s) {
			v = append(v, checkAwards(variantLabel(label, d), s.Variants[d])...)
		}
	}
	for _, name := range sortedAcknowledged(acknowledged) {
		if !present[name] {
			v = append(v, fmt.Sprintf("%s is acknowledged as an outlier, and the bundle has no such stage", name))
		}
	}
	return v
}

func checkWeightSum(label string, sum, mid float64) Violations {
	if mid > 0 && (sum > mid*outlierFactor || sum*outlierFactor < mid) {
		return Violations{fmt.Sprintf("%s has weights summing to %.0f, and the median stage sums to %.0f", label, sum, mid)}
	}
	return nil
}

func checkAwards(label string, st scoring.Stage) Violations {
	sum := weightSum(Stage{Stage: st})
	var v Violations
	for _, id := range sortedTags(st.Tags) {
		a := st.Tags[id]
		if sum > 0 && float64(a) > sum*awardCeiling {
			v = append(v, fmt.Sprintf("%s pays %d for %s, %.0f per unit of weight where no ordinary award pays over %.0f",
				label, a, TagName(id), float64(a)/sum, awardCeiling))
		}
	}
	return v
}

func variantLabel(label, difficulty string) string {
	return fmt.Sprintf("%s (%s)", label, strings.ToUpper(difficulty[:1])+difficulty[1:])
}

func checkDisplayNames(stages []Stage) Violations {
	var bad []string
	for _, s := range stages {
		if name := DisplayName(s); hasHan(name) {
			bad = append(bad, s.Mode+" "+name)
		}
	}
	if len(bad) == 0 {
		return nil
	}
	return Violations{fmt.Sprintf("%d stages have no English name: %s", len(bad), truncate(bad))}
}

func CheckRarity(entries []Entry, extra map[int]string, rarity map[int]int) Violations {
	listed := make(map[int]bool, len(entries))
	var bad []string
	for _, e := range entries {
		listed[e.Item.ID] = true
		if e.Rarity < 1 || e.Rarity > 6 {
			bad = append(bad, fmt.Sprintf("%d (%s) rarity %d", e.Item.ID, e.Name, e.Rarity))
		}
	}
	for id, name := range extra {
		if r := rarity[id]; !listed[id] && (r < 1 || r > 6) {
			bad = append(bad, fmt.Sprintf("%d (%s) rarity %d", id, name, r))
		}
	}
	if len(bad) == 0 {
		return nil
	}
	sort.Strings(bad)
	return Violations{fmt.Sprintf("%d item rows carry no rarity from 1 to 6: %s", len(bad), truncate(bad))}
}

func moved(got, want float64) bool {
	return math.Abs(got-want) > 0.01*want
}

func sortedTags(m map[int]int) []int {
	out := make([]int, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}

func sortedAcknowledged(m map[string]Acknowledged) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func weightSum(s Stage) float64 {
	var sum float64
	for _, w := range s.Stage.Weights {
		sum += w
	}
	return sum
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sorted := make([]float64, len(xs))
	copy(sorted, xs)
	sort.Float64s(sorted)
	return sorted[len(sorted)/2]
}
