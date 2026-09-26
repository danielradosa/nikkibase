package pipeline

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/danielradosa/nikkibase/core/scoring"
)

var stageTables = map[string]string{
	"levelsRaw":       "Story",
	"tasksRaw":        "Commission",
	"competitionsRaw": "Arena",
	"extraRaw":        "Event",
	"dreamWeavingRaw": "Dreamweaver",
}

var stagePairs = [...]int{0, 2, 1, 3, 4}

var (
	tableOpen  = regexp.MustCompile(`^var\s+(\w+)\s*=\s*\{`)
	stageLine  = regexp.MustCompile(`^'([^']+)':\s*\[([-\d., ]+)\],?$`)
	bonusEntry = regexp.MustCompile(`(?s)['"]([^'"]+)['"]:\s*\[([^\]]*)\]`)
	bonusAny   = regexp.MustCompile(`addBonusInfo\(`)
	bonusCall  = regexp.MustCompile(`addBonusInfo\(['"]([^'"]*)['"],\s*([-\d.]+),\s*['"]([^'"]+)['"]\)`)
)

type StageStats struct {
	Stages                        int
	WithBonus                     int
	Awards                        int
	UnknownTag                    int
	UnknownGrade                  int
	OrphanBonus                   int
	Ruled, RuleEntries            int
	Corrected                     int
	Dropped                       int
	BonusCalls                    int
	Valued, SideConflicts, Beyond int
	Variants                      int
	Required                      int
}

type StageCorrection struct {
	Stage        string     `json:"stage"`
	Mode         string     `json:"mode"`
	Weights      [5]float64 `json:"weights"`
	Basis        string     `json:"basis"`
	BonusDivisor float64    `json:"bonusDivisor"`
	BonusBasis   string     `json:"bonusBasis"`
}

type Acknowledged struct {
	Stage  string  `json:"stage"`
	Mode   string  `json:"mode"`
	Covers string  `json:"covers"`
	Value  float64 `json:"value"`
	Awards int     `json:"awards,omitempty"`
	Reason string  `json:"reason"`
	Effect string  `json:"effect"`
}

const (
	CoversWeights  = "weights"
	CoversAwards   = "awards"
	CoversTagStage = "tag"
)

type StageCorrections struct {
	Note         string            `json:"note"`
	DerivedOn    string            `json:"derivedOn"`
	Corrections  []StageCorrection `json:"corrections"`
	Acknowledged []Acknowledged    `json:"acknowledged"`
}

func ReadAcknowledged(b []byte) (map[string]Acknowledged, error) {
	var c StageCorrections
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("stage corrections: %w", err)
	}
	out := make(map[string]Acknowledged, len(c.Acknowledged))
	for _, a := range c.Acknowledged {
		switch {
		case a.Stage == "" || a.Reason == "" || a.Effect == "":
			return nil, fmt.Errorf("stage corrections: acknowledged %q must give a reason and its effect", a.Stage)
		case a.Covers != CoversWeights && a.Covers != CoversAwards && a.Covers != CoversTagStage:
			return nil, fmt.Errorf("stage corrections: acknowledged %q covers %q; it must name %q, %q or %q",
				a.Stage, a.Covers, CoversWeights, CoversAwards, CoversTagStage)
		case a.Value <= 0:
			return nil, fmt.Errorf("stage corrections: acknowledged %q gives no value for the %s it covers", a.Stage, a.Covers)
		case (a.Covers == CoversAwards || a.Covers == CoversTagStage) && a.Awards <= 0:
			return nil, fmt.Errorf("stage corrections: acknowledged %q covers its awards and does not say how many it pays", a.Stage)
		}
		if _, dup := out[a.Stage]; dup {
			return nil, fmt.Errorf("stage corrections: acknowledged %q is listed twice", a.Stage)
		}
		out[a.Stage] = a
	}
	return out, nil
}

func ReadStageCorrections(b []byte) (map[string]StageCorrection, error) {
	var c StageCorrections
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("stage corrections: %w", err)
	}
	out := make(map[string]StageCorrection, len(c.Corrections))
	for _, r := range c.Corrections {
		switch {
		case r.Stage == "":
			return nil, fmt.Errorf("stage corrections: an entry names no stage")
		case r.Basis == "":
			return nil, fmt.Errorf("stage corrections: %q gives no basis for its weights", r.Stage)
		case r.Weights == [5]float64{}:
			return nil, fmt.Errorf("stage corrections: %q has no weights", r.Stage)
		case r.BonusDivisor <= 0:
			return nil, fmt.Errorf("stage corrections: %q has a bonus divisor of %v", r.Stage, r.BonusDivisor)
		case r.BonusDivisor != 1 && r.BonusBasis == "":
			return nil, fmt.Errorf("stage corrections: %q divides its tag multipliers and gives no basis", r.Stage)
		}
		if _, dup := out[r.Stage]; dup {
			return nil, fmt.Errorf("stage corrections: %q is listed twice", r.Stage)
		}
		out[r.Stage] = r
	}
	return out, nil
}

