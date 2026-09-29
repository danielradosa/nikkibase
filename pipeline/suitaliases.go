package pipeline

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type SuitPartAlias struct {
	Suit  string `json:"suit"`
	Part  string `json:"part"`
	ID    int    `json:"id"`
	Basis string `json:"basis"`
}

func ReadSuitPartAliases(raw []byte) ([]SuitPartAlias, error) {
	var doc struct {
		Aliases []SuitPartAlias `json:"aliases"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("suit part aliases: %w", err)
	}
	seen := map[[2]string]bool{}
	for _, a := range doc.Aliases {
		key := [2]string{a.Suit, a.Part}
		switch {
		case seen[key]:
			return nil, fmt.Errorf("suit part aliases: %s's %q is listed twice", a.Suit, a.Part)
		case strings.TrimSpace(a.Suit) == "" || strings.TrimSpace(a.Part) == "" || a.ID <= 0 || strings.TrimSpace(a.Basis) == "":
			return nil, fmt.Errorf("suit part aliases: %s's %q must give its suit, the part, the item and the basis for it", a.Suit, a.Part)
		}
		seen[key] = true
	}
	return doc.Aliases, nil
}

func (c *acqContext) partAlias(name, suit string) int {
	if i := slices.IndexFunc(c.cat.PartAliases, func(a SuitPartAlias) bool { return a.Suit == suit && a.Part == name }); i >= 0 {
		return c.cat.PartAliases[i].ID
	}
	return 0
}

func (c *acqContext) checkPartAliases(suits []acqSuit) error {
	var stale []string
	for _, a := range c.cat.PartAliases {
		listed := slices.ContainsFunc(suits, func(s acqSuit) bool {
			return s.title == a.Suit && slices.ContainsFunc(s.parts, func(p suitPart) bool { return p.name == a.Part })
		})
		_, known := c.cat.Names[a.ID]
		switch {
		case !known:
			stale = append(stale, fmt.Sprintf("%d is not in the catalogue", a.ID))
		case !listed:
			stale = append(stale, fmt.Sprintf("%s's page no longer lists %q", a.Suit, a.Part))
		case c.lookup(a.Part, a.Suit, true) != 0:
			stale = append(stale, fmt.Sprintf("%s's %q matches an item without it", a.Suit, a.Part))
		}
	}
	if len(stale) > 0 {
		return fmt.Errorf("suit part aliases: %d entries no longer fit the wiki: %s", len(stale), strings.Join(stale, "; "))
	}
	return nil
}
