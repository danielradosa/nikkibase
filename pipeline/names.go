package pipeline

import (
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

var compoundPrefix = map[string]bool{
	"all": true, "anti": true, "bi": true, "co": true, "cross": true, "de": true,
	"double": true, "ex": true, "extra": true, "half": true, "high": true,
	"ill": true, "inter": true, "low": true, "mid": true, "multi": true,
	"non": true, "off": true, "one": true, "over": true, "post": true,
	"pre": true, "re": true, "self": true, "semi": true, "short": true,
	"single": true, "sub": true, "super": true, "three": true, "tri": true,
	"triple": true, "two": true, "ultra": true, "under": true, "up": true,
	"well": true, "long": true,
}

var (
	runsOfSpace     = regexp.MustCompile(`\s{2,}`)
	middot          = regexp.MustCompile(`\s*·\s*`)
	tightAmpersand  = regexp.MustCompile(`([A-Za-z]{2,})&([A-Za-z]{2,})`)
	tightHyphen     = regexp.MustCompile(`^([A-Za-z]{2,})-([A-Za-z]{3,})$`)
	separator       = regexp.MustCompile(`\s*·\s*|\s+-\s*|\s*-\s+`)
	trailingBracket = regexp.MustCompile(`^(.*\S)\s+(\([^()]+\))$`)
	oneSidedHyphen  = regexp.MustCompile(`(\S) -(\S)|(\S)- (\S)`)
	suitQualifier   = regexp.MustCompile(`\s*\([^()]*\bSuit\)$`)
	nameSeparator   = regexp.MustCompile(`\s*[·-]\s*`)
)

func gameSpelling(name string) string {
	s := strings.TrimSpace(runsOfSpace.ReplaceAllString(name, " "))
	s = middot.ReplaceAllString(s, "-")
	return oneSidedHyphen.ReplaceAllString(s, "$1$3-$2$4")
}

func inGameSpelling(name, global string) string {
	return withGameSeparators(gameSpelling(name), global)
}

func withGameSeparators(name, global string) string {
	global = strings.TrimSpace(runsOfSpace.ReplaceAllString(global, " "))
	if global == "" || hasHan(global) {
		return name
	}
	parts, _ := splitAtSeparators(name)
	globalParts, globalSeps := splitAtSeparators(global)
	if len(parts) == 1 || len(parts) != len(globalParts) {
		return name
	}
	for k := range parts {
		if !strings.EqualFold(parts[k], globalParts[k]) {
			return name
		}
	}
	var b strings.Builder
	for k, part := range parts {
		if k > 0 {
			b.WriteString(globalSeps[k-1])
		}
		b.WriteString(part)
	}
	return b.String()
}

func splitAtSeparators(s string) (parts, seps []string) {
	at := 0
	for _, m := range nameSeparator.FindAllStringIndex(s, -1) {
		parts = append(parts, s[at:m[0]])
		seps = append(seps, s[m[0]:m[1]])
		at = m[1]
	}
	return append(parts, s[at:]), seps
}

func shownSuit(suit string) string {
	return suitQualifier.ReplaceAllString(suit, "")
}

func gameName(name, global string) string {
	global = calcName(global)
	for {
		m := trailingBracket.FindStringSubmatch(name)
		if m == nil || strings.HasSuffix(global, m[2]) {
			return name
		}
		name = m[1]
	}
}

func NormalizeName(name string, spellings ...string) string {
	s := strings.TrimSpace(runsOfSpace.ReplaceAllString(name, " "))
	s = middot.ReplaceAllString(s, " · ")
	s = tightAmpersand.ReplaceAllString(s, "$1 & $2")

	words := spaceLooseHyphens(strings.Split(s, " "))
	var marked []string
	for i, w := range words {
		if !strings.Contains(w, "-") || spelledJoined(w, spellings) {
			continue
		}
		if marked == nil {
			marked = make([]string, len(spellings))
			for k, sp := range spellings {
				marked[k] = separator.ReplaceAllString(sp, " · ")
			}
		}
		if spaced, ok := spelledApart(w, marked); ok {
			words[i] = spaced
			continue
		}
		if strings.Count(w, "-") != 1 {
			continue
		}
		m := tightHyphen.FindStringSubmatch(w)
		if m == nil || compoundPrefix[strings.ToLower(m[1])] || unicode.IsLower(rune(m[2][0])) {
			continue
		}
		words[i] = m[1] + " - " + m[2]
	}
	return strings.Join(words, " ")
}

func spaceLooseHyphens(words []string) []string {
	out := make([]string, 0, len(words))
	for i, w := range words {
		if i > 0 && len(w) > 1 && w[0] == '-' && w[1] != '-' {
			out = append(out, "-")
			w = w[1:]
		}
		if i < len(words)-1 && len(w) > 1 && w[len(w)-1] == '-' && w[len(w)-2] != '-' {
			out = append(out, w[:len(w)-1], "-")
			continue
		}
		out = append(out, w)
	}
	return out
}

func spelledApart(word string, marked []string) (string, bool) {
	parts := strings.Split(word, "-")
	if slices.Contains(parts, "") {
		return word, false
	}
	quoted := make([]string, len(parts))
	for k, part := range parts {
		quoted[k] = regexp.QuoteMeta(part)
	}
	re := regexp.MustCompile(`(?i)` + strings.Join(quoted, `(-| · )`))
	apart := make([]bool, len(parts)-1)
	found := false
	for _, s := range marked {
		for from := 0; from < len(s); {
			m := re.FindStringSubmatchIndex(s[from:])
			if m == nil {
				break
			}
			start, end := from+m[0], from+m[1]
			before, _ := utf8.DecodeLastRuneInString(s[:start])
			after, _ := utf8.DecodeRuneInString(s[end:])
			if !inWord(before) && !inWord(after) {
				for k := range apart {
					if s[from+m[2+2*k]:from+m[3+2*k]] != "-" {
						apart[k], found = true, true
					}
				}
			}
			_, size := utf8.DecodeRuneInString(s[start:])
			from = start + max(size, 1)
		}
	}
	if !found {
		return word, false
	}
	var b strings.Builder
	for k, part := range parts {
		if k > 0 {
			if apart[k-1] {
				b.WriteString(" - ")
			} else {
				b.WriteByte('-')
			}
		}
		b.WriteString(part)
	}
	return b.String(), true
}

func spelledJoined(word string, spellings []string) bool {
	word = strings.ToLower(word)
	for _, s := range spellings {
		s = strings.ToLower(s)
		for from := 0; ; {
			at := strings.Index(s[from:], word)
			if at < 0 {
				break
			}
			start, end := from+at, from+at+len(word)
			before, _ := utf8.DecodeLastRuneInString(s[:start])
			after, _ := utf8.DecodeRuneInString(s[end:])
			if !inWord(before) && !inWord(after) {
				return true
			}
			from = start + 1
		}
	}
	return false
}

func inWord(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
