package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"
)

type StageDisplayEntry struct {
	Mode  string `json:"mode"`
	Stage string `json:"stage"`
	Name  string `json:"name"`
	Basis string `json:"basis"`
}

func ReadStageDisplayNames(b []byte) ([]StageDisplayEntry, error) {
	var f struct {
		Names []StageDisplayEntry `json:"names"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("stage display names: %w", err)
	}
	stages, names := map[[2]string]bool{}, map[[2]string]bool{}
	for _, e := range f.Names {
		switch {
		case strings.TrimSpace(e.Mode) == "" || strings.TrimSpace(e.Stage) == "" ||
			strings.TrimSpace(e.Name) == "" || strings.TrimSpace(e.Basis) == "":
			return nil, fmt.Errorf("stage display names: %q must give a mode, a stage, a name and a basis", e.Stage)
		case stages[[2]string{e.Mode, e.Stage}]:
			return nil, fmt.Errorf("stage display names: %s %s is listed twice", e.Mode, e.Stage)
		case names[[2]string{e.Mode, e.Name}]:
			return nil, fmt.Errorf("stage display names: two %s stages are named %q", e.Mode, e.Name)
		}
		stages[[2]string{e.Mode, e.Stage}] = true
		names[[2]string{e.Mode, e.Name}] = true
	}
	return f.Names, nil
}

func ApplyStageDisplayNames(stages []Stage, names []StageDisplayEntry) error {
	byStage := map[[2]string]string{}
	named := map[string]bool{}
	for _, e := range names {
		byStage[[2]string{e.Mode, e.Stage}] = e.Name
		named[e.Mode] = true
	}
	used := map[[2]string]bool{}
	for i, s := range stages {
		if !named[s.Mode] {
			continue
		}
		key := [2]string{s.Mode, DisplayName(s)}
		name, ok := byStage[key]
		if !ok {
			return fmt.Errorf("stage display names: every %s stage needs a name, and %s has none", s.Mode, key[1])
		}
		stages[i].Display = name
		used[key] = true
	}
	for _, e := range names {
		if !used[[2]string{e.Mode, e.Stage}] {
			return fmt.Errorf("stage display names: %s %s matches no stage that ships", e.Mode, e.Stage)
		}
	}
	return nil
}
