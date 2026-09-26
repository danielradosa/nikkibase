package pipeline

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type StageScope struct {
	Note           string         `json:"note"`
	AsOf           string         `json:"asOf"`
	Story          []StoryScope   `json:"story"`
	CommissionActs [2]int         `json:"commissionActs"`
	Whole          []string       `json:"whole"`
	Exclude        []ScopeExclude `json:"exclude"`
}

type StoryScope struct {
	Volume   int      `json:"volume"`
	Chapters [2]int   `json:"chapters"`
	Stages   []string `json:"stages,omitempty"`
}

type ScopeExclude struct {
	Stage  string `json:"stage"`
	Reason string `json:"reason"`
}

var wholeModes = []string{"Arena", "Co-op", "Event", "Dreamweaver"}

func ReadStageScope(b []byte) (*StageScope, error) {
	var s StageScope
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("stage scope: %w", err)
	}
	for _, r := range s.Story {
		switch {
		case r.Volume < 1 || r.Volume > len(volumePrefix):
			return nil, fmt.Errorf("stage scope: volume %d does not exist", r.Volume)
		case r.Chapters[0] < 1 || r.Chapters[0] > r.Chapters[1]:
			return nil, fmt.Errorf("stage scope: volume %d chapters %v are not a range", r.Volume, r.Chapters)
		case len(r.Stages) > 0 && r.Chapters[0] != r.Chapters[1]:
			return nil, fmt.Errorf("stage scope: volume %d lists stages across chapters %v; list them for one chapter", r.Volume, r.Chapters)
		}
	}
	if a := s.CommissionActs; a[0] < 1 || a[0] > a[1] {
		return nil, fmt.Errorf("stage scope: commission acts %v are not a range", a)
	}
	for _, m := range s.Whole {
		if !slices.Contains(wholeModes, m) {
			return nil, fmt.Errorf("stage scope: %q is not a mode that can be kept whole", m)
		}
	}
	for _, x := range s.Exclude {
		if x.Stage == "" || x.Reason == "" {
			return nil, fmt.Errorf("stage scope: exclusion %q must name a stage and give a reason", x.Stage)
		}
	}
	return &s, nil
}

var volumePrefix = []string{"", "II-", "III-"}

var (
	storyKey      = regexp.MustCompile(`^(II-|III-)?(\d+)-(.+)$`)
	commissionKey = regexp.MustCompile(`^联盟委托:\s*(\d+)-\d+$`)
)

func ApplyScope(stages []Stage, scope *StageScope) ([]Stage, map[string]int, error) {
	excluded := map[string]bool{}
	for _, x := range scope.Exclude {
		excluded[x.Stage] = true
	}
	present := map[string]bool{}
	matched := make([]bool, len(scope.Story))
	acts := false
	var kept []Stage
	out := map[string]int{}
	for _, s := range stages {
		present[s.Name] = true
		in := false
		switch s.Mode {
		case "Story":
			m := storyKey.FindStringSubmatch(s.Name)
			if m == nil {
				return nil, nil, fmt.Errorf("stage scope: story stage %q has no volume and chapter the scope can read", s.Name)
			}
			volume := slices.Index(volumePrefix, m[1]) + 1
			chapter, _ := strconv.Atoi(m[2])
			for i, r := range scope.Story {
				if r.Volume == volume && chapter >= r.Chapters[0] && chapter <= r.Chapters[1] &&
					(len(r.Stages) == 0 || slices.Contains(r.Stages, m[3])) {
					in, matched[i] = true, true
				}
			}
		case "Commission":
			m := commissionKey.FindStringSubmatch(s.Name)
			if m == nil {
				return nil, nil, fmt.Errorf("stage scope: commission %q has no act the scope can read", s.Name)
			}
			act, _ := strconv.Atoi(m[1])
			if act >= scope.CommissionActs[0] && act <= scope.CommissionActs[1] {
				in, acts = true, true
			}
		default:
			in = slices.Contains(scope.Whole, s.Mode)
		}
		if in && !excluded[s.Name] {
			kept = append(kept, s)
		} else {
			out[s.Mode]++
		}
	}
	for i, r := range scope.Story {
		if !matched[i] {
			return nil, nil, fmt.Errorf("stage scope: volume %d chapters %d-%d match no stage in the source", r.Volume, r.Chapters[0], r.Chapters[1])
		}
		for _, st := range r.Stages {
			key := fmt.Sprintf("%s%d-%s", volumePrefix[r.Volume-1], r.Chapters[0], st)
			if !present[key] {
				return nil, nil, fmt.Errorf("stage scope: %s is listed and the source has no such stage", key)
			}
		}
	}
	if !acts {
		return nil, nil, fmt.Errorf("stage scope: commission acts %d-%d match no stage in the source", scope.CommissionActs[0], scope.CommissionActs[1])
	}
	for _, x := range scope.Exclude {
		if !present[x.Stage] {
			return nil, nil, fmt.Errorf("stage scope: excluded stage %s is not in the source", strings.TrimSpace(x.Stage))
		}
	}
	return kept, out, nil
}
