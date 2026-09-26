package wardrobe

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

const selPrefix = "@SEL"

func DecodeSelections(src []byte) (*Wardrobe, error) {
	text := strings.TrimLeftFunc(string(src), unicode.IsSpace)
	if !strings.HasPrefix(text, selPrefix) {
		return nil, errors.New(`wardrobe: not a selections file; no "` + selPrefix + `" header`)
	}

	w := &Wardrobe{}
	for _, entry := range strings.FieldsFunc(unheader(text), selSeparator) {
		if id, ok := selID(entry); ok {
			w.Items = append(w.Items, id)
		} else {
			w.Unresolved++
		}
	}
	if len(w.Items) == 0 && w.Unresolved == 0 {
		return nil, errors.New("wardrobe: selections file lists no items")
	}
	slices.Sort(w.Items)
	read := len(w.Items)
	w.Items = slices.Compact(w.Items)
	w.Duplicates = read - len(w.Items)
	return w, nil
}

func unheader(s string) string {
	var b strings.Builder
	for {
		at := strings.Index(s, selPrefix)
		if at < 0 {
			b.WriteString(s)
			return b.String()
		}
		b.WriteString(s[:at])
		b.WriteByte(',')
		s = s[at+len(selPrefix):]
		if rest, ok := strings.CutPrefix(s, "VER"); ok {
			s = rest
			if s != "" && s[0] >= '0' && s[0] <= '9' {
				s = s[1:]
			}
		}
	}
}

func selSeparator(r rune) bool {
	return r == ',' || r == '=' || unicode.IsSpace(r)
}

func selID(entry string) (int, bool) {
	var index, id string
	switch at := strings.IndexByte(entry, '_'); {
	case at >= 0:
		index, id = entry[:at], entry[at+1:]
	case len(entry) == 5 || len(entry) == 6:
		return selGameID(entry)
	case len(entry) == 8 || len(entry) == 9:
		index, id = entry[:legacyPrefixLen], entry[legacyPrefixLen:]
	default:
		return 0, false
	}
	if !selIndex(index) {
		return 0, false
	}
	return selGameID(id)
}

func selGameID(s string) (int, bool) {
	if len(s) < 5 || len(s) > 6 || s[0] == '0' || !asciiDigits(s) {
		return 0, false
	}
	id, err := strconv.Atoi(s)
	return id, err == nil
}

func selIndex(s string) bool {
	return len(s) >= 1 && len(s) <= 3 && asciiDigits(s)
}

func asciiDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

const legacyPrefixLen = 3
