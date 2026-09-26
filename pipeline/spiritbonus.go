package pipeline

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/danielradosa/nikkibase/core/scoring"
)

var (
	spiritBonusHeading = regexp.MustCompile(`(?im)^[ \t]*=+[ \t]*Skill[ \t]+Bonus[ \t]*=+[ \t]*$`)
	spiritBonusLevel   = regexp.MustCompile(`(?i)'''[ \t]*Level[ \t]*(\d+)[ \t]*:?[ \t]*'''.*?increase[sd]?[ \t]+by[ \t]+([\d,]+)`)
	wikiHeading        = regexp.MustCompile(`(?m)^[ \t]*=+[^=\n]`)
)

type SpiritBonusStats struct {
	Pages     int
	Parsed    int
	NoSection int
	NoLevels  int
	Rejected  int
}

func ParseSpiritBonuses(r io.Reader, known map[int]bool) (map[int]int, SpiritBonusStats, error) {
	var stats SpiritBonusStats
	bonuses := map[int]int{}
	dec := xml.NewDecoder(r)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, stats, fmt.Errorf("pipeline: reading dump: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "page" {
			continue
		}
		var page wikiPage
		if err := dec.DecodeElement(&page, &start); err != nil {
			return nil, stats, fmt.Errorf("pipeline: reading page: %w", err)
		}
		if page.NS != 0 || !strings.Contains(page.Text, "{{Clothing") {
			continue
		}
		fields := map[string]string{}
		for _, m := range infoboxField.FindAllStringSubmatch(page.Text, -1) {
			if _, seen := fields[m[1]]; !seen {
				fields[m[1]] = strings.TrimSpace(m[2])
			}
		}
		if fields["type"] != "Spirit" {
			continue
		}
		stats.Pages++

		n, err := strconv.Atoi(strings.TrimSpace(fields["wardrobe nr"]))
		if err != nil {
			stats.Rejected++
			continue
		}
		id := fandomID(scoring.Spirit, n)
		if len(known) > 0 && !known[id] {
			stats.Rejected++
			continue
		}
		bonus, ok := spiritBonus(page.Text)
		switch {
		case ok:
			stats.Parsed++
			bonuses[id] = bonus
		case spiritBonusHeading.FindStringIndex(page.Text) == nil:
			stats.NoSection++
		default:
			stats.NoLevels++
		}
	}
	return bonuses, stats, nil
}

func spiritBonus(text string) (int, bool) {
	loc := spiritBonusHeading.FindStringIndex(text)
	if loc == nil {
		return 0, false
	}
	body := text[loc[1]:]
	if end := wikiHeading.FindStringIndex(body); end != nil {
		body = body[:end[0]]
	}

	best, bonus := 0, 0
	for _, m := range spiritBonusLevel.FindAllStringSubmatch(body, -1) {
		level, err := strconv.Atoi(m[1])
		if err != nil || level < best {
			continue
		}
		v, err := strconv.Atoi(strings.ReplaceAll(m[2], ",", ""))
		if err != nil {
			continue
		}
		best, bonus = level, v
	}
	return bonus, best > 0
}

type SpiritBonusCheck struct {
	Agree      int
	Disagree   []int
	WikiOnly   []int
	PackedOnly []int
}

func CompareSpiritBonuses(wiki map[int]int, packed []Entry) SpiritBonusCheck {
	var check SpiritBonusCheck
	seen := make(map[int]bool, len(wiki))
	for _, e := range packed {
		if e.Item.Slot != scoring.Spirit || e.Item.FlatBonus == 0 {
			continue
		}
		seen[e.Item.ID] = true
		switch v, ok := wiki[e.Item.ID]; {
		case !ok:
			check.PackedOnly = append(check.PackedOnly, e.Item.ID)
		case v == e.Item.FlatBonus:
			check.Agree++
		default:
			check.Disagree = append(check.Disagree, e.Item.ID)
		}
	}
	for id := range wiki {
		if !seen[id] {
			check.WikiOnly = append(check.WikiOnly, id)
		}
	}
	slices.Sort(check.Disagree)
	slices.Sort(check.WikiOnly)
	slices.Sort(check.PackedOnly)
	return check
}
