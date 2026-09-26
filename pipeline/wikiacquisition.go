package pipeline

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/danielradosa/nikkibase/core/scoring"
)

type WikiAcquisition struct {
	Items        map[int][]Acquisition
	Suits        map[int][]Acquisition
	SuitOf       map[int]string
	ChineseSuits map[string]string
	Packs        map[string]bool
	Unplaced     []SuitPart
}

type SuitPart struct {
	Suit string
	IDs  []int
}

type WikiAcquisitionStats struct {
	Pages        int
	WithEntries  int
	Moved        int
	Dropped      int
	SuitPages    int
	SuitItems    int
	SharedParts  int
	Unresolved   int
	UnknownParts int
	UnknownUnits int
	Unclassified int
}

type acqRawPage struct {
	Title    string `xml:"title"`
	NS       int    `xml:"ns"`
	Redirect struct {
		Title string `xml:"title,attr"`
	} `xml:"redirect"`
	Text string `xml:"revision>text"`
}

type acqPage struct {
	title    string
	id       int
	obtain   string
	suit     string
	intro    string
	sections map[string]string
}

type acqSuit struct {
	title   string
	obtain  string
	pack    bool
	chinese string
	parts   []suitPart
	reward  []string
}

type suitPart struct {
	name, kind string
}

var (
	acqInfoboxField  = regexp.MustCompile(`(?m)^[ \t]*\|[ \t]*([a-z0-9 ]+?)[ \t]*=[ \t]*(.*)$`)
	wikiComment      = regexp.MustCompile(`(?s)<!--.*?-->`)
	wikiLink         = regexp.MustCompile(`\[\[:?([^\]|]*)(?:\|([^\]]*))?\]\]`)
	wikiTag          = regexp.MustCompile(`<[^>]*>`)
	wikiTemplate     = regexp.MustCompile(`\{\{[^{}]*\}\}`)
	wikiBreak        = regexp.MustCompile(`(?i)<br\s*/?>|\n`)
	wikiSection      = regexp.MustCompile(`(?m)^[ \t]*=+[ \t]*(.*?)[ \t]*=+[ \t]*$`)
	wikiStages       = regexp.MustCompile(`(?i)\{\{\s*stages\s*\|([^}|]*)[^}]*\}\}`)
	wikiStageArg     = regexp.MustCompile(`(?i)^(?:v([123])\s*:\s*|(i{1,3})\s*[.:]\s*)?(\d+)-(s)?(\d+)$`)
	wikiLevel        = regexp.MustCompile(`(?i)\b(maiden|princess)\b`)
	wikiStagePage    = regexp.MustCompile(`^(V[123]:\s*\d+-S?\d+)\b`)
	wikiVolume       = regexp.MustCompile(`^V1:\s*`)
	wikiPlainStage   = regexp.MustCompile(`(?i)^(?:story\s+(?:drop\s+from\s+)?)?(?:volume\s+([123]),?\s*)?(?:stage\s+)?(\d+-s?\d+)\s*\(?(?:maiden|princess)?\)?$`)
	wikiIconItem     = regexp.MustCompile(`\{\{IconItem\|([^}|]+)([^}]*)\}\}`)
	wikiQuantity     = regexp.MustCompile(`quantity\s*=\s*(\d+)`)
	wikiAmount       = regexp.MustCompile(`([\d,]+)\s*\{\{\s*(Currency|Items)\s*\|([^}|]+)[^}]*\}\}`)
	wikiPrice        = regexp.MustCompile(`\bfor\s+([\d,]+)\s*\{\{\s*(Currency|Items)\s*\|([^}|]+)[^}]*\}\}`)
	wikiPriceMore    = regexp.MustCompile(`^\s*(?:,|and|\+)\s*([\d,]+)\s*\{\{\s*(Currency|Items)\s*\|([^}|]+)[^}]*\}\}`)
	wikiPriceShop    = regexp.MustCompile(`^\s*(?:in|at|from)\s+(?:the\s+)?\[\[([^\]|]*)(?:\|([^\]]*))?\]\]`)
	wikiTense        = regexp.MustCompile(`(?i)\b(could|was|were|can|now)\b`)
	wikiCustomizes   = regexp.MustCompile(`(?i)customi[sz]ation(?:\]\])?\s+(?:of|for)\s+(?:the\s+)?(?:\[\[([^\]|]+)(?:\|[^\]]*)?\]\]|([A-Z][^.\[\]{}\n]*?)\s*(?:\.|$))`)
	wikiGiftBoxSuit  = regexp.MustCompile(`(?i)styling gift box\]\]\s+(?:after|for|by|upon)\s+completing\s+(?:the\s+)?(?:\[\[[^\]]*\]\]\s+suit\s+)?(?:suit\s+)?\[\[([^\]|]*)(?:\|([^\]]*))?\]\]`)
	wikiGluedEvent   = regexp.MustCompile(`(?i)\]\](events?\b)`)
	wikiFriends      = regexp.MustCompile(`(?i)^(?:obtain|gain|have|make)\s+(\d+)\s+friends$`)
	wikiLastingPack  = regexp.MustCompile(`(?i)\bvip\b|privilege|first recharge|monthly card`)
	wikiListEntry    = regexp.MustCompile(`\{\{WIL\|\s*(\d+)\s*\|([^|}]+)`)
	wikiQualifier    = regexp.MustCompile(`\(\s*([^()]+?)\s*\)\s*$`)
	wikiUnitLine     = regexp.MustCompile(`(?m)^\|\s*([A-Za-z]+)\s*=\s*\[\[File:([^\]]*)\]\]`)
	wikiPixels       = regexp.MustCompile(`^\d+(x\d+)?px$`)
	wikiSuitPart     = regexp.MustCompile(`\{\{Suit Part\|([^|}]+)([^}]*)`)
	wikiPartType     = regexp.MustCompile(`\|\s*type\s*=\s*([^|}]*)`)
	wikiEventWord    = regexp.MustCompile(`(?i)\bevents?\b`)
	wikiCustomTarget = regexp.MustCompile(`(?m)^\*\s*\{\{IconItem\|([^}|]+)[^}]*\}\}\s*:?(.*)$`)
)

type fixedSource struct {
	kind, text string
}

