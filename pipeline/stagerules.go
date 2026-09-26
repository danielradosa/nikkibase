package pipeline

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/danielradosa/nikkibase/core/scoring"
)

const (
	SourceWikiQuest = "wiki quest field"
	SourceStageNote = "stage source note"
)

type StageRule struct {
	Stage      string  `json:"stage"`
	Mode       string  `json:"mode"`
	Difficulty string  `json:"difficulty,omitempty"`
	Require    [][]int `json:"require"`
	Source     string  `json:"source"`
	Quote      string  `json:"quote"`
	Basis      string  `json:"basis,omitempty"`
}

func (r StageRule) label() string {
	if r.Difficulty != "" {
		return r.Mode + " " + r.Stage + " (" + r.Difficulty + ")"
	}
	return r.Mode + " " + r.Stage
}

func ReadStageRules(b []byte) ([]StageRule, error) {
	var f struct {
		Note  string      `json:"note"`
		Rules []StageRule `json:"rules"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("stage rules: %w", err)
	}
	seen := map[string]bool{}
	for _, r := range f.Rules {
		switch {
		case r.Stage == "" || r.Mode == "":
			return nil, fmt.Errorf("stage rules: an entry must name a stage and its mode")
		case r.Difficulty != "" && r.Difficulty != Maiden:
			return nil, fmt.Errorf("stage rules: %s gives %q; a rule holds at every difficulty unless it names %q", r.label(), r.Difficulty, Maiden)
		case r.Source != SourceWikiQuest && r.Source != SourceStageNote:
			return nil, fmt.Errorf("stage rules: %s gives source %q; it must be %q or %q", r.label(), r.Source, SourceWikiQuest, SourceStageNote)
		case r.Quote == "":
			return nil, fmt.Errorf("stage rules: %s quotes nothing from its source", r.label())
		case len(r.Require) == 0:
			return nil, fmt.Errorf("stage rules: %s requires nothing", r.label())
		case seen[r.label()]:
			return nil, fmt.Errorf("stage rules: %s is listed twice", r.label())
		}
		seen[r.label()] = true
		for _, set := range r.Require {
			if len(set) == 0 || len(slices.Compact(slices.Sorted(slices.Values(set)))) != len(set) {
				return nil, fmt.Errorf("stage rules: %s has a set that is empty or names an item twice: %v", r.label(), set)
			}
		}
	}
	return f.Rules, nil
}

func ApplyStageRules(stages []Stage, rules []StageRule, entries []Entry, src []byte, stats *StageStats) error {
	src = uncomment(src)
	index := make(map[string]int, len(stages))
	for i, s := range stages {
		index[s.Mode+"|"+s.Name] = i
	}
	places := make(map[int]SubSlot, len(entries))
	for _, e := range entries {
		if place, err := ResolvePosition(e.Position, e.Item.Slot); err == nil {
			places[e.Item.ID] = place
		}
	}
	for _, r := range rules {
		if _, ok := index[r.Mode+"|"+r.Stage]; !ok {
			return fmt.Errorf("stage rules: %s is not a stage in the bundle", r.label())
		}
		if r.Source == SourceStageNote && !strings.Contains(string(src), r.Quote) {
			return fmt.Errorf("stage rules: the stage source no longer says %q for %s", r.Quote, r.label())
		}
		for _, set := range r.Require {
			for _, id := range set {
				place, ok := places[id]
				if !ok {
					return fmt.Errorf("stage rules: %s requires item %d, which is not in the catalogue", r.label(), id)
				}
				if first := places[set[0]]; slotFamily(place.Slot) != slotFamily(first.Slot) {
					return fmt.Errorf("stage rules: %s offers %d (%s) and %d (%s) as one choice",
						r.label(), set[0], SlotName(first.Slot), id, SlotName(place.Slot))
				}
			}
		}
	}
	for _, r := range rules {
		if r.Difficulty != "" {
			continue
		}
		s := &stages[index[r.Mode+"|"+r.Stage]]
		if s.Rules == nil {
			s.Rules = &StageRules{}
		}
		s.Rules.Require = append(s.Rules.Require, r.Require...)
	}
	for _, r := range rules {
		if r.Difficulty == "" {
			continue
		}
		s := &stages[index[r.Mode+"|"+r.Stage]]
		if _, ok := s.Variants[r.Difficulty]; !ok {
			return fmt.Errorf("stage rules: %s holds at one difficulty, and the stage has no %s numbers of its own to carry it", r.label(), r.Difficulty)
		}
		var v StageRules
		if s.Rules != nil {
			v.Styles, v.Require = s.Rules.Styles, slices.Clone(s.Rules.Require)
		}
		v.Require = append(v.Require, r.Require...)
		if s.VariantRules == nil {
			s.VariantRules = map[string]StageRules{}
		}
		s.VariantRules[r.Difficulty] = v
	}
	stats.Required = 0
	for _, s := range stages {
		versions := make([][][]int, 0, 1+len(s.VariantRules))
		if s.Rules != nil && len(s.Rules.Require) > 0 {
			versions = append(versions, s.Rules.Require)
		}
		for _, name := range slices.Sorted(maps.Keys(s.VariantRules)) {
			versions = append(versions, s.VariantRules[name].Require)
		}
		if len(versions) == 0 {
			continue
		}
		stats.Required++
		for _, sets := range versions {
			for i := range sets {
				for _, other := range sets[i+1:] {
					if !wearableTogether(sets[i], other, places) {
						return fmt.Errorf("stage rules: %s %s requires %v and %v, which can never be worn together",
							s.Mode, s.Name, sets[i], other)
					}
				}
			}
		}
	}
	return nil
}

func slotFamily(s scoring.Slot) scoring.Slot {
	if s == scoring.Top || s == scoring.Bottom {
		return scoring.Dress
	}
	return s
}

func wearableTogether(a, b []int, places map[int]SubSlot) bool {
	for _, x := range a {
		for _, y := range b {
			p, q := places[x], places[y]
			clash := p.Name == q.Name || (p.Group != "" && p.Group == q.Group && (p.Exclusive() || q.Exclusive()))
			if x == y || !clash {
				return true
			}
		}
	}
	return false
}
