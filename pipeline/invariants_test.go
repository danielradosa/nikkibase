package pipeline

import (
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

func TestStatsMustComeFromGrades(t *testing.T) {
	good := graded(10001, scoring.Hair, "Right", "hair", [5]string{"S", "A", "B", "C", "SS"})
	if v := checkStats([]Entry{good}); v != nil {
		t.Fatalf("a correct item failed: %v", v)
	}
	bad := good
	bad.Item.ID, bad.Name = 10002, "Wrong"
	bad.Item.Stats[2] = Stat("B", scoring.Top)
	v := checkStats([]Entry{good, bad})
	if len(v) != 1 || !strings.Contains(v[0], "10002 (Wrong)") {
		t.Errorf("violations = %v, want one naming 10002", v)
	}
	blank := graded(10003, scoring.Hair, "Blank", "hair", [5]string{})
	blank.Item.Stats[0] = 1
	if checkStats([]Entry{blank}) == nil {
		t.Error("a stat with no grade behind it passed")
	}
}

func TestEveryItemRowMustCarryARarity(t *testing.T) {
	entries := []Entry{
		{Item: scoring.Item{ID: 10001, Slot: scoring.Hair}, Name: "Rated", Rarity: 3},
		{Item: scoring.Item{ID: 880012, Slot: scoring.Spirit}, Name: "Spirit", Rarity: 6},
	}
	calc := map[int]string{10001: "Rated", 30003: "Named Only"}
	if v := CheckRarity(entries, calc, map[int]int{30003: 1}); v != nil {
		t.Fatalf("rated rows failed: %v", v)
	}
	for _, c := range []struct {
		name    string
		entries []Entry
		rarity  map[int]int
		want    string
	}{
		{"blank item", append(entries, Entry{Item: scoring.Item{ID: 20002, Slot: scoring.Dress}, Name: "Blank"}),
			map[int]int{30003: 1}, "20002 (Blank)"},
		{"out of range", append(entries, Entry{Item: scoring.Item{ID: 20002, Slot: scoring.Dress}, Name: "Seven", Rarity: 7}),
			map[int]int{30003: 1}, "20002 (Seven)"},
		{"blank name-only row", entries, map[int]int{}, "30003 (Named Only)"},
		{"out of range name-only row", entries, map[int]int{30003: 7}, "30003 (Named Only)"},
	} {
		v := CheckRarity(c.entries, calc, c.rarity)
		if len(v) != 1 || !strings.Contains(v[0], c.want) {
			t.Errorf("%s: violations = %v, want one naming %s", c.name, v, c.want)
		}
	}
}

func weighted(name string, sum float64, awards ...int) Stage {
	s := Stage{Name: name, Mode: "Story"}
	s.Stage.Weights[0] = sum
	if len(awards) > 0 {
		s.Stage.Tags = map[int]int{}
		for i, a := range awards {
			s.Stage.Tags[i+1] = a
		}
	}
	return s
}

func ordinary() []Stage {
	var out []Stage
	for _, n := range []string{"1-1", "1-2", "1-3", "1-4", "1-5"} {
		out = append(out, weighted(n, 135, 1000))
	}
	return out
}

func TestZeroWeightStageFails(t *testing.T) {
	v := checkOutliers(append(ordinary(), weighted("2-1", 0)), nil)
	if len(v) != 1 || !strings.Contains(v[0], "2-1") {
		t.Errorf("violations = %v, want one naming 2-1", v)
	}
}

func TestAcknowledgementCoversOneCheckAtOneValue(t *testing.T) {
	heavy := func(sum float64, award int) []Stage { return append(ordinary(), weighted("15-9", sum, award)) }
	weights := map[string]Acknowledged{"15-9": {Stage: "15-9", Covers: CoversWeights, Value: 1515}}

	if v := checkOutliers(heavy(1515, 1000), weights); v != nil {
		t.Errorf("an acknowledged vector failed at its own value: %v", v)
	}
	if v := checkOutliers(heavy(1530, 1000), weights); v != nil {
		t.Errorf("a move inside 1%% failed: %v", v)
	}
	if v := checkOutliers(heavy(1540, 1000), weights); len(v) != 1 || !strings.Contains(v[0], "acknowledged") {
		t.Errorf("a move past 1%% gave %v, want one naming the acknowledgement", v)
	}
	if v := checkOutliers(heavy(1515, 11000), weights); v != nil {
		t.Errorf("an award in proportion to the acknowledged weights failed: %v", v)
	}
	if v := checkOutliers(heavy(1515, 1500000), weights); len(v) != 1 || !strings.Contains(v[0], "pays 1500000") {
		t.Errorf("an outlying award on a weights-acknowledged stage gave %v", v)
	}

	awards := map[string]Acknowledged{"II-4-7": {Stage: "II-4-7", Covers: CoversAwards, Value: 9000, Awards: 2}}
	stages := append(ordinary(), weighted("II-4-7", 135, 9000, 9000))
	if v := checkOutliers(stages, awards); v != nil {
		t.Errorf("two awards at the acknowledged value failed: %v", v)
	}
	stages[len(stages)-1] = weighted("II-4-7", 135, 9000)
	if v := checkOutliers(stages, awards); len(v) != 1 || !strings.Contains(v[0], "with 2 awards, and it pays 1") {
		t.Errorf("an award removed upstream gave %v, want it reported", v)
	}
	stages[len(stages)-1] = weighted("II-4-7", 135)
	if v := checkOutliers(stages, awards); len(v) != 1 || !strings.Contains(v[0], "it pays 0") {
		t.Errorf("every award removed upstream gave %v, want it reported", v)
	}
	stages[len(stages)-1] = weighted("II-4-7", 135, 9000, 900)
	if v := checkOutliers(stages, awards); len(v) != 1 || !strings.Contains(v[0], "pays 900 ") {
		t.Errorf("one award fixed upstream gave %v, want it reported", v)
	}
	stages[len(stages)-1] = weighted("II-4-7", 5000, 9000, 9000)
	if v := checkOutliers(stages, awards); len(v) != 1 || !strings.Contains(v[0], "summing to 5000") {
		t.Errorf("an outlying vector on an awards-acknowledged stage gave %v", v)
	}
}

func TestAwardCeilingIsPerUnitOfWeight(t *testing.T) {
	if v := checkOutliers(append(ordinary(), weighted("2-1", 135, 135*639)), nil); v != nil {
		t.Errorf("an award under the ceiling failed: %v", v)
	}
	if v := checkOutliers(append(ordinary(), weighted("2-1", 135, 135*700)), nil); len(v) != 1 || !strings.Contains(v[0], "2-1") {
		t.Errorf("an award over the ceiling gave %v, want it reported", v)
	}
	if v := checkOutliers(append(ordinary(), weighted("2-1", 270, 135*700)), nil); v != nil {
		t.Errorf("the same award on twice the weight failed: %v", v)
	}
}

func TestTagStageAcknowledgement(t *testing.T) {
	tag := map[string]Acknowledged{"2-7": {Stage: "2-7", Covers: CoversTagStage, Value: 9, Awards: 2}}
	stage := func(sum float64, awards ...int) []Stage {
		s := weighted("2-7", sum, awards...)
		s.Variants = map[string]scoring.Stage{Maiden: {Weights: [5]float64{sum / 2}, Tags: map[int]int{1: 30000}}}
		return append(ordinary(), s)
	}
	if v := checkOutliers(stage(9, 20000, 5000), tag); v != nil {
		t.Errorf("an acknowledged tag stage failed: %v", v)
	}
	if v := checkOutliers(stage(9, 20000, 5000), nil); len(v) != 4 {
		t.Errorf("an unacknowledged tag stage gave %d violations, want 4: %v", len(v), v)
	}
	if v := checkOutliers(stage(12, 20000, 5000), tag); len(v) != 1 || !strings.Contains(v[0], "now sum to 12") {
		t.Errorf("moved weights gave %v", v)
	}
	if v := checkOutliers(stage(9, 20000), tag); len(v) != 1 || !strings.Contains(v[0], "it pays 1") {
		t.Errorf("a lost award gave %v", v)
	}
}

func TestVariantsAreChecked(t *testing.T) {
	s := weighted("6-10", 135, 1000)
	s.Variants = map[string]scoring.Stage{Maiden: {Weights: [5]float64{10}, Tags: map[int]int{1: 10000}}}
	v := checkOutliers(append(ordinary(), s), nil)
	if len(v) != 2 || !strings.Contains(v[0], "6-10 (Maiden)") || !strings.Contains(v[1], "6-10 (Maiden)") {
		t.Errorf("violations = %v, want the Maiden weights and award reported", v)
	}
}

func TestDisplayNamesMustBeEnglish(t *testing.T) {
	stages := []Stage{{Name: "1-1", Mode: "Story"}, {Name: "海边派对的搭配", Mode: "Arena"},
		{Name: "海边派对的搭配", Mode: "Arena", Display: "Beach Party"}}
	if v := checkDisplayNames(stages); len(v) != 1 || !strings.Contains(v[0], "海边派对的搭配") {
		t.Errorf("violations = %v, want the one Chinese name reported", v)
	}
}

func TestAcknowledgedStageMustExist(t *testing.T) {
	gone := map[string]Acknowledged{"15-9": {Stage: "15-9", Covers: CoversWeights, Value: 1515}}
	if v := checkOutliers(ordinary(), gone); len(v) != 1 || !strings.Contains(v[0], "no such stage") {
		t.Errorf("violations = %v, want the missing stage reported", v)
	}
}

func TestReadAcknowledgedNeedsCoverAndValue(t *testing.T) {
	for name, doc := range map[string]string{
		"no cover":      `{"acknowledged": [{"stage": "a", "reason": "r", "effect": "e", "value": 1}]}`,
		"unknown cover": `{"acknowledged": [{"stage": "a", "reason": "r", "effect": "e", "covers": "stage", "value": 1}]}`,
		"no value":      `{"acknowledged": [{"stage": "a", "reason": "r", "effect": "e", "covers": "weights"}]}`,
		"no count":      `{"acknowledged": [{"stage": "a", "reason": "r", "effect": "e", "covers": "awards", "value": 1}]}`,
		"tag, no count": `{"acknowledged": [{"stage": "a", "reason": "r", "effect": "e", "covers": "tag", "value": 1}]}`,
		"twice": `{"acknowledged": [{"stage": "a", "reason": "r", "effect": "e", "covers": "weights", "value": 1},
			{"stage": "a", "reason": "r", "effect": "e", "covers": "awards", "value": 1, "awards": 1}]}`,
	} {
		if _, err := ReadAcknowledged([]byte(doc)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	got, err := ReadAcknowledged([]byte(`{"acknowledged": [{"stage": "a", "reason": "r", "effect": "e", "covers": "awards", "value": 2, "awards": 2}]}`))
	if err != nil || got["a"].Covers != CoversAwards || got["a"].Value != 2 || got["a"].Awards != 2 {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestReadStageCorrectionsRefusesZeroWeights(t *testing.T) {
	_, err := ReadStageCorrections([]byte(`{"corrections": [{"stage": "a", "basis": "b", "bonusDivisor": 1}]}`))
	if err == nil {
		t.Error("a correction with no weights was accepted")
	}
}