var fixedSources = map[string]fixedSource{
	"crafting":                      {"craft", "Crafting"},
	"recipe crafting":               {"craft", "Crafting"},
	"craft":                         {"craft", "Crafting"},
	"cafting":                       {"craft", "Crafting"},
	"craftin":                       {"craft", "Crafting"},
	"shadow workshop":               {"craft", "Shadow Workshop"},
	"fantasy workshop":              {"craft", "Fantasy Workshop"},
	"lost casket":                   {"craft", "Lost Casket"},
	"customization":                 {"customize", "Customization"},
	"customisation":                 {"customize", "Customization"},
	"evolution":                     {"evolve", "Evolution"},
	"reconstruction":                {"reconstruct", "Reconstruction"},
	"clothes store":                 {"store", "Clothes Store"},
	"clothing store":                {"store", "Clothes Store"},
	"clothes shop":                  {"store", "Clothes Store"},
	"cothes store":                  {"store", "Clothes Store"},
	"secret shop":                   {"store", "Secret Shop"},
	"boutique":                      {"store", "Boutique"},
	"user's shop":                   {"store", "User's Shop"},
	"users shop":                    {"store", "User's Shop"},
	"store of starlight":            {"store", "Store of Starlight"},
	"association store":             {"association", "Association Store"},
	"association shop":              {"association", "Association Store"},
	"association requests":          {"association", "Association requests"},
	"complete association requests": {"association", "Association requests"},
	"pavilion of mystery":           {"pavilion", "Pavilion of Mystery"},
	"pavillion of mystery":          {"pavilion", "Pavilion of Mystery"},
	"pavillon of mystery":           {"pavilion", "Pavilion of Mystery"},
	"pavilion of fantasy":           {"pavilion", "Pavilion of Fantasy"},
	"crystal garden":                {"pavilion", "Crystal Garden"},
	"room of cinderella":            {"pavilion", "Room of Cinderella"},
	"porch of misty":                {"pavilion", "Porch of Misty"},
	"pavilion of jade":              {"pavilion", "Pavilion of Jade"},
	"corridor of clock":             {"pavilion", "Corridor of Clock"},
	"pavilion of time":              {"pavilion", "Pavilion of Time"},
	"tower of zen":                  {"pavilion", "Tower of Zen"},
	"villa of cloud":                {"pavilion", "Villa of Cloud"},
	"time yard":                     {"pavilion", "Time Yard"},
	"time yard pavilion":            {"pavilion", "Time Yard"},
	"pavilion of glaze":             {"pavilion", "Pavilion of Glaze"},
	"pavillion of glaze":            {"pavilion", "Pavilion of Glaze"},
	"wish gate":                     {"pavilion", "Wish Gate"},
	"recharge":                      {"recharge", "Recharge"},
	"cumulative recharge":           {"recharge", "Cumulative Recharge"},
	"lucky bags":                    {"recharge", "Lucky Bags"},
	"lucky bag":                     {"recharge", "Lucky Bags"},
	"abyssal island":                {"recharge", "Abyssal Island"},
	"monthly card":                  {"recharge", "Monthly Card"},
	"one-dollar sale":               {"recharge", "One-Dollar Sale"},
	"one dollar sale":               {"recharge", "One-Dollar Sale"},
	"monthly sign-in":               {"signin", "Monthly Sign-In"},
	"monthly sign-in reward":        {"signin", "Monthly Sign-In"},
	"monthly sign in":               {"signin", "Monthly Sign-In"},
	"log-in event":                  {"signin", "Log-in Event"},
	"login event":                   {"signin", "Log-in Event"},
	"log in event":                  {"signin", "Log-in Event"},
	"sign-in event":                 {"signin", "Log-in Event"},
	"styling gift box":              {"suit", "Styling Gift Box"},
	"suit completion reward":        {"suit", "Styling Gift Box"},
	"mailbox gift":                  {"gift", "Mailbox Gift"},
	"mailbox":                       {"gift", "Mailbox Gift"},
	"redeem code":                   {"gift", "Redeem Code"},
	"achievement":                   {"achievement", "Achievement"},
	"achievements":                  {"achievement", "Achievement"},
	"complete achievement":          {"achievement", "Achievement"},
	"unlocked at start":             {"other", "Available from the start"},
	"unlocked from start":           {"other", "Available from the start"},
	"default":                       {"other", "Available from the start"},
	"extra stage bonus":             {"other", "Extra Stage Bonus"},
	"time diary":                    {"other", "Time Diary"},
	"friends list":                  {"achievement", "Friend count reward"},
}

var pastKinds = map[string]bool{"event": true, "recharge": true, "signin": true, "gift": true}

func pastSource(kind, text string) bool {
	return pastKinds[kind] && !(kind == "recharge" && wikiLastingPack.MatchString(text))
}

func ParseFandomAcquisition(r io.Reader, known map[int]bool, corrections *IDCorrections, cat AcquisitionCatalogue) (WikiAcquisition, WikiAcquisitionStats, error) {
	var stats WikiAcquisitionStats
	ctx := &acqContext{
		cat:       cat,
		units:     map[string]map[string]string{},
		titles:    map[string]int{},
		redirects: map[string]string{},
		listed:    map[string]int{},
		stats:     &stats,
	}
	var pages []acqPage
	var suits []acqSuit
	dec := xml.NewDecoder(r)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return WikiAcquisition{}, stats, fmt.Errorf("pipeline: reading dump: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "page" {
			continue
		}
		var page acqRawPage
		if err := dec.DecodeElement(&page, &start); err != nil {
			return WikiAcquisition{}, stats, fmt.Errorf("pipeline: reading page: %w", err)
		}
		switch {
		case page.Redirect.Title != "":
			ctx.redirects[page.Title] = page.Redirect.Title
		case page.NS == 10 && (page.Title == "Template:Currency" || page.Title == "Template:Items"):
			ctx.units[strings.TrimPrefix(page.Title, "Template:")] = wikiUnits(page.Text)
		case page.NS != 0:
		case listSlotOK(page.Title):
			ctx.readList(page)
		case strings.Contains(page.Text, "{{Clothing"):
			if p, ok := acqItemPage(page, known, corrections, cat, &stats); ok {
				pages = append(pages, p)
			}
		case strings.Contains(page.Text, "{{Suit Infobox"):
			suits = append(suits, acqSuitPage(page))
		}
	}

	byID := map[int]int{}
	kept := pages[:0]
	for _, p := range pages {
		if at, dup := byID[p.id]; dup {
			stats.Dropped++
			if !sameGarment(kept[at].title, cat.Names[p.id]) && sameGarment(p.title, cat.Names[p.id]) {
				delete(ctx.titles, kept[at].title)
				kept[at] = p
				ctx.titles[p.title] = p.id
			}
			continue
		}
		byID[p.id] = len(kept)
		kept = append(kept, p)
		ctx.titles[p.title] = p.id
	}
	pages = kept
	ctx.pages = make(map[int]acqPage, len(pages))
	for _, p := range pages {
		ctx.pages[p.id] = p
	}
	ctx.indexCatalogue()
	sort.Slice(suits, func(i, j int) bool { return suits[i].title < suits[j].title })
	ctx.suits = ctx.knownSuits(pages, suits)
	ctx.customs = map[string]map[string][]Cost{}
	ctx.customizers = map[string][]string{}
	for _, p := range pages {
		if body, ok := p.sections["customization"]; ok {
			ctx.customs[p.title] = ctx.customizationCosts(body)
			for key := range ctx.customs[p.title] {
				ctx.customizers[key] = append(ctx.customizers[key], p.title)
			}
		}
	}

	out := WikiAcquisition{Items: map[int][]Acquisition{}, Suits: map[int][]Acquisition{}, SuitOf: ctx.suits,
		ChineseSuits: chineseSuits(suits), Packs: packTitles(suits), Unplaced: ctx.unplaced(suits)}
	stats.Pages = len(pages)
	for _, p := range pages {
		if list := ctx.pageAcquisition(p); len(list) > 0 {
			out.Items[p.id] = list
			stats.WithEntries++
		}
	}
	stats.SuitPages = len(suits)
	for _, s := range suits {
		ctx.suitAcquisition(s, out.Suits)
	}
	stats.SuitItems = len(out.Suits)
	return out, stats, nil
}

