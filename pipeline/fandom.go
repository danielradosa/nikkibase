package pipeline

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/danielradosa/nikkibase/core/scoring"
)

var fandomSlots = map[string]scoring.Slot{
	"Hair": scoring.Hair, "Dress": scoring.Dress, "Coat": scoring.Coat,
	"Top": scoring.Top, "Bottom": scoring.Bottom, "Hosiery": scoring.Hosiery,
	"Shoes": scoring.Shoes, "Makeup": scoring.Makeup, "Spirit": scoring.Spirit,
}

var attributeSides = [5]struct{ first, second []string }{
	{[]string{"gorgeous", "g"}, []string{"simple", "si", "s"}},
	{[]string{"elegant", "elegance", "e"}, []string{"lively", "active", "l"}},
	{[]string{"mature", "m"}, []string{"cute", "cu", "c"}},
	{[]string{"sexy", "se", "s"}, []string{"pure", "p"}},
	{[]string{"warm", "w"}, []string{"cool", "co", "c"}},
}

var (
	attributesTemplate = regexp.MustCompile(`\{\{Attributes\|([^}]+)\}\}`)
	infoboxField       = regexp.MustCompile(`(?m)^\s*\|\s*([a-z0-9 ]+?)\s*=\s*(.*)$`)
	styleTag           = regexp.MustCompile(`\{\{S\|([^}|]+)`)
)

type wikiPage struct {
	Title string `xml:"title"`
	NS    int    `xml:"ns"`
	Text  string `xml:"revision>text"`
}

type FandomStats struct {
	Pages        int
	Parsed       int
	NoGrades     int
	UnknownAny   int
	UnknownStyle int
}

func ParseFandomDump(r io.Reader, known map[int]bool) ([]Entry, FandomStats, error) {
	var (
		entries []Entry
		stats   FandomStats
	)
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
		stats.Pages++

		entry, why := fandomEntry(page, &stats)
		switch why {
		case "":
		case "no grades":
			stats.NoGrades++
			continue
		default:
			stats.UnknownAny++
			continue
		}
		if len(known) > 0 && !known[entry.Item.ID] {
			stats.UnknownAny++
			continue
		}
		stats.Parsed++
		entries = append(entries, entry)
	}
	return entries, stats, nil
}

func fandomEntry(page wikiPage, stats *FandomStats) (Entry, string) {
	fields := map[string]string{}
	for _, m := range infoboxField.FindAllStringSubmatch(page.Text, -1) {
		if _, seen := fields[m[1]]; !seen {
			fields[m[1]] = strings.TrimSpace(m[2])
		}
	}
	slot, position, ok := fandomSlot(fields["type"])
	if !ok {
		return Entry{}, "unknown type"
	}
	n, err := strconv.Atoi(strings.TrimSpace(fields["wardrobe nr"]))
	if err != nil {
		return Entry{}, "bad wardrobe number"
	}
	grades := attributesTemplate.FindStringSubmatch(page.Text)
	if grades == nil {
		return Entry{}, "no grades"
	}
	parts := strings.Split(grades[1], "|")
	if len(parts) < 10 {
		return Entry{}, "short attributes"
	}

	it := scoring.Item{ID: fandomID(slot, n), Slot: slot}
	var letters [5]string
	for p := range 5 {
		side, grade := strings.TrimSpace(parts[p*2]), strings.TrimSpace(parts[p*2+1])
		code, ok := attributeCode(p, side)
		if !ok {
			return Entry{}, "unknown attribute side"
		}
		it.Attrs[p], it.Stats[p], letters[p] = int8(code), Stat(grade, slot), grade
	}
	for _, m := range styleTag.FindAllStringSubmatch(fields["style"], -1) {
		style, ok := StyleFromCode(m[1])
		if !ok {
			stats.UnknownStyle++
			continue
		}
		id, ok := TagID(style)
		if !ok {
			stats.UnknownStyle++
			continue
		}
		it.Tags = append(it.Tags, id)
	}
	return Entry{Item: it, Name: page.Title, Position: position, Grades: letters}, ""
}

func fandomSlot(kind string) (scoring.Slot, string, bool) {
	if slot, ok := fandomSlots[kind]; ok {
		return slot, strings.ToLower(kind), true
	}
	base, sub, found := strings.Cut(kind, ",")
	if !found {
		return 0, "", false
	}
	sub = strings.ToLower(strings.TrimSpace(sub))
	switch strings.TrimSpace(base) {
	case "Accessory":
		return scoring.Accessory, "accessory:" + sub, true
	case "Hosiery":
		return scoring.Hosiery, "hosiery:" + sub, true
	}
	return 0, "", false
}

func fandomID(slot scoring.Slot, n int) int {
	switch {
	case slot == scoring.Spirit:
		return 880000 + n
	case slot == scoring.Accessory && n >= 10000:
		return 170000 + n
	default:
		return idPrefix[slot]*10000 + n
	}
}

func attributeCode(pair int, side string) (int, bool) {
	side = strings.ToLower(strings.TrimSpace(side))
	for _, name := range attributeSides[pair].first {
		if side == name {
			return pair * 2, true
		}
	}
	for _, name := range attributeSides[pair].second {
		if side == name {
			return pair*2 + 1, true
		}
	}
	return 0, false
}
