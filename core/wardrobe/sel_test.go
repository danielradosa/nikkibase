package wardrobe

import (
	"regexp"
	"slices"
	"strconv"
	"testing"
)

func TestDecodeSelections(t *testing.T) {
	for _, tc := range []struct {
		name           string
		src            string
		want           []int
		wantUnresolved int
	}{
		{
			name: "ver1",
			src:  "@SELVER1=742_20062,131_20063,7_10001",
			want: []int{10001, 20062, 20063},
		},
		{
			name: "bare header with underscores",
			src:  "@SEL=742_20062,131_20063",
			want: []int{20062, 20063},
		},
		{
			name: "legacy entries lose their first three characters",
			src:  "@SEL=74220062,13120063,00710001",
			want: []int{10001, 20062, 20063},
		},
		{
			name: "ver1 without underscores keeps the whole entry",
			src:  "@SELVER1=20062,20063",
			want: []int{20062, 20063},
		},
		{
			name: "bare and underscored entries in one file are each read by their own shape",
			src:  "@SEL=20062,131_20063",
			want: []int{20062, 20063},
		},
		{
			name: "accessory and spirit ids",
			src:  "@SELVER1=1_170123,2_880045",
			want: []int{170123, 880045},
		},
		{
			name: "duplicates collapse",
			src:  "@SELVER1=742_20062,131_20062,999_20062,131_20063",
			want: []int{20062, 20063},
		},
		{
			name: "trailing separator and newline",
			src:  "@SELVER1=742_20062, 131_20063,\n",
			want: []int{20062, 20063},
		},
		{
			name: "a second export's entries are read, not just counted",
			src:  "@SELVER1=742_20062,131_20063=1_20064,2_20065",
			want: []int{20062, 20063, 20064, 20065},
		},
		{
			name: "three merged exports lose nothing",
			src:  "@SELVER1=1_10001,2_10002=1_10003,2_10004=1_10005,2_10006",
			want: []int{10001, 10002, 10003, 10004, 10005, 10006},
		},
		{
			name: "merged exports with their headers",
			src:  "@SELVER1=1_10001,2_10002\n@SELVER1=1_10003\n@SEL=74210004",
			want: []int{10001, 10002, 10003, 10004},
		},
		{
			name: "a line break separates entries",
			src:  "@SELVER1=1_10001\n2_10002,3_10003",
			want: []int{10001, 10002, 10003},
		},
		{
			name: "spaces, tabs and CRLF separate entries",
			src:  "@SEL=1_10001 2_10002\t3_10003\r\n4_10004",
			want: []int{10001, 10002, 10003, 10004},
		},
		{
			name: "a header without '=' ends at itself",
			src:  "@SELVER1=1_10001,2_10002@SELVER1\n3_10003",
			want: []int{10001, 10002, 10003},
		},
		{
			name: "the first header needs no '=' either",
			src:  "@SELVER1\n1_10001,2_10002",
			want: []int{10001, 10002},
		},
		{
			name: "a header without '=' takes one digit of version, not the entry after it",
			src:  "@SELVER1=1_10001@SELVER1742_20062,@SELVER174210003\n@SELVER120063",
			want: []int{10001, 10003, 20062, 20063},
		},
		{
			name: "a file of one header and one entry, with no '=' between",
			src:  "@SELVER120062",
			want: []int{20062},
		},
		{
			name:           "a version is at most one digit",
			src:            "@SELVER12=1_10001\n@SELVER=2_10002",
			want:           []int{10001, 10002},
			wantUnresolved: 1,
		},
		{
			name: "legacy and underscored entries in one file",
			src:  "@SEL=74210001,13110002,1_10003",
			want: []int{10001, 10002, 10003},
		},
		{
			name:           "ids of the wrong shape are counted, not owned",
			src:            "@SELVER1=1_123456789,2_42,3_1234567",
			wantUnresolved: 3,
		},
		{
			name:           "a signed or zero-padded id is counted, not owned",
			src:            "@SELVER1=1_+20062,2_020062,+20062,020062,742020062",
			wantUnresolved: 5,
		},
		{
			name: "an index of 0 or zero-padded is genuine",
			src:  "@SELVER1=0_20062,007_20063,00720064",
			want: []int{20062, 20063, 20064},
		},
		{
			name:           "an index nikkicalc would not write is counted",
			src:            "@SELVER1=_20062,+1_20063,a_20064,1000_20065,abc20066",
			wantUnresolved: 5,
		},
		{
			name:           "entries run together are counted, not half-read",
			src:            "@SEL=131_200631_20064,742100041_10003",
			wantUnresolved: 2,
		},
		{
			name:           "unreadable entries are counted",
			src:            "@SELVER1=742_20062,131_,_,742_20x62,nonsense",
			want:           []int{20062},
			wantUnresolved: 4,
		},
		{
			name:           "legacy entry too short to hold an id",
			src:            "@SEL=74220062,12",
			want:           []int{20062},
			wantUnresolved: 1,
		},
		{
			name:           "nothing readable at all",
			src:            "@SELVER1=a_b,c_d",
			wantUnresolved: 2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if items, unresolved, _ := reading(tc.src); !slices.Equal(items, tc.want) || unresolved != tc.wantUnresolved {
				t.Fatalf("the format reads %v with %d unresolved; this case wants %v with %d",
					items, unresolved, tc.want, tc.wantUnresolved)
			}

			got, err := DecodeSelections([]byte(tc.src))
			if err != nil {
				t.Fatalf("DecodeSelections: %v", err)
			}
			n := len(entries(tc.src))
			if total := len(got.Items) + got.Unresolved + got.Duplicates; total != n {
				t.Errorf("%d items + %d unresolved + %d duplicates = %d, and the file lists %d entries",
					len(got.Items), got.Unresolved, got.Duplicates, total, n)
			}
			if !slices.Equal(got.Items, tc.want) {
				t.Errorf("Items = %v, want %v", got.Items, tc.want)
			}
			if got.Unresolved != tc.wantUnresolved {
				t.Errorf("Unresolved = %d, want %d", got.Unresolved, tc.wantUnresolved)
			}
		})
	}
}