func acqItemPage(page acqRawPage, known map[int]bool, corrections *IDCorrections, cat AcquisitionCatalogue, stats *WikiAcquisitionStats) (acqPage, bool) {
	text := page.Text
	begin := strings.Index(text, "{{Clothing")
	end := strings.Index(text[begin:], "\n}}")
	if end < 0 {
		return acqPage{}, false
	}
	box, body := text[begin:begin+end], text[begin+end+3:]
	fields := infobox(box)
	slot, _, ok := fandomSlot(fields["type"])
	if !ok {
		return acqPage{}, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(fields["wardrobe nr"]))
	if err != nil {
		return acqPage{}, false
	}
	id := fandomID(slot, n)
	if len(known) > 0 && !known[id] {
		return acqPage{}, false
	}
	if _, ok := cat.Names[id]; !ok {
		return acqPage{}, false
	}
	title := page.Title
	if corrections != nil {
		if corrections.misnumbered(id, title) {
			id = corrections.RealID[id]
			if _, ok := cat.Names[id]; !ok {
				stats.Dropped++
				return acqPage{}, false
			}
			stats.Moved++
		}
		title = corrections.nameOf(id, title)
	}
	intro := body
	if i := wikiSection.FindStringIndex(body); i != nil {
		intro = body[:i[0]]
	}
	return acqPage{
		title:    title,
		id:       id,
		obtain:   fields["how to obtain"],
		suit:     fields["part of suit"],
		intro:    intro,
		sections: wikiSections(body),
	}, true
}

func acqSuitPage(page acqRawPage) acqSuit {
	s := acqSuit{title: page.Title}
	text := page.Text
	begin := strings.Index(text, "{{Suit Infobox")
	box := text[begin:]
	if end := strings.Index(box, "\n}}"); end >= 0 {
		box = box[:end]
	}
	for _, m := range acqInfoboxField.FindAllStringSubmatch(box, -1) {
		switch m[1] {
		case "how to obtain":
			if s.obtain == "" {
				s.obtain = strings.TrimSpace(m[2])
			}
		case "type":
			s.pack = strings.EqualFold(cleanWiki(m[2]), "Pack")
		case "cnwiki":
			if s.chinese == "" {
				s.chinese = cleanWiki(m[2])
			}
		case "reward":
			if i := strings.Index(m[2], "{{Gift Box|"); i >= 0 && s.reward == nil {
				args := templateArgs(m[2], i)
				if len(args) > 1 {
					for _, name := range strings.Split(args[1], ";") {
						if name = strings.TrimSpace(wikiTemplate.ReplaceAllString(name, "")); name != "" && !isNumber(name) {
							s.reward = append(s.reward, name)
						}
					}
				}
			}
		}
	}
	for _, m := range wikiSuitPart.FindAllStringSubmatch(text, -1) {
		part := suitPart{name: strings.TrimSpace(m[1])}
		if t := wikiPartType.FindStringSubmatch(m[2]); t != nil {
			part.kind = strings.TrimSpace(t[1])
		}
		s.parts = append(s.parts, part)
	}
	return s
}

func infobox(box string) map[string]string {
	fields := map[string]string{}
	for _, m := range acqInfoboxField.FindAllStringSubmatch(box, -1) {
		if _, seen := fields[m[1]]; !seen {
			fields[m[1]] = strings.TrimSpace(m[2])
		}
	}
	return fields
}

func isNumber(s string) bool {
	_, err := strconv.Atoi(strings.ReplaceAll(strings.TrimSpace(s), ",", ""))
	return err == nil
}

func wikiSections(body string) map[string]string {
	out := map[string]string{}
	heads := wikiSection.FindAllStringSubmatchIndex(body, -1)
	for i, h := range heads {
		name := strings.ToLower(cleanWiki(body[h[2]:h[3]]))
		name = strings.TrimSpace(strings.TrimRight(name, ":"))
		end := len(body)
		if i+1 < len(heads) {
			end = heads[i+1][0]
		}
		if _, seen := out[name]; !seen {
			out[name] = body[h[1]:end]
		}
	}
	return out
}

func wikiUnits(text string) map[string]string {
	out := map[string]string{}
	for _, m := range wikiUnitLine.FindAllStringSubmatch(text, -1) {
		if _, seen := out[m[1]]; seen {
			continue
		}
		parts := strings.Split(m[2], "|")
		for i := len(parts) - 1; i > 0; i-- {
			p := strings.TrimSpace(parts[i])
			if p == "" || strings.Contains(p, "=") || wikiPixels.MatchString(p) {
				continue
			}
			out[m[1]] = p
			break
		}
	}
	return out
}

func cleanWiki(s string) string {
	s = wikiComment.ReplaceAllString(s, "")
	s = wikiLink.ReplaceAllStringFunc(s, func(l string) string {
		m := wikiLink.FindStringSubmatch(l)
		if m[2] != "" {
			return m[2]
		}
		if i := strings.LastIndex(m[1], "#"); i > 0 {
			return m[1][:i]
		}
		return m[1]
	})
	for prev := ""; prev != s; {
		prev = s
		s = wikiTemplate.ReplaceAllString(s, "")
	}
	s = wikiTag.ReplaceAllString(s, "")
	s = strings.NewReplacer("{", "", "}", "").Replace(s)
	s = strings.ReplaceAll(s, "'''", "")
	s = strings.ReplaceAll(s, "''", "")
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(strings.TrimRight(s, ".:;, "))
}

func templateArgs(text string, at int) []string {
	depth, i := 0, at
	var args []string
	var cur strings.Builder
	links := 0
	for i < len(text) {
		switch {
		case strings.HasPrefix(text[i:], "{{"):
			depth++
			if depth > 1 {
				cur.WriteString("{{")
			}
			i += 2
			continue
		case strings.HasPrefix(text[i:], "}}"):
			depth--
			if depth == 0 {
				return append(args, cur.String())
			}
			cur.WriteString("}}")
			i += 2
			continue
		case strings.HasPrefix(text[i:], "[["):
			links++
			cur.WriteString("[[")
			i += 2
			continue
		case strings.HasPrefix(text[i:], "]]"):
			links--
			cur.WriteString("]]")
			i += 2
			continue
		case text[i] == '|' && depth == 1 && links <= 0:
			args = append(args, cur.String())
			cur.Reset()
			i++
			continue
		}
		cur.WriteByte(text[i])
		i++
	}
	return append(args, cur.String())
}

func namedArgs(args []string) map[string]string {
	out := map[string]string{}
	for _, a := range args[1:] {
		k, v, ok := strings.Cut(a, "=")
		if !ok {
			continue
		}
		k = strings.ToLower(strings.TrimSpace(k))
		k = strings.ReplaceAll(k, " ", "_")
		if _, seen := out[k]; !seen {
			out[k] = strings.TrimSpace(v)
		}
	}
	return out
}

