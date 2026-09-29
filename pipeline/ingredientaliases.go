package pipeline

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type IngredientAlias struct {
	Page  string `json:"page"`
	Name  string `json:"name"`
	ID    int    `json:"id"`
	Basis string `json:"basis"`
}

func ReadIngredientAliases(raw []byte) ([]IngredientAlias, error) {
	var doc struct {
		Aliases []IngredientAlias `json:"aliases"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("ingredient aliases: %w", err)
	}
	seen := map[[2]string]bool{}
	for _, a := range doc.Aliases {
		key := [2]string{a.Page, a.Name}
		switch {
		case seen[key]:
			return nil, fmt.Errorf("ingredient aliases: %s's %q is listed twice", a.Page, a.Name)
		case strings.TrimSpace(a.Page) == "" || strings.TrimSpace(a.Name) == "" || a.ID <= 0 || strings.TrimSpace(a.Basis) == "":
			return nil, fmt.Errorf("ingredient aliases: %s's %q must give its page, the name, the item and the basis for it", a.Page, a.Name)
		}
		seen[key] = true
	}
	return doc.Aliases, nil
}

func (c *acqContext) ingredientAlias(page, name string) int {
	if i := slices.IndexFunc(c.cat.IngredientAliases, func(a IngredientAlias) bool { return a.Page == page && a.Name == name }); i >= 0 {
		c.aliased[i] = true
		return c.cat.IngredientAliases[i].ID
	}
	return 0
}

func (c *acqContext) checkIngredientAliases() error {
	var stale []string
	for i, a := range c.cat.IngredientAliases {
		_, known := c.cat.Names[a.ID]
		switch {
		case !known:
			stale = append(stale, fmt.Sprintf("%d is not in the catalogue", a.ID))
		case c.resolve(a.Name, "") != 0:
			stale = append(stale, fmt.Sprintf("%s's %q matches an item without it", a.Page, a.Name))
		case !c.aliased[i]:
			stale = append(stale, fmt.Sprintf("%s's page no longer names %q", a.Page, a.Name))
		}
	}
	if len(stale) > 0 {
		return fmt.Errorf("ingredient aliases: %d entries no longer fit the wiki: %s", len(stale), strings.Join(stale, "; "))
	}
	return nil
}
