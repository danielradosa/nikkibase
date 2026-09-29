package pipeline

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/danielradosa/nikkibase/core/scoring"
)

type Duplicate struct {
	ID         int    `json:"id"`
	Keep       string `json:"keep"`
	Drop       string `json:"drop"`
	Page       string `json:"dropWikiPage"`
	RealID     int    `json:"correctIdForDropped"`
	Basis      string `json:"basis"`
	Confidence string `json:"confidence"`
}

type SlotOverride struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Slot          string `json:"slot"`
	Was           string `json:"was"`
	Position      string `json:"position"`
	PositionBasis string `json:"positionBasis"`
	Basis         string `json:"basis"`
}

type Excluded struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Slot   string `json:"slot"`
	Reason string `json:"reason"`
	Effect string `json:"effect"`
}

type NameOverride struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Was   string `json:"was"`
	Basis string `json:"basis"`
}

type GradeOverride struct {
	ID        int    `json:"id"`
	Attribute string `json:"attribute"`
	Grade     string `json:"grade"`
	Was       string `json:"was"`
	Basis     string `json:"basis"`
}

var attributeNames = [...]string{
	scoring.Gorgeous: "Gorgeous", scoring.Simple: "Simple", scoring.Elegant: "Elegant", scoring.Lively: "Lively",
	scoring.Mature: "Mature", scoring.Cute: "Cute", scoring.Sexy: "Sexy", scoring.Pure: "Pure",
	scoring.Warm: "Warm", scoring.Cool: "Cool",
}

func attributeOf(name string) (int8, bool) {
	for a, n := range attributeNames {
		if n == name {
			return int8(a), true
		}
	}
	return 0, false
}

type Corrections struct {
	Note       string          `json:"note"`
	DerivedOn  string          `json:"derivedOn"`
	Duplicates []Duplicate     `json:"duplicates"`
	Overrides  []SlotOverride  `json:"slotOverrides"`
	Excluded   []Excluded      `json:"excluded"`
	Names      []NameOverride  `json:"nameOverrides"`
	Shown      []NameOverride  `json:"displayNames"`
	Grades     []GradeOverride `json:"gradeOverrides"`
}

type IDCorrections struct {
	Owner    map[int]string
	Drop     map[int]string
	RealID   map[int]int
	Slot     map[int]scoring.Slot
	Position map[int]string
	Excluded map[int]string
	Name     map[int]NameOverride
	Shown    map[int]NameOverride
	Grade    map[int][]GradeOverride
}