type acqContext struct {
	cat         AcquisitionCatalogue
	units       map[string]map[string]string
	titles      map[string]int
	redirects   map[string]string
	pages       map[int]acqPage
	listed      map[string]int
	suits       map[int]string
	lower       map[string][]int
	norm        map[string][]int
	customs     map[string]map[string][]Cost
	customizers map[string][]string
	stats       *WikiAcquisitionStats
}

func (c *acqContext) indexCatalogue() {
	c.lower = map[string][]int{}
	c.norm = map[string][]int{}
	for _, id := range sortedIDs(c.cat.Names) {
		lower, norm := strings.ToLower(strings.TrimSpace(c.cat.Names[id])), normalizeGarment(c.cat.Names[id])
		c.lower[lower] = append(c.lower[lower], id)
		c.norm[norm] = append(c.norm[norm], id)
	}
}

func (c *acqContext) resolve(name, suit string) int {
	return c.lookup(name, suit, false)
}

func (c *acqContext) resolvePart(name, suit string) int {
	return c.lookup(name, suit, true)
}

func (c *acqContext) lookup(name, suit string, part bool) int {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0
	}
	for _, t := range []string{name, upperFirst(name)} {
		if id, ok := c.titles[t]; ok {
			return id
		}
		if target, ok := c.redirects[t]; ok {
			if id, ok := c.titles[target]; ok {
				return id
			}
		}
	}
	if id := c.pick(c.lower[strings.ToLower(name)], suit, part); id != 0 {
		return id
	}
	if id := c.pick(c.normalized(name), suit, part); id != 0 {
		return id
	}
	if spaced := NormalizeName(name); spaced != name {
		if id := c.pick(c.lower[strings.ToLower(spaced)], suit, part); id != 0 {
			return id
		}
		if id := c.pick(c.normalized(spaced), suit, part); id != 0 {
			return id
		}
	}
	if id := pickQualified(c.norm[normalizeGarment(name)], name); id != 0 {
		return id
	}
	for _, t := range []string{name, upperFirst(name)} {
		if id := c.listed[t]; id != 0 {
			return id
		}
	}
	return 0
}

var wikiLists = map[string]scoring.Slot{
	"Hairs": scoring.Hair, "Dresses": scoring.Dress, "Coats": scoring.Coat, "Tops": scoring.Top,
	"Bottoms": scoring.Bottom, "Hosiery": scoring.Hosiery, "Shoes": scoring.Shoes, "Makeup": scoring.Makeup,
	"Background Items": scoring.Accessory, "Bracelets": scoring.Accessory, "Brooches": scoring.Accessory,
	"Earrings": scoring.Accessory, "Ears": scoring.Accessory, "Face Accessories": scoring.Accessory,
	"Foreground Items": scoring.Accessory, "Ground Items": scoring.Accessory, "Hair Ornaments": scoring.Accessory,
	"Hairpins": scoring.Accessory, "Handheld": scoring.Accessory, "Head Ornaments": scoring.Accessory,
	"Neckwear": scoring.Accessory, "Skin": scoring.Accessory, "Tails": scoring.Accessory,
	"Tattoos": scoring.Accessory, "Veils": scoring.Accessory, "Waist": scoring.Accessory, "Wings": scoring.Accessory,
}

func listSlot(title string) (scoring.Slot, bool) {
	base, _, _ := strings.Cut(title, "/")
	slot, ok := wikiLists[base]
	return slot, ok
}

func listSlotOK(title string) bool {
	_, ok := listSlot(title)
	return ok
}

func (c *acqContext) readList(page acqRawPage) {
	slot, _ := listSlot(page.Title)
	for _, m := range wikiListEntry.FindAllStringSubmatch(page.Text, -1) {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		id, title := fandomID(slot, n), strings.TrimSpace(m[2])
		if _, ok := c.cat.Names[id]; !ok {
			continue
		}
		if at, seen := c.listed[title]; seen && at != id {
			c.listed[title] = 0
			continue
		}
		c.listed[title] = id
	}
}

func (c *acqContext) normalized(name string) []int {
	ids := c.norm[normalizeGarment(name)]
	want, ok := qualifierSlot(name)
	if !ok {
		return ids
	}
	var out []int
	for _, id := range ids {
		if got, ok := qualifierSlot(c.cat.Names[id]); ok && got != want {
			continue
		}
		out = append(out, id)
	}
	return out
}

func qualifierSlot(name string) (scoring.Slot, bool) {
	m := wikiQualifier.FindStringSubmatch(name)
	if m == nil {
		return 0, false
	}
	return slotByName(m[1])
}

func pickQualified(ids []int, name string) int {
	m := wikiQualifier.FindStringSubmatch(name)
	if m == nil {
		return 0
	}
	slot, ok := slotByName(m[1])
	if !ok {
		return 0
	}
	found := 0
	for _, id := range ids {
		if SlotOfID(id) == slot {
			if found != 0 {
				return 0
			}
			found = id
		}
	}
	return found
}

func chineseSuits(suits []acqSuit) map[string]string {
	out := map[string]string{}
	claimed := map[string]bool{}
	for _, s := range suits {
		if s.chinese == "" {
			continue
		}
		if claimed[s.chinese] {
			delete(out, s.chinese)
			continue
		}
		claimed[s.chinese] = true
		out[s.chinese] = s.title
	}
	return out
}

func packTitles(suits []acqSuit) map[string]bool {
	out := map[string]bool{}
	for _, s := range suits {
		if s.pack {
			out[suitName(s.title)] = true
		}
	}
	return out
}

func suitsFirst(suits []acqSuit) []acqSuit {
	out := make([]acqSuit, 0, len(suits))
	for _, pack := range []bool{false, true} {
		for _, s := range suits {
			if s.pack == pack {
				out = append(out, s)
			}
		}
	}
	return out
}

func (c *acqContext) knownSuits(pages []acqPage, suits []acqSuit) map[int]string {
	suits = suitsFirst(suits)
	out := make(map[int]string, len(c.cat.Suits)+len(pages))
	for id, s := range c.cat.Suits {
		out[id] = s
	}
	for _, p := range pages {
		if s := cleanWiki(p.suit); s != "" && out[p.id] == "" {
			out[p.id] = s
		}
	}
	c.suits = out
	c.unfileMisnamed(pages, suits, out)
	for _, s := range suits {
		for _, part := range s.parts {
			if id := c.resolve(part.name, ""); id != 0 {
				if out[id] == "" {
					out[id] = s.title
				}
				continue
			}
			for _, id := range c.dayNightForms(part) {
				if out[id] == "" {
					out[id] = s.title
				}
			}
		}
	}
	learned := map[int]string{}
	for _, s := range suits {
		for _, part := range s.parts {
			if id := c.resolvePart(part.name, s.title); id != 0 && out[id] == "" && learned[id] == "" {
				learned[id] = s.title
			}
		}
	}
	for id, s := range learned {
		out[id] = s
	}
	first := map[int]string{}
	shared := map[int]bool{}
	for _, s := range suits {
		if s.pack {
			continue
		}
		for _, part := range s.parts {
			id := c.resolvePart(part.name, s.title)
			if id == 0 {
				continue
			}
			if at, seen := first[id]; seen && at != s.title {
				shared[id] = true
			} else if !seen {
				first[id] = s.title
			}
		}
	}
	c.stats.SharedParts = len(shared)
	return out
}

