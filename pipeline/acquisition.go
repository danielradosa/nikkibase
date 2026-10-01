package pipeline

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

var AcquisitionKinds = []string{
	"store", "craft", "evolve", "customize", "reconstruct", "stage", "pavilion", "event",
	"recharge", "signin", "suit", "association", "dream", "gift", "achievement", "other",
}

type Acquisition struct {
	Kind   string
	Text   string
	From   []Ingredient
	Cost   []Cost
	Recipe string
	Stage  string
	Level  string
	Last   string
	Past   bool
	CN     bool
}

type Ingredient struct {
	ID  int
	Qty int
}

type Cost struct {
	Amount int
	Unit   string
}

type AcquisitionCatalogue struct {
	Names             map[int]string
	Shown             map[int]string
	Calc              map[int]string
	Display           map[int]string
	Suits             map[int]string
	Stages            map[string]bool
	PartAliases       []SuitPartAlias
	IngredientAliases []IngredientAlias
}

func (c AcquisitionCatalogue) shown(id int) string {
	if name, ok := c.Shown[id]; ok {
		return inGameSpelling(name, c.Calc[id])
	}
	if name, ok := c.Display[id]; ok {
		return name
	}
	return c.Names[id]
}

func NewAcquisitionCatalogue(entries []Entry, names ItemNames, stages []Stage) AcquisitionCatalogue {
	extra := names.Calc
	cat := AcquisitionCatalogue{
		Names:   make(map[int]string, len(entries)+len(extra)),
		Shown:   names.Shown,
		Calc:    names.Calc,
		Display: make(map[int]string, len(entries)+len(extra)),
		Suits:   make(map[int]string, len(entries)),
		Stages:  make(map[string]bool, len(stages)),
	}
	for _, e := range entries {
		cat.Names[e.Item.ID] = sourceName(e, names)
		cat.Display[e.Item.ID] = plainName(e, names)
		if e.Suit != "" {
			cat.Suits[e.Item.ID] = e.Suit
		}
	}
	for id, name := range extra {
		if _, ok := cat.Names[id]; !ok {
			cat.Names[id] = calcName(name)
			cat.Display[id] = inGameSpelling(name, name)
		}
	}
	for _, s := range stages {
		cat.Stages[StageKey(s.Mode, DisplayName(s))] = true
	}
	return cat
}

func StageKey(mode, name string) string { return mode + "/" + name }

func storyAcquisition(name, level string, cat AcquisitionCatalogue) Acquisition {
	a := Acquisition{Kind: "stage", Text: "Story " + name, Level: level}
	if level != "" {
		a.Text += " (" + level + ")"
	}
	if key := StageKey("Story", name); cat.Stages[key] {
		a.Stage = key
	}
	return a
}

func storyName(volume int, chapter, number string, side bool) string {
	var b strings.Builder
	switch volume {
	case 2:
		b.WriteString("II-")
	case 3:
		b.WriteString("III-")
	}
	b.WriteString(chapter)
	b.WriteByte('-')
	if side {
		b.WriteString("Side ")
	}
	b.WriteString(number)
	return b.String()
}

func withCosts(text string, costs []Cost) string {
	if len(costs) == 0 {
		return text
	}
	parts := make([]string, len(costs))
	for i, c := range costs {
		parts[i] = thousands(c.Amount) + " " + c.Unit
	}
	return text + " · " + strings.Join(parts, " + ")
}

