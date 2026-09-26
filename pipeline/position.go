package pipeline

import (
	"fmt"
	"sort"
	"strings"

	"github.com/danielradosa/nikkibase/core/scoring"
)

type SubSlot struct {
	Name    string
	Slot    scoring.Slot
	Display string
	Group   string
}

const (
	GroupTorso    = "torso"
	GroupHandheld = "handheld"
)

var subSlots = []SubSlot{
	{"hair", scoring.Hair, "Hair", ""},
	{"dress", scoring.Dress, "Dress", GroupTorso},
	{"coat", scoring.Coat, "Coat", ""},
	{"top", scoring.Top, "Top", GroupTorso},
	{"bottom", scoring.Bottom, "Bottom", GroupTorso},
	{"hosiery_leglet", scoring.Hosiery, "Leglets", ""},
	{"hosiery_socks", scoring.Hosiery, "Hosiery", ""},
	{"shoes", scoring.Shoes, "Shoes", ""},
	{"makeup", scoring.Makeup, "Makeup", ""},
	{"accessory_hair_ornament", scoring.Accessory, "Hair ornament", ""},
	{"accessory_veil", scoring.Accessory, "Veil", ""},
	{"accessory_hairpin", scoring.Accessory, "Hairpin", ""},
	{"accessory_ears", scoring.Accessory, "Ears", ""},
	{"accessory_earrings", scoring.Accessory, "Earrings", ""},
	{"accessory_scarf", scoring.Accessory, "Scarf", ""},
	{"accessory_necklace", scoring.Accessory, "Necklace", ""},
	{"accessory_hand_right", scoring.Accessory, "Right hand", ""},
	{"accessory_hand_left", scoring.Accessory, "Left hand", ""},
	{"accessory_hand_both", scoring.Accessory, "Gloves", ""},
	{"accessory_handheld_right", scoring.Accessory, "Held, right", GroupHandheld},
	{"accessory_handheld_left", scoring.Accessory, "Held, left", GroupHandheld},
	{"accessory_handheld_both", scoring.Accessory, "Held, both hands", GroupHandheld},
	{"accessory_waist", scoring.Accessory, "Waist", ""},
	{"accessory_face", scoring.Accessory, "Face", ""},
	{"accessory_brooch", scoring.Accessory, "Brooch", ""},
	{"accessory_tattoo", scoring.Accessory, "Tattoo", ""},
	{"accessory_wings", scoring.Accessory, "Wings", ""},
	{"accessory_tail", scoring.Accessory, "Tail", ""},
	{"accessory_foreground", scoring.Accessory, "Foreground", ""},
	{"accessory_background", scoring.Accessory, "Background", ""},
	{"accessory_headwear", scoring.Accessory, "Headwear", ""},
	{"accessory_ground", scoring.Accessory, "Ground", ""},
	{"accessory_skin", scoring.Accessory, "Skin", ""},
	{"spirit", scoring.Spirit, "Spirit", ""},
}

