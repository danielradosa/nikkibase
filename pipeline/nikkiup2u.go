package pipeline

import (
	"bufio"
	"bytes"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
)

type Entry struct {
	Item      scoring.Item
	Name      string
	Position  string
	Grades    [5]string
	Subgrades [5]string
	Rarity    int
	Suit      string
}

type Stage struct {
	Name         string
	Mode         string
	Display      string
	Stage        scoring.Stage
	Rules        *StageRules
	Variants     map[string]scoring.Stage
	VariantRules map[string]StageRules
}

type StageRules struct {
	Styles  []string
	Require [][]int
}

var gradeColumns = [...]struct{ first, second int }{
	{scoring.Gorgeous, scoring.Simple},
	{scoring.Elegant, scoring.Lively},
	{scoring.Mature, scoring.Cute},
	{scoring.Sexy, scoring.Pure},
	{scoring.Cool, scoring.Warm},
}

var category = map[string]scoring.Slot{
	"发型": scoring.Hair, "连衣裙": scoring.Dress, "外套": scoring.Coat,
	"上装": scoring.Top, "下装": scoring.Bottom, "鞋子": scoring.Shoes, "妆容": scoring.Makeup,
}

var idPrefix = map[scoring.Slot]int{
	scoring.Hair: 1, scoring.Dress: 2, scoring.Coat: 3, scoring.Top: 4,
	scoring.Bottom: 5, scoring.Hosiery: 6, scoring.Shoes: 7, scoring.Accessory: 8,
	scoring.Makeup: 9,
}

var jsString = regexp.MustCompile(`'((?:[^'\\]|\\.)*)'`)

func ParseWardrobe(src []byte) (entries []Entry, skipped int, err error) {
	tags := map[string]int{}
	sc := bufio.NewScanner(bytes.NewReader(src))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if !bytes.HasPrefix(line, []byte("['")) {
			continue
		}
		fields := jsString.FindAllStringSubmatch(string(line), -1)
		if len(fields) < 15 {
			continue
		}
		e, ok := entry(fields, tags)
		if !ok {
			skipped++
			continue
		}
		entries = append(entries, e)
	}
	return entries, skipped, sc.Err()
}

func entry(fields [][]string, tags map[string]int) (Entry, bool) {
	name, cat, num := fields[0][1], fields[1][1], fields[2][1]
	slot, position, ok := classify(cat)
	if !ok {
		return Entry{}, false
	}
	n, err := strconv.Atoi(num)
	if err != nil {
		return Entry{}, false
	}
	id := idPrefix[slot]*10000 + n
	if slot == scoring.Accessory && n >= 10000 {
		id = 170000 + n
	}

	it := scoring.Item{ID: id, Slot: slot}
	var grades [5]string
	for p, col := range gradeColumns {
		first, second := strings.TrimSpace(fields[4+p*2][1]), strings.TrimSpace(fields[5+p*2][1])
		switch {
		case first != "":
			it.Attrs[p], it.Stats[p], grades[p] = int8(col.first), Stat(first, slot), first
		case second != "":
			it.Attrs[p], it.Stats[p], grades[p] = int8(col.second), Stat(second, slot), second
		default:
			it.Attrs[p] = int8(col.first)
		}
	}
	if tag := strings.TrimSpace(fields[14][1]); tag != "" {
		if _, seen := tags[tag]; !seen {
			tags[tag] = len(tags)
		}
		it.Tags = []int{tags[tag]}
	}

	if english, _, found := strings.Cut(name, "("); found {
		name = strings.TrimSpace(english)
	}
	name = strings.NewReplacer(`\'`, "'", `\"`, `"`, `\\`, `\`).Replace(name)
	return Entry{Item: it, Name: name, Position: position, Grades: grades}, true
}

func classify(cat string) (scoring.Slot, string, bool) {
	if slot, ok := category[cat]; ok {
		return slot, cat, true
	}
	switch {
	case strings.HasPrefix(cat, "袜子"):
		return scoring.Hosiery, cat, true
	case strings.HasPrefix(cat, "饰品"):
		return scoring.Accessory, cat, true
	}
	return 0, "", false
}

var gradeBase = map[string]float64{
	"SSS": 213.3, "SS": 174.2, "S": 139.3, "A": 112.7, "B": 87.3, "C": 54.5,
}

func Stat(grade string, slot scoring.Slot) int {
	base, ok := gradeBase[strings.ToUpper(grade)]
	if !ok {
		return 0
	}
	return int(math.Round(base * scoring.SlotSize(slot)))
}

func Positions(entries []Entry, owned []int) ([]optimizer.Position, int) {
	have := make(map[int]bool, len(owned))
	for _, id := range owned {
		have[id] = true
	}
	byPlace := map[string][]scoring.Item{}
	places := map[string]SubSlot{}
	unplaced := 0
	placed := map[int]bool{}
	var order []string
	for _, e := range entries {
		if !have[e.Item.ID] {
			continue
		}
		if placed[e.Item.ID] {
			unplaced++
			continue
		}
		place, err := ResolvePosition(e.Position, e.Item.Slot)
		if err != nil {
			unplaced++
			continue
		}
		if _, seen := byPlace[place.Name]; !seen {
			order = append(order, place.Name)
			places[place.Name] = place
		}
		byPlace[place.Name] = append(byPlace[place.Name], e.Item)
		placed[e.Item.ID] = true
	}
	positions := make([]optimizer.Position, 0, len(order))
	for _, name := range order {
		place := places[name]
		positions = append(positions, optimizer.Position{
			Items:     byPlace[name],
			Group:     uint8(GroupIndex(place.Group)),
			Exclusive: place.Exclusive(),
		})
	}
	return positions, unplaced
}