func (c *acqContext) unfileMisnamed(pages []acqPage, suits []acqSuit, out map[int]string) {
	byTitle := map[string]acqSuit{}
	listed := map[int]bool{}
	for _, s := range suits {
		byTitle[strings.ToLower(s.title)] = s
		if s.pack {
			continue
		}
		for _, part := range s.parts {
			if id := c.resolve(part.name, ""); id != 0 {
				listed[id] = true
			}
		}
	}
	var misnamed []int
	for _, p := range pages {
		s := cleanWiki(p.suit)
		named, ok := byTitle[strings.ToLower(s)]
		if s == "" || out[p.id] != s || !ok || !listed[p.id] || c.lists(named, p.id) {
			continue
		}
		misnamed = append(misnamed, p.id)
	}
	for _, id := range misnamed {
		out[id] = ""
	}
}

func (c *acqContext) lists(s acqSuit, id int) bool {
	for _, part := range s.parts {
		if c.resolvePart(part.name, s.title) == id {
			return true
		}
	}
	return false
}

func (c *acqContext) dayNightForms(part suitPart) []int {
	open := strings.LastIndex(part.name, "(")
	if open <= 0 {
		return nil
	}
	form := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(part.name[open+1:]), ")"))
	if !strings.EqualFold(form, "Day") && !strings.EqualFold(form, "Night") {
		return nil
	}
	var out []int
	for _, id := range c.lower[strings.ToLower(strings.TrimSpace(part.name[:open]))] {
		if partSlotFits(part.kind, SlotOfID(id)) {
			out = append(out, id)
		}
	}
	return out
}

func partSlotFits(kind string, slot scoring.Slot) bool {
	if kind == "" {
		return false
	}
	if want, ok := fandomSlots[kind]; ok {
		return slot == want
	}
	return slot == scoring.Accessory || slot == scoring.Hosiery
}

func (c *acqContext) unplaced(suits []acqSuit) []SuitPart {
	var out []SuitPart
	for _, s := range suitsFirst(suits) {
		if s.pack {
			continue
		}
		for _, part := range s.parts {
			if c.resolvePart(part.name, s.title) != 0 {
				continue
			}
			if ids := c.normalized(part.name); len(ids) > 1 {
				out = append(out, SuitPart{Suit: s.title, IDs: ids})
			}
		}
	}
	return out
}

func (c *acqContext) pick(ids []int, suit string, part bool) int {
	if len(ids) == 1 {
		return ids[0]
	}
	if suit == "" {
		return 0
	}
	held := 0
	for _, id := range ids {
		if strings.EqualFold(c.suits[id], suit) {
			held++
		}
	}
	if id := c.only(ids, func(id int) bool { return strings.EqualFold(c.suits[id], suit) }); id != 0 || !part || held > 1 {
		return id
	}
	return c.only(ids, func(id int) bool { return c.suits[id] == "" })
}

func (c *acqContext) only(ids []int, keep func(int) bool) int {
	found := 0
	for _, id := range ids {
		if keep(id) {
			if found != 0 {
				return 0
			}
			found = id
		}
	}
	return found
}