func ReadIDCorrections(b []byte) (*IDCorrections, error) {
	var c Corrections
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("id corrections: %w", err)
	}
	out := &IDCorrections{
		Owner: map[int]string{}, Drop: map[int]string{}, RealID: map[int]int{}, Slot: map[int]scoring.Slot{},
		Position: map[int]string{}, Excluded: map[int]string{}, Name: map[int]NameOverride{},
		Shown: map[int]NameOverride{}, Grade: map[int][]GradeOverride{},
	}
	for _, n := range c.Shown {
		switch {
		case n.ID <= 0 || n.Name == "" || n.Was == "":
			return nil, fmt.Errorf("id corrections: %d must give an item ID, the name to show and the one its sources give", n.ID)
		case n.Basis == "":
			return nil, fmt.Errorf("id corrections: %d gives no basis for the name it shows", n.ID)
		case n.Name == n.Was:
			return nil, fmt.Errorf("id corrections: %d shows the name its sources already give", n.ID)
		}
		if _, dup := out.Shown[n.ID]; dup {
			return nil, fmt.Errorf("id corrections: %d is given two names to show", n.ID)
		}
		out.Shown[n.ID] = n
	}
	for _, g := range c.Grades {
		a, known := attributeOf(g.Attribute)
		switch {
		case g.ID <= 0 || !known:
			return nil, fmt.Errorf("id corrections: %d must give an item ID and one of its attributes", g.ID)
		case gradeBase[g.Grade] == 0 || gradeBase[g.Was] == 0:
			return nil, fmt.Errorf("id corrections: %d must give the grade to keep and the one its sources give", g.ID)
		case g.Grade == g.Was:
			return nil, fmt.Errorf("id corrections: %d keeps the grade its sources already give", g.ID)
		case g.Basis == "":
			return nil, fmt.Errorf("id corrections: %d gives no basis for its grade", g.ID)
		}
		for _, o := range out.Grade[g.ID] {
			if b, _ := attributeOf(o.Attribute); b/2 == a/2 {
				return nil, fmt.Errorf("id corrections: %d is given two grades for %s", g.ID, g.Attribute)
			}
		}
		out.Grade[g.ID] = append(out.Grade[g.ID], g)
	}
	for _, n := range c.Names {
		switch {
		case n.ID <= 0 || n.Name == "" || n.Was == "":
			return nil, fmt.Errorf("id corrections: %d must give an item ID, the name its sources agree on and the one it was given", n.ID)
		case n.Basis == "":
			return nil, fmt.Errorf("id corrections: %d gives no basis for its name", n.ID)
		case sameGarment(n.Name, n.Was):
			return nil, fmt.Errorf("id corrections: %d renames %q to the same garment", n.ID, n.Was)
		}
		if _, dup := out.Name[n.ID]; dup {
			return nil, fmt.Errorf("id corrections: %d is renamed twice", n.ID)
		}
		out.Name[n.ID] = n
	}
	for _, d := range c.Duplicates {
		switch {
		case d.Keep == "" || d.Drop == "":
			return nil, fmt.Errorf("id corrections: %d must name the garment kept and the one dropped", d.ID)
		case d.Basis == "":
			return nil, fmt.Errorf("id corrections: %d gives no basis", d.ID)
		case d.Keep == d.Drop:
			return nil, fmt.Errorf("id corrections: %d keeps and drops the same name", d.ID)
		case d.RealID < 0 || d.RealID == d.ID:
			return nil, fmt.Errorf("id corrections: %d gives %d as the dropped garment's own ID", d.ID, d.RealID)
		}
		if _, dup := out.Owner[d.ID]; dup {
			return nil, fmt.Errorf("id corrections: %d is listed twice", d.ID)
		}
		out.Owner[d.ID], out.Drop[d.ID] = d.Keep, d.Drop
		if d.RealID > 0 {
			out.RealID[d.ID] = d.RealID
		}
	}
	for _, o := range c.Overrides {
		slot, ok := slotByName(o.Slot)
		if !ok {
			return nil, fmt.Errorf("id corrections: %d names no slot %q", o.ID, o.Slot)
		}
		if o.Basis == "" {
			return nil, fmt.Errorf("id corrections: %d gives no basis", o.ID)
		}
		if want := SlotOfID(o.ID); slot != want {
			return nil, fmt.Errorf("id corrections: %d is put in %s, and its ID says %s",
				o.ID, o.Slot, SlotName(want))
		}
		if o.Position == "" || o.PositionBasis == "" {
			return nil, fmt.Errorf("id corrections: %d moves slot and gives no wearable place", o.ID)
		}
		if _, ok := ResolvePosition(o.Position, slot); ok != nil {
			return nil, fmt.Errorf("id corrections: %d: %w", o.ID, ok)
		}
		out.Slot[o.ID], out.Position[o.ID] = slot, o.Position
	}
	for _, e := range c.Excluded {
		if e.Reason == "" || e.Effect == "" {
			return nil, fmt.Errorf("id corrections: excluded %d must give a reason and its effect", e.ID)
		}
		out.Excluded[e.ID] = e.Reason
	}
	return out, nil
}

func (c *IDCorrections) DisplayNames(entries []Entry, names ItemNames) (map[int]string, error) {
	given := make(map[int]string, len(entries)+len(names.Calc))
	for id, name := range names.Calc {
		given[id] = gameSpelling(name)
	}
	for _, e := range entries {
		given[e.Item.ID] = plainName(e, names)
	}
	out := make(map[int]string, len(c.Shown))
	var wrong []string
	for _, id := range slices.Sorted(maps.Keys(c.Shown)) {
		n := c.Shown[id]
		name, ok := given[id]
		switch {
		case !ok:
			wrong = append(wrong, fmt.Sprintf("%d is not in the catalogue", id))
		case name != n.Was:
			wrong = append(wrong, fmt.Sprintf("%d is %q, not %q", id, name, n.Was))
		default:
			out[id] = n.Name
		}
	}
	if len(wrong) > 0 {
		return nil, fmt.Errorf("id corrections: %d display names no longer fit what the sources give: %s",
			len(wrong), strings.Join(wrong, "; "))
	}
	return out, nil
}

func (c *IDCorrections) nameOf(id int, name string) string {
	if n, ok := c.Name[id]; ok && sameGarment(name, n.Was) {
		return n.Name
	}
	return name
}

func (c *IDCorrections) Apply(entries []Entry) ([]Entry, error) {
	for i := range entries {
		entries[i].Name = c.nameOf(entries[i].Item.ID, entries[i].Name)
		if slot, ok := c.Slot[entries[i].Item.ID]; ok {
			entries[i].Item.Slot = slot
			entries[i].Position = c.Position[entries[i].Item.ID]
			for p := range entries[i].Item.Stats {
				entries[i].Item.Stats[p] = Stat(entries[i].Grades[p], slot)
			}
		}
	}
	out, err := c.dropDuplicates(entries, true)
	if err != nil {
		return nil, err
	}
	return out, c.keepGrades(out)
}

