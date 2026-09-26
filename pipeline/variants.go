package pipeline

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/danielradosa/nikkibase/core/scoring"
)

type StageVariant struct {
	Stage      string        `json:"stage"`
	Difficulty string        `json:"difficulty"`
	Weights    *[5]float64   `json:"weights,omitempty"`
	Bonus      []VariantCall `json:"bonus"`
	KeepAwards bool          `json:"keepAwards,omitempty"`
	Basis      string        `json:"basis"`
	Quote      string        `json:"quote,omitempty"`
}

type VariantCall struct {
	Grade      string  `json:"grade"`
	Multiplier float64 `json:"multiplier"`
	Tag        string  `json:"tag"`
}

const Maiden = "maiden"

func ReadStageVariants(b []byte) ([]StageVariant, error) {
	var f struct {
		Note     string         `json:"note"`
		Variants []StageVariant `json:"variants"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("stage difficulty: %w", err)
	}
	seen := map[string]bool{}
	for _, v := range f.Variants {
		switch {
		case v.Stage == "" || v.Basis == "":
			return nil, fmt.Errorf("stage difficulty: %q must name a stage and give a basis", v.Stage)
		case v.Difficulty != Maiden:
			return nil, fmt.Errorf("stage difficulty: %s gives %q; a stage's own numbers are Princess's, so a variant is %q", v.Stage, v.Difficulty, Maiden)
		case (v.Bonus != nil) == v.KeepAwards:
			return nil, fmt.Errorf("stage difficulty: %s must either give its tag calls or keep the stage's awards", v.Stage)
		case v.Weights != nil && *v.Weights == [5]float64{}:
			return nil, fmt.Errorf("stage difficulty: %s has no weights", v.Stage)
		case seen[v.Stage]:
			return nil, fmt.Errorf("stage difficulty: %s is listed twice", v.Stage)
		}
		seen[v.Stage] = true
		for _, c := range v.Bonus {
			if _, ok := StyleFromStageTag(c.Tag); !ok {
				return nil, fmt.Errorf("stage difficulty: %s names tag %q, which is no known style", v.Stage, c.Tag)
			}
			if _, ok := gradeBase[strings.ToUpper(c.Grade)]; !ok && !strings.EqualFold(c.Grade, factorGrade) {
				return nil, fmt.Errorf("stage difficulty: %s names grade %q", v.Stage, c.Grade)
			}
		}
	}
	return f.Variants, nil
}

var difficultyHint = regexp.MustCompile(`(?m)^\s*['"]([^'"]+)['"]\s*:.*(少女级tag|少女级权重|公主级tag)`)

func ApplyVariants(stages []Stage, variants []StageVariant, src []byte, stats *StageStats) error {
	src = uncomment(src)
	index := map[string]int{}
	for i, s := range stages {
		if s.Mode == "Story" {
			index[s.Name] = i
		}
	}
	have := map[string]bool{}
	for _, v := range variants {
		have[v.Stage] = true
	}
	var unmodelled []string
	for _, m := range difficultyHint.FindAllSubmatch(table(src, "addHintInfo"), -1) {
		if _, carried := index[string(m[1])]; carried && !have[string(m[1])] {
			unmodelled = append(unmodelled, string(m[1]))
		}
	}
	if len(unmodelled) > 0 {
		return fmt.Errorf("stage difficulty: the stage source gives one difficulty's numbers for %s, and no variant models them",
			strings.Join(unmodelled, ", "))
	}
	for _, v := range variants {
		at, ok := index[v.Stage]
		if !ok {
			return fmt.Errorf("stage difficulty: %s is not a story stage in the bundle", v.Stage)
		}
		if v.Quote != "" && !strings.Contains(string(src), v.Quote) {
			return fmt.Errorf("stage difficulty: the stage source no longer says %q for %s", v.Quote, v.Stage)
		}
		s := &stages[at]
		st := scoring.Stage{Attrs: s.Stage.Attrs, Weights: s.Stage.Weights}
		if v.Weights != nil {
			for i, w := range v.Weights {
				p := stagePairs[i]
				side := int8(p * 2)
				if w > 0 {
					side++
				}
				if side != s.Stage.Attrs[p] {
					return fmt.Errorf("stage difficulty: %s puts weight %d on the other side from the stage", v.Stage, i+1)
				}
				st.Weights[p] = math.Round(math.Abs(w) * 15)
			}
		}
		if v.KeepAwards {
			st.Tags = maps.Clone(s.Stage.Tags)
		} else {
			calls := make([][][]byte, len(v.Bonus))
			for i, c := range v.Bonus {
				calls[i] = [][]byte{nil, []byte(c.Grade), []byte(strconv.FormatFloat(c.Multiplier, 'g', -1, 64)), []byte(c.Tag)}
			}
			var priced StageStats
			pay(&st, calls, weightSum(Stage{Stage: st}), 1, &priced)
			if priced.UnknownTag+priced.UnknownGrade > 0 {
				return fmt.Errorf("stage difficulty: %s has a call that cannot be priced", v.Stage)
			}
		}
		if s.Variants == nil {
			s.Variants = map[string]scoring.Stage{}
		}
		s.Variants[v.Difficulty] = st
		stats.Variants++
	}
	return nil
}

func variantNames(s Stage) []string {
	names := slices.Collect(maps.Keys(s.Variants))
	slices.Sort(names)
	return names
}