var aliases = map[string]string{
	"hair": "hair", "dress": "dress", "coat": "coat", "top": "top",
	"bottom": "bottom", "shoes": "shoes", "makeup": "makeup", "spirit": "spirit",
	"hosiery":        "hosiery_socks",
	"hosiery:leglet": "hosiery_leglet",

	"accessory:hair ornament":    "accessory_hair_ornament",
	"accessory:veil":             "accessory_veil",
	"accessory:hairpin":          "accessory_hairpin",
	"accessory:ears":             "accessory_ears",
	"accessory:earrings":         "accessory_earrings",
	"accessory:scarf":            "accessory_scarf",
	"accessory:necklace":         "accessory_necklace",
	"accessory:bracelet (right)": "accessory_hand_right",
	"accessory:bracelet (left)":  "accessory_hand_left",
	"accessory:gloves":           "accessory_hand_both",
	"accessory:handheld (right)": "accessory_handheld_right",
	"accessory:handheld (left)":  "accessory_handheld_left",
	"accessory:handheld (both)":  "accessory_handheld_both",
	"accessory:waist":            "accessory_waist",
	"accessory:face":             "accessory_face",
	"accessory:brooch":           "accessory_brooch",
	"accessory:tattoo":           "accessory_tattoo",
	"accessory:wings":            "accessory_wings",
	"accessory:tail":             "accessory_tail",
	"accessory:foreground":       "accessory_foreground",
	"accessory:background":       "accessory_background",
	"accessory:head ornament":    "accessory_headwear",
	"accessory:ground":           "accessory_ground",
	"accessory:skin":             "accessory_skin",

	"袜子-袜套":    "hosiery_leglet",
	"袜子-袜子":    "hosiery_socks",
	"饰品-头饰·发饰": "accessory_hair_ornament",
	"饰品-头饰·头纱": "accessory_veil",
	"饰品-头饰·发卡": "accessory_hairpin",
	"饰品-头饰·耳朵": "accessory_ears",
	"饰品-耳饰":    "accessory_earrings",
	"饰品-颈饰·围巾": "accessory_scarf",
	"饰品-颈饰·项链": "accessory_necklace",
	"饰品-手饰·右":  "accessory_hand_right",
	"饰品-手饰·左":  "accessory_hand_left",
	"饰品-手饰·双":  "accessory_hand_both",
	"饰品-手持·右":  "accessory_handheld_right",
	"饰品-手持·左":  "accessory_handheld_left",
	"饰品-手持·双":  "accessory_handheld_both",
	"饰品-腰饰":    "accessory_waist",
	"饰品-特殊·面饰": "accessory_face",
	"饰品-特殊·胸饰": "accessory_brooch",
	"饰品-特殊·纹身": "accessory_tattoo",
	"饰品-特殊·翅膀": "accessory_wings",
	"饰品-特殊·尾巴": "accessory_tail",
	"饰品-特殊·前景": "accessory_foreground",
	"饰品-特殊·后景": "accessory_background",
	"饰品-特殊·顶饰": "accessory_headwear",
	"饰品-特殊·地面": "accessory_ground",
	"饰品-皮肤":    "accessory_skin",
	"饰品-特殊·皮肤": "accessory_skin",
	"发型":       "hair",
	"连衣裙":      "dress",
	"外套":       "coat",
	"上装":       "top",
	"下装":       "bottom",
	"鞋子":       "shoes",
	"妆容":       "makeup",
	"萤光之灵":     "spirit",
}

var byName = func() map[string]SubSlot {
	m := make(map[string]SubSlot, len(subSlots))
	for _, s := range subSlots {
		m[s.Name] = s
	}
	return m
}()

func SubSlots() []SubSlot { return subSlots }

func ResolvePosition(raw string, slot scoring.Slot) (SubSlot, error) {
	key := strings.ToLower(strings.TrimSpace(raw))
	name, ok := aliases[key]
	if !ok {
		if _, canonical := byName[key]; !canonical {
			return SubSlot{}, fmt.Errorf("no wearable place is named %q", raw)
		}
		name = key
	}
	s := byName[name]
	if s.Slot != slot {
		return SubSlot{}, fmt.Errorf("%q is a %s place, and the item is a %s",
			raw, SlotName(s.Slot), SlotName(slot))
	}
	return s, nil
}

func PositionIndex(name string) (int, bool) {
	for i, s := range subSlots {
		if s.Name == name {
			return i, true
		}
	}
	return 0, false
}

func GroupIndex(group string) int {
	if group == "" {
		return 0
	}
	groups := groupNames()
	for i, g := range groups {
		if g == group {
			return i + 1
		}
	}
	return 0
}

func groupNames() []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range subSlots {
		if s.Group != "" && !seen[s.Group] {
			seen[s.Group] = true
			out = append(out, s.Group)
		}
	}
	sort.Strings(out)
	return out
}

func Canonicalise(entries []Entry) error {
	unplaced := map[string]int{}
	misfiled := map[string]int{}
	for i := range entries {
		s, err := ResolvePosition(entries[i].Position, entries[i].Item.Slot)
		if err == nil {
			entries[i].Position = s.Name
			continue
		}
		key := strings.ToLower(strings.TrimSpace(entries[i].Position))
		_, known := aliases[key]
		if _, canonical := byName[key]; known || canonical {
			misfiled[fmt.Sprintf("%q on a %s", entries[i].Position, SlotName(entries[i].Item.Slot))]++
			continue
		}
		unplaced[entries[i].Position]++
	}
	if len(unplaced) == 0 && len(misfiled) == 0 {
		return nil
	}
	var v Violations
	if len(unplaced) > 0 {
		v = append(v, "these are not places the game has: "+counted(unplaced))
	}
	if len(misfiled) > 0 {
		v = append(v, "these items are filed under a place their slot contradicts: "+counted(misfiled))
	}
	return v
}

func counted(m map[string]int) string {
	out := make([]string, 0, len(m))
	for k, n := range m {
		out = append(out, fmt.Sprintf("%s (%d items)", k, n))
	}
	sort.Strings(out)
	return truncate(out)
}

func (s SubSlot) Exclusive() bool {
	switch s.Name {
	case "accessory_handheld_both", "dress":
		return true
	}
	return false
}