func ParseStages(src []byte) ([]Stage, StageStats, error) {
	return ParseStagesWith(src, nil)
}

func ParseStagesWith(src []byte, corrections map[string]StageCorrection) ([]Stage, StageStats, error) {
	src = uncomment(src)
	var stats StageStats
	stages, index, err := readStageTables(src, corrections, &stats)
	if err != nil {
		return nil, stats, err
	}
	stats.Stages = len(stages)
	applyBonuses(src, stages, index, corrections, &stats)
	applyRules(src, stages, index, &stats)
	return stages, stats, nil
}

func readStageTables(src []byte, corrections map[string]StageCorrection, stats *StageStats) ([]Stage, map[string]int, error) {
	var (
		stages []Stage
		table  string
		index  = map[string]int{}
	)
	sc := bufio.NewScanner(bytes.NewReader(src))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if m := tableOpen.FindStringSubmatch(line); m != nil {
			table = stageTables[m[1]]
			continue
		}
		if line == "};" || line == "}" {
			table = ""
			continue
		}
		if table == "" {
			continue
		}
		m := stageLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		parts := strings.Split(m[2], ",")
		if len(parts) != 5 {
			continue
		}
		var raw [5]float64
		var st scoring.Stage
		valid := true
		for i, part := range parts {
			v, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
			if err != nil {
				valid = false
				break
			}
			raw[i] = v
		}
		if !valid {
			continue
		}
		if fixed, ok := corrections[m[1]]; ok {
			raw = fixed.Weights
			stats.Corrected++
		}
		for i, v := range raw {
			p := stagePairs[i]
			st.Attrs[p] = int8(p * 2)
			if v > 0 {
				st.Attrs[p]++
			}
			st.Weights[p] = math.Round(math.Abs(v) * 15)
		}
		if _, seen := index[m[1]]; seen {
			continue
		}
		index[m[1]] = len(stages)
		mode := table
		if _, hint := SplitStageName(m[1]); hint != "" {
			mode = hint
		}
		stages = append(stages, Stage{Name: m[1], Mode: mode, Stage: st})
	}
	if err := sc.Err(); err != nil {
		return nil, nil, fmt.Errorf("pipeline: reading stages: %w", err)
	}
	return stages, index, nil
}

func applyBonuses(src []byte, stages []Stage, index map[string]int,
	corrections map[string]StageCorrection, stats *StageStats) {
	start := bytes.Index(src, []byte("var levelBonus"))
	if start < 0 {
		return
	}
	end := bytes.Index(src[start:], []byte("\n};"))
	if end < 0 {
		return
	}
	coop := coopIndex(stages, index)
	region := src[start : start+end]
	stats.BonusCalls = len(bonusAny.FindAll(region, -1))
	stats.Dropped = stats.BonusCalls
	for _, m := range bonusEntry.FindAllSubmatch(region, -1) {
		name := string(m[1])
		calls := bonusCall.FindAllSubmatch(m[2], -1)
		if len(calls) == 0 {
			continue
		}
		stats.Dropped -= len(calls)
		at, ok := index[name]
		if !ok {
			at, ok = coop[coopKey(name)]
		}
		if !ok {
			stats.OrphanBonus++
			continue
		}
		divisor := 1.0
		if fixed, ok := corrections[stages[at].Name]; ok {
			divisor = fixed.BonusDivisor
		}
		st := &stages[at].Stage
		pay(st, calls, weightSum(stages[at]), divisor, stats)
	}
}

const factorGrade = "F"

func pay(st *scoring.Stage, calls [][][]byte, weightSum, divisor float64, stats *StageStats) {
	for _, c := range calls {
		style, ok := StyleFromStageTag(string(c[3]))
		if !ok {
			stats.UnknownTag++
			continue
		}
		id, ok := TagID(style)
		if !ok {
			stats.UnknownTag++
			continue
		}
		base, ok := gradeBase[strings.ToUpper(string(c[1]))]
		if strings.EqualFold(string(c[1]), factorGrade) {
			base, ok = 1, true
		}
		if !ok {
			stats.UnknownGrade++
			continue
		}
		mult, err := strconv.ParseFloat(string(c[2]), 64)
		if err != nil {
			stats.UnknownGrade++
			continue
		}
		mult /= divisor
		if st.Tags == nil {
			st.Tags = map[int]int{}
			stats.WithBonus++
		}
		st.Tags[id] = int(math.Round(weightSum * base * mult))
		stats.Awards++
	}
}