func (c *IDCorrections) keepGrades(entries []Entry) error {
	found := make(map[int]bool, len(c.Grade))
	var wrong []string
	for i := range entries {
		e := &entries[i]
		for _, g := range c.Grade[e.Item.ID] {
			found[e.Item.ID] = true
			a, _ := attributeOf(g.Attribute)
			p := a / 2
			if e.Item.Attrs[p] != a || strings.ToUpper(e.Grades[p]) != g.Was {
				wrong = append(wrong, fmt.Sprintf("%d is %s %s, not %s %s",
					e.Item.ID, attributeNames[e.Item.Attrs[p]], e.Grades[p], g.Attribute, g.Was))
				continue
			}
			e.Grades[p] = g.Grade
			e.Item.Stats[p] = Stat(g.Grade, e.Item.Slot)
		}
	}
	for _, id := range slices.Sorted(maps.Keys(c.Grade)) {
		if !found[id] {
			wrong = append(wrong, fmt.Sprintf("%d is not in the catalogue", id))
		}
	}
	if len(wrong) > 0 {
		return fmt.Errorf("id corrections: %d grades no longer fit what the sources give: %s",
			len(wrong), strings.Join(wrong, "; "))
	}
	return nil
}

func (c *IDCorrections) DropDuplicates(entries []Entry) []Entry {
	out, _ := c.dropDuplicates(entries, false)
	return out
}

func (c *IDCorrections) misnumbered(id int, name string) bool {
	drop, listed := c.Drop[id]
	return listed && sameGarment(name, drop) && !sameGarment(name, c.Owner[id])
}

func (c *IDCorrections) dropDuplicates(entries []Entry, strict bool) ([]Entry, error) {
	staying := make(map[int]bool, len(entries))
	for _, e := range entries {
		if !c.misnumbered(e.Item.ID, e.Name) {
			staying[e.Item.ID] = true
		}
	}
	out := make([]Entry, 0, len(entries))
	seen := make(map[int]int, len(entries))
	var unresolved []string
	for _, e := range entries {
		if _, gone := c.Excluded[e.Item.ID]; gone {
			continue
		}
		if c.misnumbered(e.Item.ID, e.Name) {
			to := c.RealID[e.Item.ID]
			if _, gone := c.Excluded[to]; gone || to == 0 || staying[to] || SlotOfID(to) != e.Item.Slot {
				continue
			}
			staying[to] = true
			e.Item.ID = to
		}
		at, clash := seen[e.Item.ID]
		if !clash {
			seen[e.Item.ID] = len(out)
			out = append(out, e)
			continue
		}
		owner, known := c.Owner[e.Item.ID]
		if !known {
			if !strict {
				out = append(out, e)
				continue
			}
			unresolved = append(unresolved, fmt.Sprintf("%d (%s, %s)", e.Item.ID, out[at].Name, e.Name))
			continue
		}
		if sameGarment(e.Name, owner) && !sameGarment(out[at].Name, owner) {
			out[at] = e
		}
	}
	if len(unresolved) > 0 {
		sort.Strings(unresolved)
		return nil, Violations{fmt.Sprintf(
			"%d item IDs name two garments and no correction covers them: %s",
			len(unresolved), truncate(unresolved))}
	}
	return out, nil
}

type Placed struct {
	Slot     scoring.Slot
	Position string
	Name     string
	Grades   [5]string
}

func (p Placed) samePlace(q Placed) bool { return p.Slot == q.Slot && p.Position == q.Position }

func PlacesOf(entries []Entry) map[int]Placed {
	out := make(map[int]Placed, len(entries))
	torn := map[int]bool{}
	for _, e := range entries {
		p := Placed{e.Item.Slot, e.Position, e.Name, e.Grades}
		if prev, seen := out[e.Item.ID]; seen && !prev.samePlace(p) {
			torn[e.Item.ID] = true
		}
		out[e.Item.ID] = p
	}
	for id := range torn {
		delete(out, id)
	}
	return out
}

func CheckGarments(entries []Entry, tables map[int]Placed, names map[int]string) []string {
	var out []string
	for _, e := range entries {
		t, ok := tables[e.Item.ID]
		if !ok || sameGarmentAs(e, t) {
			continue
		}
		if n, ok := names[e.Item.ID]; ok && sameGarment(e.Name, n) {
			continue
		}
		out = append(out, fmt.Sprintf("%d (%q here, and the packed table has %q)", e.Item.ID, e.Name, t.Name))
	}
	sort.Strings(out)
	return out
}

func sameGarmentAs(e Entry, p Placed) bool {
	return sameGarment(e.Name, p.Name) || e.Grades == p.Grades
}

func sameGarment(a, b string) bool {
	return normalizeGarment(a) == normalizeGarment(b)
}

func normalizeGarment(s string) string {
	if i := strings.IndexAny(s, "(/"); i > 0 {
		s = s[:i]
	}
	s = strings.ToLower(strings.TrimSpace(s))
	s = garmentSeparators.Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

var garmentSeparators = strings.NewReplacer("·", " ", "-", " ", "–", " ", "—", " ", "’", "'", "‘", "'")

func slotByName(name string) (scoring.Slot, bool) {
	for s := scoring.Hair; s <= scoring.Spirit; s++ {
		if SlotName(s) == strings.ToLower(strings.TrimSpace(name)) {
			return s, true
		}
	}
	return 0, false
}