func upperFirst(s string) string {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (c *acqContext) unit(template, code string) (string, bool) {
	u, ok := c.units[template][strings.TrimSpace(code)]
	return u, ok
}

func (c *acqContext) amounts(text string) []Cost {
	var out []Cost
	for _, m := range wikiAmount.FindAllStringSubmatch(text, -1) {
		if cost, ok := c.cost(m[1], m[2], m[3]); ok {
			out = append(out, cost)
		}
	}
	return out
}

func (c *acqContext) cost(amount, template, code string) (Cost, bool) {
	n, err := strconv.Atoi(strings.ReplaceAll(amount, ",", ""))
	if err != nil || n <= 0 {
		return Cost{}, false
	}
	unit, ok := c.unit(template, code)
	if !ok {
		c.stats.UnknownUnits++
		return Cost{}, false
	}
	return Cost{Amount: n, Unit: unit}, true
}

func (c *acqContext) named(name, suit string) namedIngredient {
	name = strings.TrimSpace(cleanWiki(name))
	it := namedIngredient{name: name}
	if id := c.resolve(name, suit); id != 0 {
		it.id = id
		if display := c.cat.shown(id); display != "" && !hasHan(display) {
			it.name = display
		}
	} else {
		c.stats.Unresolved++
	}
	return it
}

func (c *acqContext) customizationCosts(body string) map[string][]Cost {
	out := map[string][]Cost{}
	for _, m := range wikiCustomTarget.FindAllStringSubmatch(body, -1) {
		key := normalizeGarment(m[1])
		if _, seen := out[key]; !seen {
			out[key] = c.amounts(m[2])
		}
	}
	return out
}

type acqToken struct {
	kind, text   string
	stage, level string
}

func (c *acqContext) pageAcquisition(p acqPage) []Acquisition {
	suit := cleanWiki(p.suit)
	if suit == "" {
		suit = c.suits[p.id]
	}
	recipes := c.recipes(p.sections["crafted from"], suit)
	details := map[string][]Acquisition{
		"Crafting":       recipes,
		"Evolution":      c.evolutions(evolvedFrom(p), suit),
		"Reconstruction": c.reconstruction(p.sections["reconstructed from"]),
		"Customization":  c.customization(p, suit),
	}
	used := map[string]bool{}

	var out []Acquisition
	for _, tok := range obtainTokens(p.obtain) {
		for _, t := range tokenSources(tok) {
			if d := details[t.text]; len(d) > 0 && t.kind != "other" {
				if !used[t.text] {
					used[t.text] = true
					for _, a := range d {
						out = appendAcquisition(out, a)
					}
				}
				continue
			}
			a, ok := c.fromToken(t, p, suit)
			if ok {
				out = appendAcquisition(out, a)
			}
		}
	}
	for _, key := range []string{"Crafting", "Evolution", "Reconstruction", "Customization"} {
		if !used[key] {
			for _, a := range details[key] {
				out = appendAcquisition(out, a)
			}
		}
	}
	return currentShops(c.applyPrices(out, p.intro), p.intro)
}

func (c *acqContext) fromToken(t acqToken, p acqPage, suit string) (Acquisition, bool) {
	switch {
	case t.kind == "":
		return Acquisition{}, false
	case t.kind == "stage" && t.stage != "":
		return storyAcquisition(t.stage, t.level, c.cat), true
	case t.kind == "stage" && !nameLike(t.text):
		c.stats.Unclassified++
		return Acquisition{}, false
	case t.kind == "stage":
		a := Acquisition{Kind: "stage", Text: "Story " + t.text}
		if t.level != "" {
			a.Text += " (" + t.level + ")"
		}
		return a, true
	case t.kind == "other" && t.text == "Other":
		c.stats.Unclassified++
	}
	a := Acquisition{Kind: t.kind, Text: t.text, Past: pastSource(t.kind, t.text)}
	if t.kind == "suit" && t.text == "Styling Gift Box" {
		a.Text = c.giftBoxText(p, suit)
	}
	return a, true
}

func obtainTokens(v string) []string {
	v = wikiComment.ReplaceAllString(v, "")
	var out []string
	for _, t := range wikiBreak.Split(v, -1) {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func tokenSources(tok string) []acqToken {
	if wikiStages.MatchString(tok) {
		level := stageLevel(tok)
		var out []acqToken
		for _, m := range wikiStages.FindAllStringSubmatch(tok, -1) {
			arg := strings.TrimSpace(m[1])
			name, _ := wikiStageName(arg)
			out = append(out, acqToken{kind: "stage", text: wikiVolume.ReplaceAllString(arg, ""), stage: name, level: level})
		}
		return out
	}
	if parts := splitTopLevel(tok, ','); len(parts) > 1 {
		var out []acqToken
		for _, p := range parts {
			t := classifySource(p)
			_, fixed := fixedSources[strings.ToLower(cleanWiki(p))]
			if !fixed && t.kind != "stage" && !singleLink(p) {
				out = nil
				break
			}
			out = append(out, t)
		}
		if out != nil {
			return out
		}
	}
	return []acqToken{classifySource(tok)}
}

func stageLevel(s string) string {
	if m := wikiLevel.FindStringSubmatch(s); m != nil {
		return strings.ToUpper(m[1][:1]) + strings.ToLower(m[1][1:])
	}
	return ""
}

func singleLink(s string) bool {
	s = strings.TrimSpace(s)
	loc := wikiLink.FindStringIndex(s)
	return loc != nil && loc[0] == 0 && strings.Trim(s[loc[1]:], " .") == ""
}

func splitTopLevel(s string, sep byte) []string {
	var out []string
	depth, last := 0, 0
	for i := 0; i < len(s); i++ {
		switch {
		case strings.HasPrefix(s[i:], "[[") || strings.HasPrefix(s[i:], "{{"):
			depth++
			i++
		case strings.HasPrefix(s[i:], "]]") || strings.HasPrefix(s[i:], "}}"):
			depth--
			i++
		case s[i] == sep && depth == 0:
			out = append(out, s[last:i])
			last = i + 1
		}
	}
	return append(out, s[last:])
}

func classifySource(tok string) acqToken {
	tok = wikiGluedEvent.ReplaceAllString(tok, "]] $1")
	text := cleanWiki(tok)
	lower := strings.ToLower(text)
	target := ""
	if m := wikiLink.FindStringSubmatch(tok); m != nil {
		target = strings.TrimSpace(m[1])
	}
	for _, s := range []string{target, text} {
		if rest, ok := cutPrefixFold(s, "Dreamland - "); ok {
			if i := strings.IndexAny(rest, "/#"); i >= 0 {
				rest = rest[:i]
			}
			return acqToken{kind: "dream", text: "Dream Weaver: " + strings.TrimSpace(rest)}
		}
	}
	if m := wikiStagePage.FindStringSubmatch(target); m != nil {
		name, _ := wikiStageName(m[1])
		return acqToken{kind: "stage", text: name, stage: name, level: stageLevel(tok)}
	}
	if m := wikiPlainStage.FindStringSubmatch(text); m != nil {
		arg := m[2]
		if m[1] != "" {
			arg = "V" + m[1] + ": " + arg
		}
		name, _ := wikiStageName(arg)
		return acqToken{kind: "stage", text: name, stage: name, level: stageLevel(text)}
	}
	if f, ok := fixedSources[lower]; ok {
		return acqToken{kind: f.kind, text: f.text}
	}
	switch {
	case lower == "" || lower == "unknown" || lower == "?":
		return acqToken{}
	case wikiFriends.MatchString(text):
		return acqToken{kind: "achievement", text: "Reach " + wikiFriends.FindStringSubmatch(text)[1] + " friends"}
	case strings.HasPrefix(lower, "completing ") || strings.HasPrefix(lower, "complete "):
		suit := strings.TrimSpace(text[strings.Index(text, " ")+1:])
		for _, prefix := range []string{"the suit ", "suit "} {
			if rest, ok := cutPrefixFold(suit, prefix); ok {
				suit = strings.TrimSpace(rest)
				break
			}
		}
		if strings.EqualFold(suit, "suit") || !nameLike(suit) {
			return acqToken{kind: "suit", text: "Styling Gift Box"}
		}
		return acqToken{kind: "suit", text: "Styling Gift Box for completing " + suit}
	case strings.HasPrefix(lower, "evolve"):
		return acqToken{kind: "evolve", text: "Evolution"}
	case strings.HasPrefix(lower, "customiz"):
		return acqToken{kind: "customize", text: "Customization"}
	case strings.Contains(lower, "sign-in") || strings.Contains(lower, "sign in"):
		return acqToken{kind: "signin", text: "Monthly Sign-In"}
	case strings.Contains(lower, "log-in") || strings.Contains(lower, "login"):
		return acqToken{kind: "signin", text: "Log-in Event"}
	case strings.Contains(lower, "recharge") || strings.Contains(lower, "pack") ||
		strings.Contains(lower, "vip") || strings.Contains(lower, "lucky bag"):
		return acqToken{kind: "recharge", text: nameOr(text, "Recharge")}
	case wikiEventWord.MatchString(text):
		name := strings.TrimSpace(strings.TrimRight(text[:wikiEventWord.FindStringIndex(text)[0]], " ,:-"))
		if f, ok := fixedSources[strings.ToLower(name)]; ok {
			return acqToken{kind: f.kind, text: f.text}
		}
		if name == "" || !nameLike(name) {
			return acqToken{kind: "event", text: "Limited event"}
		}
		return acqToken{kind: "event", text: name + " event"}
	case strings.Contains(lower, "mailbox"):
		return acqToken{kind: "gift", text: "Mailbox Gift"}
	case strings.Contains(lower, "redeem"):
		return acqToken{kind: "gift", text: "Redeem Code"}
	case strings.Contains(lower, "gift"):
		return acqToken{kind: "gift", text: nameOr(text, "Gift")}
	case strings.Contains(lower, "achievement") || strings.Contains(lower, "phase"):
		return acqToken{kind: "achievement", text: "Achievement"}
	case strings.Contains(lower, "pavilion") && nameLike(text):
		return acqToken{kind: "pavilion", text: text}
	case (strings.Contains(lower, "store") || strings.Contains(lower, "shop")) && nameLike(text):
		return acqToken{kind: "store", text: text}
	case singleLink(tok) && nameLike(text):
		return acqToken{kind: "event", text: text + " event"}
	case nameLike(text):
		return acqToken{kind: "other", text: text}
	}
	return acqToken{kind: "other", text: "Other"}
}

func nameOr(text, fallback string) string {
	if nameLike(text) {
		return text
	}
	return fallback
}

func cutPrefixFold(s, prefix string) (string, bool) {
	if len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix) {
		return s[len(prefix):], true
	}
	return s, false
}

func nameLike(s string) bool {
	words := strings.Fields(s)
	if len(words) == 0 || len(words) > 6 {
		return false
	}
	return !strings.ContainsAny(s, ".;:()[]{}<>=|\"")
}

func (c *acqContext) giftBoxText(p acqPage, suit string) string {
	if m := wikiGiftBoxSuit.FindStringSubmatch(p.intro); m != nil {
		name := m[2]
		if name == "" {
			name = m[1]
		}
		if name = cleanWiki(name); nameLike(name) {
			return giftBoxLine(name)
		}
	}
	return giftBoxLine(nameOr(suit, ""))
}

func wikiStageName(arg string) (string, bool) {
	m := wikiStageArg.FindStringSubmatch(arg)
	if m == nil {
		return "", false
	}
	volume := 1
	switch {
	case m[1] != "":
		volume, _ = strconv.Atoi(m[1])
	case m[2] != "":
		volume = len(m[2])
	}
	return storyName(volume, m[3], m[5], m[4] != ""), true
}

func (c *acqContext) recipes(section, suit string) []Acquisition {
	var out []Acquisition
	for i := strings.Index(section, "{{Recipe"); i >= 0; {
		args := namedArgs(templateArgs(section, i))
		var items []namedIngredient
		for n := 1; n <= 5; n++ {
			name := args["item"+strconv.Itoa(n)]
			if name == "" {
				continue
			}
			qty, _ := strconv.Atoi(args["item"+strconv.Itoa(n)+"_quantity"])
			it := c.named(name, suit)
			it.qty = qty
			items = append(items, it)
		}
		if len(items) > 0 {
			a := Acquisition{Kind: "craft", Text: "Craft: " + ingredientList(items), From: ingredients(items)}
			for _, extra := range []string{"dye", "special"} {
				if name := cleanWiki(args[extra]); name != "" {
					if qty, err := strconv.Atoi(args[extra+"_quantity"]); err == nil && qty > 0 {
						a.Cost = append(a.Cost, Cost{Amount: qty, Unit: name})
					}
				}
			}
			a.Text = plusCosts(a.Text, a.Cost)
			c.recipeSource(&a, args)
			out = append(out, a)
		}
		next := strings.Index(section[i+2:], "{{Recipe")
		if next < 0 {
			break
		}
		i += 2 + next
	}
	return out
}

func (c *acqContext) recipeSource(a *Acquisition, args map[string]string) {
	raw := args["source"]
	source := strings.ToLower(cleanWiki(raw))
	price := func(template, code string) string {
		cost, ok := c.cost(args["price"], template, code)
		if !ok {
			return ""
		}
		return withCosts("", []Cost{cost})
	}
	switch source {
	case "":
	case "default":
		a.Recipe = "Available by default"
	case "time diary":
		a.Recipe = "Time Diary"
	case "store of starlight", "store":
		a.Recipe = "Store of Starlight" + price("Currency", "SC")
	case "user's shop", "users shop":
		a.Recipe = "User's Shop" + price("Currency", "D")
	case "extra stage bonus", "stage dialogue", "stage":
		name, ok := wikiStageName(strings.TrimSpace(args["stage"]))
		if !ok {
			a.Recipe = "Extra Stage Bonus"
			if source != "extra stage bonus" {
				a.Recipe = "Stage dialogue"
			}
			return
		}
		level := "Maiden"
		if strings.EqualFold(strings.TrimSpace(args["stage_type"]), "p") {
			level = "Princess"
		}
		if source != "extra stage bonus" {
			level = ""
		}
		s := storyAcquisition(name, level, c.cat)
		a.Stage, a.Level = s.Stage, s.Level
		if source == "extra stage bonus" {
			a.Recipe = "Extra Stage Bonus on " + s.Text
		} else {
			a.Recipe = "Dialogue of " + s.Text
		}
	default:
		if m := wikiLink.FindStringSubmatchIndex(raw); m != nil {
			label := raw[m[2]:m[3]]
			if m[4] >= 0 {
				label = raw[m[4]:m[5]]
			}
			label = cleanWiki(label)
			if i := strings.IndexAny(label, "/#"); i > 0 {
				label = label[:i]
			}
			if f, ok := fixedSources[strings.ToLower(label)]; ok {
				label = f.text
			}
			rest := strings.ToLower(strings.TrimSpace(raw[m[1]:]))
			if strings.HasPrefix(rest, "event") && !strings.HasSuffix(strings.ToLower(label), "event") {
				label += " event"
			}
			if nameLike(label) {
				a.Recipe = label
			}
			return
		}
		if strings.Contains(source, "mailbox") {
			a.Recipe = "Mailbox Gift"
		}
	}
}

func plusCosts(text string, costs []Cost) string {
	for _, cost := range costs {
		text += " + " + thousands(cost.Amount) + " " + cost.Unit
	}
	return text
}

func evolvedFrom(p acqPage) string {
	if s := p.sections["evolved from"]; s != "" {
		return s
	}
	return p.sections["evolves from"]
}

type evolveStep struct {
	items []string
	costs []Cost
}

func (c *acqContext) evolutions(section, suit string) []Acquisition {
	var steps []evolveStep
	names := 0
	for _, line := range strings.Split(section, "\n") {
		icons := wikiIconItem.FindAllString(line, -1)
		if len(icons) == 0 {
			continue
		}
		names += len(icons)
		steps = append(steps, evolveStep{icons, c.amounts(wikiIconItem.ReplaceAllString(line, ""))})
	}
	var chosen []evolveStep
	for _, s := range steps {
		if len(s.costs) > 0 {
			chosen = append(chosen, s)
		}
	}
	if len(chosen) > 1 {
		chosen = []evolveStep{c.directStep(chosen, suit)}
	}
	if len(chosen) == 0 && names == 1 {
		chosen = steps
	}
	if len(chosen) == 0 {
		return nil
	}
	var items []namedIngredient
	var costs []Cost
	for _, s := range chosen {
		for _, icon := range s.items {
			m := wikiIconItem.FindStringSubmatch(icon)
			it := c.named(m[1], suit)
			it.qty = 1
			if q := wikiQuantity.FindStringSubmatch(m[2]); q != nil {
				it.qty, _ = strconv.Atoi(q[1])
			}
			items = append(items, it)
		}
		costs = append(costs, s.costs...)
	}
	return []Acquisition{{
		Kind: "evolve", Text: plusCosts("Evolve: "+ingredientList(items), costs),
		From: ingredients(items), Cost: costs,
	}}
}

func (c *acqContext) directStep(steps []evolveStep, suit string) evolveStep {
	ids := make([]int, len(steps))
	for i, s := range steps {
		ids[i] = c.resolve(cleanWiki(wikiIconItem.FindStringSubmatch(s.items[0])[1]), suit)
	}
	for i, id := range ids {
		p, ok := c.pages[id]
		if !ok {
			continue
		}
		for _, icon := range wikiIconItem.FindAllStringSubmatch(evolvedFrom(p), -1) {
			from := c.resolve(cleanWiki(icon[1]), suit)
			for j, other := range ids {
				if j != i && from != 0 && from == other {
					return steps[i]
				}
			}
		}
	}
	return steps[0]
}

func (c *acqContext) reconstruction(section string) []Acquisition {
	costs := c.amounts(section)
	if len(costs) == 0 {
		return nil
	}
	parts := make([]string, len(costs))
	for i, cost := range costs {
		parts[i] = thousands(cost.Amount) + " " + cost.Unit
	}
	return []Acquisition{{Kind: "reconstruct", Text: "Reconstruct: " + strings.Join(parts, " + "), Cost: costs}}
}

func (c *acqContext) customization(p acqPage, suit string) []Acquisition {
	m := wikiCustomizes.FindStringSubmatch(p.intro)
	if m == nil {
		return nil
	}
	if m[1] == "" {
		m[1] = m[2]
	}
	base := c.named(m[1], suit)
	target := strings.TrimSpace(m[1])
	if base.id != 0 && !madeFrom(base.id, p.id) {
		base, target = c.customizedFrom(p, suit)
	}
	a := Acquisition{Kind: "customize", Text: "Customization"}
	if base.name != "" {
		a.Text = "Customize: " + base.name
	}
	if base.id != 0 {
		a.From = []Ingredient{{ID: base.id, Qty: 1}}
	}
	if t, ok := c.redirects[target]; ok {
		target = t
	}
	if costs := c.customs[target][normalizeGarment(p.title)]; len(costs) > 0 {
		a.Cost = costs
		a.Text = plusCosts(a.Text, costs)
	}
	return []Acquisition{a}
}

func madeFrom(base, id int) bool {
	return base != id && SlotOfID(base) == SlotOfID(id)
}

func (c *acqContext) customizedFrom(p acqPage, suit string) (namedIngredient, string) {
	var found []string
	for _, title := range c.customizers[normalizeGarment(p.title)] {
		if id, ok := c.titles[title]; ok && madeFrom(id, p.id) {
			found = append(found, title)
		}
	}
	if len(found) != 1 {
		return namedIngredient{}, ""
	}
	return c.named(found[0], suit), found[0]
}

type pricedSource struct {
	token acqToken
	costs []Cost
}

func (c *acqContext) prices(intro string) []pricedSource {
	var out []pricedSource
	for _, m := range wikiPrice.FindAllStringSubmatchIndex(intro, -1) {
		amounts := [][3]string{{intro[m[2]:m[3]], intro[m[4]:m[5]], intro[m[6]:m[7]]}}
		rest := intro[m[1]:]
		for {
			more := wikiPriceMore.FindStringSubmatchIndex(rest)
			if more == nil {
				break
			}
			amounts = append(amounts, [3]string{rest[more[2]:more[3]], rest[more[4]:more[5]], rest[more[6]:more[7]]})
			rest = rest[more[1]:]
		}
		f, ok := shopAfterPrice(rest)
		if !ok {
			f, ok = shopBeforePrice(intro[:m[0]])
		}
		if !ok {
			continue
		}
		var costs []Cost
		for _, a := range amounts {
			if cost, ok := c.cost(a[0], a[1], a[2]); ok {
				costs = append(costs, cost)
			}
		}
		if len(costs) > 0 {
			out = append(out, pricedSource{token: acqToken{kind: f.kind, text: f.text}, costs: costs})
		}
	}
	return out
}

func shopAfterPrice(rest string) (fixedSource, bool) {
	m := wikiPriceShop.FindStringSubmatch(rest)
	if m == nil {
		return fixedSource{}, false
	}
	label := m[1]
	if m[2] != "" {
		label = m[2]
	}
	f, ok := fixedSources[strings.ToLower(cleanWiki(label))]
	return f, ok
}

func shopBeforePrice(before string) (fixedSource, bool) {
	links := wikiLink.FindAllStringSubmatchIndex(before, -1)
	if len(links) == 0 {
		return fixedSource{}, false
	}
	last := links[len(links)-1]
	if strings.ContainsAny(before[last[1]:], ".[{") {
		return fixedSource{}, false
	}
	label := before[last[2]:last[3]]
	if last[4] >= 0 {
		label = before[last[4]:last[5]]
	}
	f, ok := fixedSources[strings.ToLower(cleanWiki(label))]
	return f, ok
}

var currentShopKinds = map[string]bool{"store": true, "pavilion": true, "association": true}

func currentShops(list []Acquisition, intro string) []Acquisition {
	for _, l := range wikiLink.FindAllStringSubmatchIndex(intro, -1) {
		label := intro[l[2]:l[3]]
		if l[4] >= 0 {
			label = intro[l[4]:l[5]]
		}
		f, ok := fixedSources[strings.ToLower(cleanWiki(label))]
		if !ok || !currentShopKinds[f.kind] || !currentClause(intro[:l[0]]) || listsShop(list, f) {
			continue
		}
		list = append(list, Acquisition{Kind: f.kind, Text: f.text})
	}
	return list
}

func currentClause(before string) bool {
	if i := strings.LastIndexAny(before, ".\n"); i >= 0 {
		before = before[i+1:]
	}
	words := wikiTense.FindAllString(before, -1)
	if len(words) == 0 {
		return false
	}
	switch strings.ToLower(words[len(words)-1]) {
	case "can", "now":
		return true
	}
	return false
}

func listsShop(list []Acquisition, f fixedSource) bool {
	for _, a := range list {
		if a.Kind == f.kind && (a.Text == f.text || strings.HasPrefix(a.Text, f.text+" · ")) {
			return true
		}
	}
	return false
}

var priceable = map[string]bool{"store": true, "association": true, "pavilion": true, "craft": true}

func (c *acqContext) applyPrices(list []Acquisition, intro string) []Acquisition {
	for _, p := range c.prices(intro) {
		if !priceable[p.token.kind] {
			continue
		}
		matched := false
		for i := range list {
			a := &list[i]
			if a.Kind == p.token.kind && a.Text == p.token.text && len(a.Cost) == 0 {
				a.Cost = p.costs
				a.Text = withCosts(a.Text, p.costs)
				matched = true
				break
			}
		}
		if !matched && p.token.text != "Crafting" {
			list = appendAcquisition(list, Acquisition{Kind: p.token.kind, Text: withCosts(p.token.text, p.costs), Cost: p.costs})
		}
	}
	return list
}

func (c *acqContext) suitAcquisition(s acqSuit, out map[int][]Acquisition) {
	reward := map[int]bool{}
	for _, name := range s.reward {
		if id := c.resolvePart(name, s.title); id != 0 {
			reward[id] = true
		} else {
			c.stats.UnknownParts++
		}
	}
	var tokens []acqToken
	for _, tok := range obtainTokens(s.obtain) {
		for _, t := range tokenSources(tok) {
			if t.kind != "" && t.kind != "stage" && t.text != "Other" {
				tokens = append(tokens, t)
			}
		}
	}
	add := func(id int, a Acquisition) {
		out[id] = appendAcquisition(out[id], a)
	}
	for id := range reward {
		add(id, Acquisition{Kind: "suit", Text: "Styling Gift Box for completing " + s.title})
	}
	for _, part := range s.parts {
		id := c.resolvePart(part.name, s.title)
		if id == 0 {
			c.stats.UnknownParts++
			continue
		}
		if reward[id] {
			continue
		}
		for _, t := range tokens {
			a := Acquisition{Kind: t.kind, Text: t.text, Past: pastSource(t.kind, t.text)}
			if t.kind == "suit" {
				a.Text = "Styling Gift Box for completing " + s.title
			}
			add(id, a)
		}
	}
}