func ApplyStageValues(stages []Stage, src []byte, corrections map[string]StageCorrection, stats *StageStats) error {
	src = uncomment(src)
	var own StageStats
	theirs, index, err := readStageTables(src, nil, &own)
	if err != nil {
		return err
	}
	ours := joinIndex(stages)
	calls := make([][][][]byte, len(theirs))
	coop := coopIndex(theirs, index)
	region := table(src, "levelBonus")
	n := len(bonusAny.FindAll(region, -1))
	stats.BonusCalls += n
	stats.Dropped += n
	for _, m := range bonusEntry.FindAllSubmatch(region, -1) {
		c := bonusCall.FindAllSubmatch(m[2], -1)
		if len(c) == 0 {
			continue
		}
		stats.Dropped -= len(c)
		at, ok := index[string(m[1])]
		if !ok {
			at, ok = coop[coopKey(string(m[1]))]
		}
		if !ok {
			stats.OrphanBonus++
			continue
		}
		calls[at] = append(calls[at], c...)
	}
	for at, t := range theirs {
		mine, ok := ours[joinKey(t.Name)]
		if !ok {
			stats.Beyond++
			continue
		}
		s := &stages[mine]
		if _, fixed := corrections[s.Name]; fixed {
			return fmt.Errorf("pipeline: stage %s has a weight correction and values from a second source; one would be discarded", s.Name)
		}
		if s.Stage.Attrs != t.Stage.Attrs {
			stats.SideConflicts++
			continue
		}
		if len(s.Stage.Tags) > 0 {
			stats.Awards -= len(s.Stage.Tags)
			stats.WithBonus--
		}
		s.Stage.Weights, s.Stage.Tags = t.Stage.Weights, nil
		pay(&s.Stage, calls[at], weightSum(*s), 1, stats)
		stats.Valued++
	}
	return nil
}

func (s *StageStats) Recount(stages []Stage) {
	s.Stages, s.WithBonus, s.Awards = len(stages), 0, 0
	for _, st := range stages {
		if len(st.Stage.Tags) > 0 {
			s.WithBonus++
			s.Awards += len(st.Stage.Tags)
		}
	}
}

func joinKey(name string) string {
	if k := coopKey(name); k != "" {
		return coopPrefix + ":" + k
	}
	s := strings.TrimSpace(name)
	if m := parenthesised.FindStringSubmatch(s); m != nil && hasHan(m[2]) && !hasHan(m[1]) {
		return strings.TrimSpace(m[2])
	}
	return s
}

func joinIndex(stages []Stage) map[string]int {
	out := map[string]int{}
	clash := map[string]bool{}
	for i, s := range stages {
		key := joinKey(s.Name)
		if _, seen := out[key]; seen {
			clash[key] = true
			continue
		}
		out[key] = i
	}
	for key := range clash {
		delete(out, key)
	}
	return out
}

var stagePrefix = map[string]string{
	"联盟委托": "Commission",
	"协战":   "Co-op",
	"关卡":   "Story",
	"竞技场":  "Arena",
	"活动地图": "Event",
}

func SplitStageName(name string) (display, mode string) {
	s := strings.TrimSpace(name)
	if label, rest, found := strings.Cut(s, ":"); found {
		if m, ok := stagePrefix[strings.TrimSpace(label)]; ok {
			return StageDisplayName(rest), m
		}
	}
	return StageDisplayName(s), ""
}

func DisplayName(s Stage) string {
	if s.Display != "" {
		return s.Display
	}
	display, _ := SplitStageName(s.Name)
	return display
}

func ApplyStageNames(stages []Stage, src []byte) ([]string, error) {
	var own StageStats
	theirs, _, err := readStageTables(uncomment(src), nil, &own)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	for _, t := range theirs {
		if display, _ := SplitStageName(t.Name); !hasHan(display) {
			names[joinKey(t.Name)] = display
		}
	}
	var missing []string
	for i := range stages {
		display, _ := SplitStageName(stages[i].Name)
		if !hasHan(display) {
			continue
		}
		if english, ok := names[joinKey(stages[i].Name)]; ok {
			stages[i].Display = english
		} else {
			missing = append(missing, display)
		}
	}
	return missing, nil
}

var stageToken = strings.NewReplacer(
	"支", "Side ",
	"绫罗", "Lunar ",
	"奥兰多", "Orlando ",
	"洁洁云", "Yvette ",
	"克洛里斯", "Chloris ",
)

func StageDisplayName(name string) string {
	s := strings.TrimSpace(name)
	if m := parenthesised.FindStringSubmatch(s); m != nil {
		if english := strings.TrimSpace(m[1]); english != "" && !hasHan(english) {
			return nameFixes.Replace(NormalizeName(english))
		}
	}
	return NormalizeName(stageToken.Replace(s))
}

