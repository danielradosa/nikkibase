package pipeline

import (
	"fmt"
	"regexp"
	"strings"
)

var handAlias = map[string]string{
	"乐队风":  "Musician",
	"轻熟风":  "Chic",
	"动物系":  "Animal",
	"潮酷风":  "Street",
	"工装风":  "Workwear",
	"航海风":  "Sailor",
	"现代流行": "POP",
	"异域风":  "Multicultural",
}

var renamed = map[string]string{
	"Pet":      "Animal",
	"Rock":     "Musician",
	"Office":   "Chic",
	"Harajuku": "Street",
	"Navy":     "Sailor",
	"Hindu":    "Multicultural",
	"Denim":    "Workwear",
}

func current(name string) string {
	if now, ok := renamed[name]; ok {
		return now
	}
	return name
}

var tagID = func() map[string]int {
	m := make(map[string]int, len(styleOrder))
	for i, name := range styleOrder {
		m[name] = i
	}
	return m
}()

func TagNames() []string { return styleOrder }

func TagName(id int) string {
	if id >= 0 && id < len(styleOrder) {
		return styleOrder[id]
	}
	return fmt.Sprintf("tag %d", id)
}

func TagID(name string) (int, bool) {
	id, ok := tagID[name]
	return id, ok
}

func StyleFromCode(code string) (string, bool) {
	name, ok := styleName[strings.TrimSpace(code)]
	return current(name), ok
}

var parenthesised = regexp.MustCompile(`^(.*?)\((.*)\)$`)

func StyleFromStageTag(tag string) (string, bool) {
	name, ok := styleFromStageTag(tag)
	return current(name), ok
}

func styleFromStageTag(tag string) (string, bool) {
	s := strings.TrimSpace(tag)
	for range 4 {
		if name, ok := styleAlias[s]; ok {
			return name, true
		}
		if name, ok := handAlias[s]; ok {
			return name, true
		}
		if _, ok := tagID[s]; ok {
			return s, true
		}
		m := parenthesised.FindStringSubmatch(s)
		if m == nil {
			return "", false
		}
		if outer := strings.TrimSpace(m[1]); outer != "" {
			if _, ok := tagID[outer]; ok {
				return outer, true
			}
		}
		s = strings.TrimSpace(m[2])
	}
	return "", false
}
