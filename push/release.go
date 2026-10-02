package push

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/danielradosa/nikkibase/core/catalogue"
)

type Release struct {
	Version string            `json:"version"`
	Items   map[string]string `json:"items"`
	Stages  map[string]string `json:"stages"`
	Other   map[string]string `json:"other"`
	Scores  map[string]string `json:"scores,omitempty"`
	names   map[string]string
	suits   map[string]string
	order   []string
	labels  map[string]string
	stageAt []string
}

var versionName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func ReadRelease(webRoot string) (*Release, error) {
	data := filepath.Join(webRoot, "data")
	var index struct {
		Version string `json:"version"`
	}
	if err := readJSON(filepath.Join(data, "index.json"), &index); err != nil {
		return nil, err
	}
	if !versionName.MatchString(index.Version) {
		return nil, fmt.Errorf("data/index.json: bad version %q", index.Version)
	}
	dir := filepath.Join(data, index.Version)
	r := &Release{
		Version: index.Version,
		Items:   map[string]string{},
		Stages:  map[string]string{},
		Other:   map[string]string{},
		Scores:  map[string]string{},
		names:   map[string]string{},
		suits:   map[string]string{},
		labels:  map[string]string{},
	}

	var acquire struct {
		Items map[string]json.RawMessage `json:"items"`
	}
	if err := readJSON(filepath.Join(dir, "acquire.json"), &acquire); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	var items struct {
		Items [][]json.RawMessage `json:"items"`
	}
	if err := readJSON(filepath.Join(dir, "items.json"), &items); err != nil {
		return nil, err
	}
	for i, row := range items.Items {
		if len(row) < 3 {
			return nil, fmt.Errorf("items.json row %d: too short", i)
		}
		id, err := strconv.Atoi(string(row[0]))
		if err != nil {
			return nil, fmt.Errorf("items.json row %d: id: %w", i, err)
		}
		key := strconv.Itoa(id)
		var name, suit string
		_ = json.Unmarshal(row[1], &name)
		_ = json.Unmarshal(row[len(row)-1], &suit)
		rowHash, err := canon(mustMarshal(row))
		if err != nil {
			return nil, fmt.Errorf("items.json row %d: %w", i, err)
		}
		acquireHash := ""
		if raw, ok := acquire.Items[key]; ok {
			if acquireHash, err = canon(raw); err != nil {
				return nil, fmt.Errorf("acquire.json item %s: %w", key, err)
			}
		}
		r.Items[key] = short(rowHash + acquireHash)
		r.names[key] = name
		r.suits[key] = suit
		r.order = append(r.order, key)
	}

	bin, err := os.ReadFile(filepath.Join(dir, "items.bin"))
	if err != nil {
		return nil, err
	}
	cat, err := catalogue.Read(bin)
	if err != nil {
		return nil, fmt.Errorf("items.bin: %w", err)
	}
	for i, id := range cat.IDs {
		it := catalogue.Item{
			ID: id, Slot: cat.Slots[i], Position: cat.Positions[i], Group: cat.Groups[i],
			Tags: cat.Tags[cat.TagOffset[i]:cat.TagOffset[i+1]], FlatBonus: cat.FlatBonus[i],
		}
		copy(it.Attrs[:], cat.Attrs[i*5:i*5+5])
		copy(it.Stats[:], cat.Stats[i*5:i*5+5])
		r.Scores[strconv.Itoa(int(id))] = short(string(mustMarshal(it)))
	}

	var stages []map[string]json.RawMessage
	if err := readJSON(filepath.Join(dir, "stages.json"), &stages); err != nil {
		return nil, err
	}
	for i, st := range stages {
		var mode, name string
		_ = json.Unmarshal(st["mode"], &mode)
		_ = json.Unmarshal(st["name"], &name)
		if mode == "" || name == "" {
			return nil, fmt.Errorf("stages.json stage %d: no mode or name", i)
		}
		key := mode + "/" + name
		h, err := canon(mustMarshal(st))
		if err != nil {
			return nil, fmt.Errorf("stages.json stage %d: %w", i, err)
		}
		r.Stages[key] = short(h)
		r.labels[key] = mode + " " + name
		r.stageAt = append(r.stageAt, key)
	}

	for _, name := range []string{"tags.json", "positions.json"} {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		h, err := canon(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		r.Other[name] = short(h)
	}
	return r, nil
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func canon(raw []byte) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return "", err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func short(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

func LoadRelease(path string) (*Release, error) {
	var r Release
	if err := readJSON(path, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *Release) Save(path string) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return writeFile(path, b)
}

type Changes struct {
	Version     string
	NewItems    []string
	NewSuits    []string
	NewStages   []string
	FixedItems  int
	FixedStages int
	FixedOther  bool
}

func Compare(old, cur *Release) Changes {
	c := Changes{Version: cur.Version}
	seenSuit := map[string]bool{}
	for _, key := range cur.order {
		was, ok := old.Items[key]
		switch {
		case !ok:
			c.NewItems = append(c.NewItems, cur.names[key])
			if suit := suitName(cur.suits[key]); suit != "" && !seenSuit[suit] {
				seenSuit[suit] = true
				c.NewSuits = append(c.NewSuits, suit)
			}
		case was != cur.Items[key], old.Scores != nil && old.Scores[key] != cur.Scores[key]:
			c.FixedItems++
		}
	}
	for key := range old.Items {
		if _, ok := cur.Items[key]; !ok {
			c.FixedItems++
		}
	}
	for _, key := range cur.stageAt {
		was, ok := old.Stages[key]
		switch {
		case !ok:
			c.NewStages = append(c.NewStages, cur.labels[key])
		case was != cur.Stages[key]:
			c.FixedStages++
		}
	}
	for key := range old.Stages {
		if _, ok := cur.Stages[key]; !ok {
			c.FixedStages++
		}
	}
	for name, h := range cur.Other {
		if old.Other[name] != h {
			c.FixedOther = true
		}
	}
	for name := range old.Other {
		if _, ok := cur.Other[name]; !ok {
			c.FixedOther = true
		}
	}
	return c
}

func suitName(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, " ("); i > 0 && strings.HasSuffix(s, " Suit)") {
		s = s[:i]
	}
	return s
}

func (c Changes) Touches(topic string) bool {
	switch topic {
	case "items":
		return len(c.NewItems) > 0
	case "stages":
		return len(c.NewStages) > 0
	case "fixes":
		return c.FixedItems > 0 || c.FixedStages > 0 || c.FixedOther
	}
	return false
}

func (c Changes) Empty() bool {
	return !c.Touches("items") && !c.Touches("stages") && !c.Touches("fixes")
}

type Message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Tag   string `json:"tag"`
	URL   string `json:"url"`
}

func (c Changes) Message(topics []string) (Message, bool) {
	var lines []string
	news := false
	for _, topic := range Topics {
		if !has(topics, topic) || !c.Touches(topic) {
			continue
		}
		switch topic {
		case "items":
			lines = append(lines, c.itemsLine())
			news = true
		case "stages":
			lines = append(lines, c.stagesLine())
			news = true
		case "fixes":
			lines = append(lines, c.fixesLine())
		}
	}
	if len(lines) == 0 {
		return Message{}, false
	}
	title := "NikkiBase update"
	if news {
		title = "New in NikkiBase"
	}
	return Message{Title: title, Body: strings.Join(lines, " "), Tag: "nikkibase-data", URL: "/"}, true
}

func has(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func count(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return thousands(n) + " " + many
}

func thousands(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func and(list []string) string {
	switch len(list) {
	case 0:
		return ""
	case 1:
		return list[0]
	}
	return strings.Join(list[:len(list)-1], ", ") + " and " + list[len(list)-1]
}

func (c Changes) itemsLine() string {
	n := len(c.NewItems)
	if n == 1 {
		return "New item: " + c.NewItems[0] + "."
	}
	switch len(c.NewSuits) {
	case 0:
		return count(n, "new item", "new items") + "."
	case 1:
		return count(n, "new item", "new items") + ", including the " + c.NewSuits[0] + " suit."
	case 2:
		return count(n, "new item", "new items") + ", including the " + and(c.NewSuits) + " suits."
	}
	more := len(c.NewSuits) - 2
	return count(n, "new item", "new items") + ", including the " + c.NewSuits[0] + ", " + c.NewSuits[1] + " and " + count(more, "more suit", "more suits") + "."
}

func (c Changes) stagesLine() string {
	n := len(c.NewStages)
	switch {
	case n == 1:
		return "New stage: " + c.NewStages[0] + "."
	case n <= 3:
		return "New stages: " + and(c.NewStages) + "."
	}
	return count(n, "new stage", "new stages") + "."
}

func (c Changes) fixesLine() string {
	var parts []string
	if c.FixedItems > 0 {
		parts = append(parts, count(c.FixedItems, "item", "items"))
	}
	if c.FixedStages > 0 {
		parts = append(parts, count(c.FixedStages, "stage", "stages"))
	}
	if len(parts) == 0 {
		return "Fixes to the data."
	}
	return "Fixes to " + and(parts) + "."
}
