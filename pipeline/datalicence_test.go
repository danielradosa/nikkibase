package pipeline

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestDataLicenceAgreesWithRegistry(t *testing.T) {
	raw, err := os.ReadFile("../data/sources.json")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := ReadRegistry(raw)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile("../data/production-exceptions.json")
	if err != nil {
		t.Fatal(err)
	}
	ex, err := ReadExceptions(raw, reg)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := os.ReadFile("../DATA-LICENSE.md")
	if err != nil {
		t.Fatal(err)
	}

	rows := map[string][]string{}
	row := regexp.MustCompile("^\\|[^|]*\\| `([a-z0-9-]+)` \\|(.*)\\|\\s*$")
	for _, line := range strings.Split(string(doc), "\n") {
		if m := row.FindStringSubmatch(line); m != nil {
			cells := strings.Split(m[2], "|")
			for i := range cells {
				cells[i] = strings.TrimSpace(cells[i])
			}
			rows[m[1]] = cells
		}
	}

	for _, s := range reg.Sources {
		cells, ok := rows[s.ID]
		if !ok {
			t.Errorf("%s is in data/sources.json but has no row in DATA-LICENSE.md", s.ID)
			continue
		}
		if len(cells) != 4 {
			t.Fatalf("%s: want 4 cells after the ID, got %q", s.ID, cells)
		}
		licensed := s.Status == Redistributable
		permitted := s.Status == Permitted
		excepted := s.Status == PermissionRequired && ex.find(s.ID) != nil
		want := [4]string{"No", "No", "No", "Yes"}
		switch {
		case licensed:
			want = [4]string{"Yes — " + s.Licence, "Yes — under that licence", "Yes — under licence", "No"}
		case permitted:
			want = [4]string{"No", "Yes — in writing, from the maintainer", "Yes — with permission", "No"}
		case excepted:
			want = [4]string{"No", "No", "Yes — recorded exception", "No"}
		}
		for i, w := range want {
			if cells[i] != w {
				t.Errorf("%s column %d: DATA-LICENSE.md says %q, the registry implies %q", s.ID, i+3, cells[i], w)
			}
		}
		delete(rows, s.ID)
	}
	for id := range rows {
		t.Errorf("DATA-LICENSE.md has a row for %s, which is not in data/sources.json", id)
	}
}

func TestDataLicenceNamesWhatAcquisitionShips(t *testing.T) {
	doc, err := os.ReadFile("../DATA-LICENSE.md")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`It reproduces (.+?) names only`).FindStringSubmatch(strings.Join(strings.Fields(string(doc)), " "))
	if m == nil {
		t.Fatal("DATA-LICENSE.md no longer says which game names it reproduces")
	}
	for _, kind := range []string{"item", "suit", "stage", "style", "event", "shop", "pavilion", "character", "currency", "material"} {
		if !strings.Contains(m[1], kind) {
			t.Errorf("acquire.json ships %s names, and DATA-LICENSE.md lists only %q", kind, m[1])
		}
	}
}
