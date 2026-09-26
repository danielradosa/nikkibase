package pipeline

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
)

type MappedSource struct {
	Kind  string `json:"k"`
	Text  string `json:"t"`
	Past  bool   `json:"past"`
	Basis string `json:"basis"`
}

type AcquisitionMap struct {
	Note         string                  `json:"note"`
	Codes        map[string]MappedSource `json:"codes"`
	SignIn       MappedSource            `json:"signIn"`
	Texts        map[string]MappedSource `json:"texts"`
	DreamWeavers map[string]string       `json:"dreamWeavers"`
}

var (
	packedSignIn = regexp.MustCompile(`^签到·\d+月$`)
	packedStage  = regexp.MustCompile(`^(II-|III-)?(\d+)-(支)?(\d+)(少|公)$`)
	packedDream  = regexp.MustCompile(`^(\D+?)[\d-]*$`)
)

const limitedEvent = "Limited event"

func ReadAcquisitionMap(b []byte) (*AcquisitionMap, error) {
	var m AcquisitionMap
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("acquisition map: %w", err)
	}
	check := func(where string, s MappedSource) error {
		switch {
		case !slices.Contains(AcquisitionKinds, s.Kind):
			return fmt.Errorf("acquisition map: %s has unknown kind %q", where, s.Kind)
		case strings.TrimSpace(s.Text) == "" || hasHan(s.Text):
			return fmt.Errorf("acquisition map: %s needs an English line, not %q", where, s.Text)
		case s.Kind == "event" && s.Text != limitedEvent && s.Basis == "":
			return fmt.Errorf("acquisition map: %s names the event %q without the basis for that name", where, s.Text)
		}
		return nil
	}
	if err := check("signIn", m.SignIn); err != nil {
		return nil, err
	}
	for _, set := range []map[string]MappedSource{m.Codes, m.Texts} {
		for _, key := range sortedStrings(set) {
			if err := check(key, set[key]); err != nil {
				return nil, err
			}
		}
	}
	for key, name := range m.DreamWeavers {
		if hasHan(name) {
			return nil, fmt.Errorf("acquisition map: Dream Weaver %s needs an English name, not %q", key, name)
		}
	}
	return &m, nil
}

func sortedStrings[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (m *AcquisitionMap) Translate(src map[int][]PackedSource, cat AcquisitionCatalogue) (map[int][]Acquisition, error) {
	out := map[int][]Acquisition{}
	unknown := map[string]bool{}
	for _, id := range sortedIDs(src) {
		if _, ok := cat.Names[id]; !ok {
			continue
		}
		var list []Acquisition
		for _, s := range src[id] {
			a, ok := m.translate(id, s, cat)
			if !ok {
				unknown[s.Kind+" "+s.Value] = true
				continue
			}
			a.CN = true
			list = appendAcquisition(list, a)
		}
		if len(list) > 0 {
			out[id] = list
		}
	}
	if len(unknown) > 0 {
		keys := sortedStrings(unknown)
		return nil, fmt.Errorf("acquisition: %d sources in the packed table have no English line in data/acquisition-cn.json: %s",
			len(keys), truncate(keys))
	}
	return out, nil
}

func (m *AcquisitionMap) translate(id int, s PackedSource, cat AcquisitionCatalogue) (Acquisition, bool) {
	switch s.Kind {
	case "code":
		mapped, ok := m.Codes[s.Value]
		if !ok && packedSignIn.MatchString(s.Value) {
			mapped, ok = m.SignIn, true
		}
		if !ok {
			return Acquisition{}, false
		}
		return m.mapped(id, mapped, cat), true
	case "text":
		if p := packedStage.FindStringSubmatch(s.Value); p != nil {
			volume := 1
			if p[1] != "" {
				volume = len(p[1]) - 1
			}
			level := "Maiden"
			if p[5] == "公" {
				level = "Princess"
			}
			return storyAcquisition(storyName(volume, p[2], p[4], p[3] != ""), level, cat), true
		}
		mapped, ok := m.Texts[s.Value]
		if !ok {
			return Acquisition{}, false
		}
		return m.mapped(id, mapped, cat), true
	case "suit":
		return Acquisition{Kind: "suit", Text: giftBoxLine(cat.Suits[id])}, true
	case "customize", "evolve":
		a := Acquisition{Kind: s.Kind, Text: "Customization"}
		verb, qty := "Customize: ", 1
		if s.Kind == "evolve" {
			a.Text, verb, qty = "Evolution", "Evolve: ", 0
		}
		if _, ok := cat.Names[s.ID]; ok {
			a.From = []Ingredient{{ID: s.ID, Qty: qty}}
			if name := cat.shown(s.ID); !hasHan(name) {
				a.Text = verb + name
			}
		}
		return a, true
	case "dream":
		p := packedDream.FindStringSubmatch(s.Value)
		if p == nil {
			return Acquisition{}, false
		}
		name, ok := m.DreamWeavers[p[1]]
		if !ok {
			return Acquisition{}, false
		}
		a := Acquisition{Kind: "dream", Text: "Dream Weaver"}
		if name != "" {
			a.Text += ": " + name
		}
		return a, true
	}
	return Acquisition{}, false
}

var basisCount = regexp.MustCompile(`^named by the wiki on (\d+) of the (\d+) items both sources cover$`)

func (m *AcquisitionMap) CheckBasis(src map[int][]PackedSource, wiki map[int][]Acquisition) error {
	var bad []string
	for _, set := range []struct {
		kind  string
		lines map[string]MappedSource
	}{{"code", m.Codes}, {"text", m.Texts}} {
		for _, key := range sortedStrings(set.lines) {
			s := set.lines[key]
			if s.Basis == "" {
				continue
			}
			named, covered := basisAgreement(set.kind, key, s, src, wiki)
			want := fmt.Sprintf("named by the wiki on %d of the %d items both sources cover", named, covered)
			switch {
			case !basisCount.MatchString(s.Basis):
				bad = append(bad, fmt.Sprintf("%s gives the basis %q, not a count", key, s.Basis))
			case s.Basis != want:
				bad = append(bad, fmt.Sprintf("%s says %q, and the sources give %q", key, s.Basis, want))
			case s.Kind == "event" && 4*named < 3*covered, 2*named <= covered:
				bad = append(bad, fmt.Sprintf("%s names %q, which the wiki gives on only %d of %d items", key, s.Text, named, covered))
			}
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("acquisition: %d basis lines in data/acquisition-cn.json disagree with the sources: %s", len(bad), truncate(bad))
	}
	return nil
}

func basisAgreement(kind, key string, s MappedSource, src map[int][]PackedSource, wiki map[int][]Acquisition) (named, covered int) {
	for id, sources := range src {
		if len(wiki[id]) == 0 || !slices.ContainsFunc(sources, func(p PackedSource) bool { return p.Kind == kind && p.Value == key }) {
			continue
		}
		covered++
		if slices.ContainsFunc(wiki[id], func(a Acquisition) bool {
			return a.Kind == s.Kind && (a.Text == s.Text || strings.HasPrefix(a.Text, s.Text+" · "))
		}) {
			named++
		}
	}
	return named, covered
}

func (m *AcquisitionMap) mapped(id int, s MappedSource, cat AcquisitionCatalogue) Acquisition {
	a := Acquisition{Kind: s.Kind, Text: s.Text, Past: s.Past}
	if s.Kind == "suit" {
		a.Text = giftBoxLine(cat.Suits[id])
	}
	return a
}

func giftBoxLine(suit string) string {
	if suit == "" || hasHan(suit) {
		return "Styling Gift Box"
	}
	return "Styling Gift Box for completing " + suit
}