var nameFixes = strings.NewReplacer("Neve - ", "Neva - ", "Ofiice", "Office")

func hasHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func uncomment(src []byte) []byte {
	out := make([]byte, len(src))
	copy(out, src)
	var quote byte
	for i := 0; i < len(out); i++ {
		c := out[i]
		switch {
		case quote != 0:
			if c == '\\' {
				i++
			} else if c == quote || c == '\n' {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '/' && i+1 < len(out) && out[i+1] == '/':
			for ; i < len(out) && out[i] != '\n'; i++ {
				out[i] = ' '
			}
		case c == '/' && i+1 < len(out) && out[i+1] == '*':
			for ; i < len(out); i++ {
				if out[i] == '*' && i+1 < len(out) && out[i+1] == '/' {
					out[i], out[i+1] = ' ', ' '
					i++
					break
				}
				if out[i] != '\n' {
					out[i] = ' '
				}
			}
		}
	}
	return out
}

func coopKey(name string) string {
	body, ok := strings.CutPrefix(name, coopPrefix)
	if !ok {
		return ""
	}
	body = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(body), ":"))
	open := strings.LastIndex(body, "(")
	if open < 0 || !strings.HasSuffix(body, ")") {
		return strings.ReplaceAll(body, "的", "")
	}
	head, inside := strings.TrimSpace(body[:open]), strings.TrimSpace(body[open+1:len(body)-1])
	if !strings.Contains(inside, "-") {
		character, _, _ := strings.Cut(head, "-")
		inside = strings.TrimSpace(character) + "-" + inside
	}
	return strings.ReplaceAll(inside, "的", "")
}

func coopIndex(stages []Stage, index map[string]int) map[string]int {
	out := map[string]int{}
	clash := map[string]bool{}
	for name, at := range index {
		key := coopKey(name)
		if key == "" {
			continue
		}
		if _, seen := out[key]; seen {
			clash[key] = true
			continue
		}
		out[key] = at
	}
	for key := range clash {
		delete(out, key)
	}
	return out
}

const coopPrefix = "协战"

var (
	ruleEntry    = regexp.MustCompile(`(?m)^\s*['"]([^'"]+)['"]\s*:\s*(.+)$`)
	quoted       = regexp.MustCompile(`['"]([^'"]*)['"]`)
	filterStyles = regexp.MustCompile(`Filter\(\s*"([^"]*)"`)
)

func applyRules(src []byte, stages []Stage, index map[string]int, stats *StageStats) {
	coop := coopIndex(stages, index)
	find := func(name string) (int, bool) {
		if at, ok := index[name]; ok {
			return at, true
		}
		at, ok := coop[coopKey(name)]
		return at, ok
	}
	flag := func(at int, styles []string) {
		st := &stages[at]
		if st.Rules == nil {
			st.Rules = &StageRules{}
			stats.Ruled++
		}
		for _, style := range styles {
			if !slices.Contains(st.Rules.Styles, style) {
				st.Rules.Styles = append(st.Rules.Styles, style)
			}
		}
	}

	for _, m := range ruleEntry.FindAllSubmatch(table(src, "levelFilters"), -1) {
		at, ok := find(string(m[1]))
		if !ok {
			continue
		}
		stats.RuleEntries++
		var styles []string
		if f := filterStyles.FindSubmatch(m[2]); f != nil {
			for _, part := range strings.Split(string(f[1]), "/") {
				if style, ok := StyleFromStageTag(part); ok {
					styles = append(styles, style)
					continue
				}
				english, _, _ := strings.Cut(part, "(")
				if english = strings.TrimSpace(english); english != "" {
					styles = append(styles, english)
				}
			}
		}
		flag(at, styles)
	}

	for _, m := range ruleEntry.FindAllSubmatch(table(src, "addHintInfo"), -1) {
		at, ok := find(string(m[1]))
		if !ok {
			continue
		}
		fields := quoted.FindAllSubmatch(m[2], -1)
		ruled := false
		for i, f := range fields {
			if i == 1 {
				continue
			}
			if strings.TrimSpace(string(f[1])) != "" {
				ruled = true
				break
			}
		}
		if ruled {
			stats.RuleEntries++
			flag(at, nil)
		}
	}
	for i := range stages {
		if stages[i].Rules != nil {
			slices.Sort(stages[i].Rules.Styles)
		}
	}
}

func table(src []byte, name string) []byte {
	start := bytes.Index(src, []byte("var "+name))
	if start < 0 {
		return nil
	}
	end := bytes.Index(src[start:], []byte("\n};"))
	if end < 0 {
		return nil
	}
	return src[start : start+end]
}