func TestDecodeSelectionsRejectsGarbage(t *testing.T) {
	for _, tc := range []struct{ name, src string }{
		{"empty", ""},
		{"whitespace only", " \n\t"},
		{"entries without a header", "742_20062,131_20063"},
		{"a clothes_date file", "e1sxMDAwMV09MTcwMDAwMDAwMCx9"},
		{"header alone", "@SELVER1"},
		{"header without entries", "@SELVER1="},
		{"headers without entries", "@SELVER1=\n@SELVER1\n@SEL="},
		{"separators without entries", "@SELVER1=,,,"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := DecodeSelections([]byte(tc.src)); err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}

var (
	selOpening   = regexp.MustCompile(`^[\t\n\v\f\r\x{85}\p{Z}]*@SEL`)
	selHeader    = regexp.MustCompile(`@SEL(VER\d?)?`)
	selEntry     = regexp.MustCompile(`[^,=\t\n\v\f\r\x{85}\p{Z}]+`)
	selGameEntry = regexp.MustCompile(`^(?:[0-9]{1,3}_|[0-9]{3})?([1-9][0-9]{4,5})$`)
)

func entries(src string) []string {
	return selEntry.FindAllString(selHeader.ReplaceAllLiteralString(src, ","), -1)
}

func reading(src string) (items []int, unresolved, duplicates int) {
	seen := map[int]bool{}
	for _, e := range entries(src) {
		m := selGameEntry.FindStringSubmatch(e)
		if m == nil {
			unresolved++
			continue
		}
		id, _ := strconv.Atoi(m[1])
		if seen[id] {
			duplicates++
			continue
		}
		seen[id] = true
		items = append(items, id)
	}
	slices.Sort(items)
	return items, unresolved, duplicates
}

func FuzzDecodeSelections(f *testing.F) {
	for _, seed := range []string{
		"@SELVER1=742_20062,131_20063",
		"@SEL=74210001,13110002,1_10003",
		"@SELVER1=1_10001,2_10002=1_10003\n@SEL=74210004",
		"@SELVER1=1_123456789,,_,a_b",
		"@SELVER1=1_10001\n2_10002,3_10003",
		"@SELVER1=1_10001,2_10002@SELVER1\n3_10003",
		"@SELVER1=1_10001@SELVER1742_20062,@SELVER174210003@SELVER120063",
		"@SELVER12=1_10001\n@SELVER=2_10002",
		"@SELVER1=1_+20062,020062,0_20063,007_20064,131_200631_20065",
		"@SEL=",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, src string) {
		w, err := DecodeSelections([]byte(src))
		isFile := selOpening.MatchString(src) && len(entries(src)) > 0
		if err != nil {
			if isFile {
				t.Fatalf("refused %q, which opens with a header and lists entries: %v", src, err)
			}
			return
		}
		if !isFile {
			t.Fatalf("accepted %q, which does not open with a header or lists nothing", src)
		}
		for i, id := range w.Items {
			if id < 10000 || id > 999999 {
				t.Fatalf("item %d is not shaped like a game ID", id)
			}
			if i > 0 && w.Items[i-1] >= id {
				t.Fatalf("items are not sorted and unique: %v", w.Items)
			}
		}
		if total, n := len(w.Items)+w.Unresolved+w.Duplicates, len(entries(src)); total != n {
			t.Fatalf("accounted for %d of %d entries in %q", total, n, src)
		}
		items, unresolved, duplicates := reading(src)
		if !slices.Equal(w.Items, items) || w.Unresolved != unresolved || w.Duplicates != duplicates {
			t.Fatalf("read %q as %v, %d unresolved, %d duplicates; the format says %v, %d, %d",
				src, w.Items, w.Unresolved, w.Duplicates, items, unresolved, duplicates)
		}
	})
}