func thousands(n int) string {
	s := strconv.Itoa(n)
	if n < 0 {
		return s
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func ingredientList(items []namedIngredient) string {
	parts := make([]string, len(items))
	for i, it := range items {
		if it.qty > 0 {
			parts[i] = strconv.Itoa(it.qty) + "× " + it.name
		} else {
			parts[i] = it.name
		}
	}
	return strings.Join(parts, ", ")
}

type namedIngredient struct {
	name string
	id   int
	qty  int
}

func ingredients(items []namedIngredient) []Ingredient {
	var out []Ingredient
	for _, it := range items {
		if it.id != 0 {
			out = append(out, Ingredient{ID: it.id, Qty: it.qty})
		}
	}
	return out
}

func sameAcquisition(a, b Acquisition) bool {
	return a.Kind == b.Kind && a.Text == b.Text && a.Stage == b.Stage && a.Level == b.Level && a.Last == b.Last &&
		a.Recipe == b.Recipe && slices.Equal(a.From, b.From) && slices.Equal(a.Cost, b.Cost)
}

func appendAcquisition(list []Acquisition, a Acquisition) []Acquisition {
	if slices.ContainsFunc(list, func(b Acquisition) bool { return sameAcquisition(a, b) }) {
		return list
	}
	return append(list, a)
}

type AcquisitionStats struct {
	Catalogue  int
	Covered    int
	FromWiki   int
	FromPacked int
	FromSuits  int
	Named      int
	Based      int
	Events     int
	Reruns     int
}

func vagueKinds(a Acquisition) ([]string, bool) {
	switch {
	case a.Kind == "event" && a.Text == "Limited event":
		return []string{"event"}, true
	case a.Kind == "recharge" && (a.Text == "Recharge" || a.Text == "Event recharge"):
		return []string{"recharge", "store"}, true
	case (a.Kind == "customize" || a.Kind == "evolve") && len(a.From) == 0:
		return []string{a.Kind}, true
	case a.Kind == "suit" && a.Text == "Styling Gift Box":
		return []string{"suit"}, true
	case a.Kind == "dream" && a.Text == "Dream Weaver":
		return []string{"dream"}, true
	case a.Kind == "gift" && a.Text == "Gift":
		return nil, true
	}
	return nil, false
}

func vague(a Acquisition) bool {
	_, ok := vagueKinds(a)
	return ok
}

func allVague(list []Acquisition) bool {
	return len(list) > 0 && !slices.ContainsFunc(list, func(a Acquisition) bool { return !vague(a) })
}

func namedFrom(list, from []Acquisition) ([]Acquisition, bool) {
	if !allVague(list) {
		return list, false
	}
	var out []Acquisition
	for _, a := range list {
		kinds, _ := vagueKinds(a)
		var named []Acquisition
		for _, b := range from {
			if !vague(b) && slices.Contains(kinds, b.Kind) {
				named = append(named, b)
			}
		}
		if len(named) == 0 {
			named = []Acquisition{a}
		}
		for _, b := range named {
			out = appendAcquisition(out, b)
		}
	}
	return out, !slices.EqualFunc(out, list, sameAcquisition)
}

func basedByTable(wiki, packed []Acquisition) ([]Acquisition, bool) {
	out := make([]Acquisition, 0, len(wiki))
	changed := false
	for _, a := range wiki {
		if (a.Kind != "customize" && a.Kind != "evolve") || len(a.From) > 0 {
			out = appendAcquisition(out, a)
			continue
		}
		var based []Acquisition
		for _, b := range packed {
			if b.Kind == a.Kind && len(b.From) > 0 {
				based = append(based, b)
			}
		}
		if len(based) == 0 {
			out = appendAcquisition(out, a)
			continue
		}
		changed = true
		for _, b := range based {
			out = appendAcquisition(out, b)
		}
	}
	return out, changed
}

func rerunOn(list, reruns []Acquisition) ([]Acquisition, bool) {
	if len(reruns) == 0 || !allVague(list) || slices.ContainsFunc(list, func(a Acquisition) bool { return a.Kind != "recharge" }) {
		return list, false
	}
	var out []Acquisition
	for _, a := range reruns {
		out = appendAcquisition(out, a)
	}
	return out, true
}

func MergeAcquisition(cat AcquisitionCatalogue, wiki WikiAcquisition, packed map[int][]Acquisition) (map[int][]Acquisition, AcquisitionStats) {
	stats := AcquisitionStats{Catalogue: len(cat.Names)}
	out := make(map[int][]Acquisition, len(cat.Names))
	for id := range cat.Names {
		switch {
		case len(wiki.Items[id]) > 0:
			list, based := basedByTable(wiki.Items[id], packed[id])
			out[id] = list
			stats.FromWiki++
			if based {
				stats.Based++
			}
		case len(packed[id]) > 0:
			list, named := namedFrom(packed[id], wiki.Suits[id])
			out[id] = list
			stats.FromPacked++
			if named {
				stats.Named++
			}
		case len(wiki.Suits[id]) > 0:
			out[id] = wiki.Suits[id]
			stats.FromSuits++
		default:
			continue
		}
		if list, named := namedFrom(out[id], wiki.Events[id]); named {
			out[id] = list
			stats.Events++
		}
		if list, rerun := rerunOn(out[id], wiki.Reruns[id]); rerun {
			out[id] = list
			stats.Reruns++
		}
		stats.Covered++
	}
	return out, stats
}

func CheckAcquisition(acq map[int][]Acquisition, cat AcquisitionCatalogue, want Coverage) Violations {
	var v Violations
	covered := 0
	var bad []string
	for _, id := range slices.Sorted(maps.Keys(acq)) {
		list := acq[id]
		if len(list) > 0 {
			covered++
		}
		if _, ok := cat.Names[id]; !ok {
			bad = append(bad, fmt.Sprintf("%d is not in the catalogue", id))
		}
		for _, a := range list {
			if why := acquisitionProblem(id, a, cat); why != "" {
				bad = append(bad, fmt.Sprintf("%d: %s", id, why))
			}
		}
	}
	if len(bad) > 0 {
		v = append(v, fmt.Sprintf("%d acquisition entries are malformed: %s", len(bad), truncate(bad)))
	}
	if want.AcquisitionItems > 0 && covered < want.AcquisitionItems {
		v = append(v, fmt.Sprintf("%d items say how to get them, and the committed floor is %d",
			covered, want.AcquisitionItems))
	}
	return v
}

func CheckAcquisitionFallbacks(stats AcquisitionStats, want Coverage) Violations {
	var v Violations
	if want.EventItems > 0 && stats.Events < want.EventItems {
		v = append(v, fmt.Sprintf("%d items whose lines are all vague take the one event page that lists their suit, and the committed floor is %d, so the event pages may no longer be read",
			stats.Events, want.EventItems))
	}
	if want.RerunItems > 0 && stats.Reruns < want.RerunItems {
		v = append(v, fmt.Sprintf("%d items whose only lines are a plain recharge take the recharge channels the wiki's timeline reruns them on, and the committed floor is %d, so the timeline pages may no longer be read",
			stats.Reruns, want.RerunItems))
	}
	if stats.Events > want.MaxEventItems {
		v = append(v, fmt.Sprintf("%d items whose lines are all vague take the one event page that lists their suit; the committed ceiling is %d, so the event pages may have been misread",
			stats.Events, want.MaxEventItems))
	}
	if stats.Reruns > want.MaxRerunItems {
		v = append(v, fmt.Sprintf("%d items whose only lines are a plain recharge take the recharge channels the wiki's timeline reruns them on; the committed ceiling is %d, so the timeline pages may have been misread",
			stats.Reruns, want.MaxRerunItems))
	}
	return v
}

func CheckAcquisitionNames(stats WikiAcquisitionStats, want Coverage) Violations {
	if n := stats.Unresolved + stats.UnknownParts; n > want.MaxUnmatchedNames {
		return Violations{fmt.Sprintf("%d item names on wiki pages match no catalogue item (%d ingredients and bases, %d suit parts and rewards), and the committed ceiling is %d",
			n, stats.Unresolved, stats.UnknownParts, want.MaxUnmatchedNames)}
	}
	return nil
}

func acquisitionProblem(id int, a Acquisition, cat AcquisitionCatalogue) string {
	switch {
	case !slices.Contains(AcquisitionKinds, a.Kind):
		return fmt.Sprintf("unknown kind %q", a.Kind)
	case strings.TrimSpace(a.Text) == "":
		return "no text"
	case hasHan(a.Text) || hasHan(a.Recipe):
		return fmt.Sprintf("text %q is not English", a.Text)
	case a.Stage != "" && !cat.Stages[a.Stage]:
		return fmt.Sprintf("stage %q is not in the bundle", a.Stage)
	case a.Level != "" && a.Level != "Maiden" && a.Level != "Princess":
		return fmt.Sprintf("level %q", a.Level)
	}
	for _, in := range a.From {
		if _, ok := cat.Names[in.ID]; !ok {
			return fmt.Sprintf("ingredient %d is not in the catalogue", in.ID)
		}
		if in.Qty < 0 {
			return fmt.Sprintf("ingredient %d has quantity %d", in.ID, in.Qty)
		}
		if (a.Kind == "customize" || a.Kind == "evolve") && !madeFrom(in.ID, id) {
			return fmt.Sprintf("%s from %d, which is the item itself or in another slot", a.Kind, in.ID)
		}
	}
	for _, c := range a.Cost {
		if c.Amount <= 0 || c.Unit == "" || hasHan(c.Unit) {
			return fmt.Sprintf("cost %d %q", c.Amount, c.Unit)
		}
	}
	return ""
}
