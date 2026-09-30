package push

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type data struct {
	version string
	items   string
	acquire string
	stages  string
	tags    string
}

var base = data{
	version: "cloud",
	items: `{"items":[
		[10001,"Nikki's Pinky",0,1,3,5,7,8,"S","A","A","A","A",2,""],
		[10002,"Rose Bun",0,1,3,5,7,8,"S","A","A","A","A",3,"Metallic Crisis"]
	]}`,
	acquire: `{"version":"cloud","items":{"10001":[{"k":"other","t":"Available from the start"}]}}`,
	stages:  `[{"name":"1-1","mode":"Story","weights":[1,2,3,4,5],"attrs":[1,3,5,7,9]}]`,
	tags:    `["Animal","Apron"]`,
}

func writeRoot(t *testing.T, d data) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"data/index.json":                        `{"version":"` + d.version + `"}`,
		"data/" + d.version + "/items.json":      d.items,
		"data/" + d.version + "/acquire.json":    d.acquire,
		"data/" + d.version + "/stages.json":     d.stages,
		"data/" + d.version + "/tags.json":       d.tags,
		"data/" + d.version + "/positions.json":  `[{"name":"Hair","slot":0}]`,
		"data/" + d.version + "/provenance.json": `{"changes":"every release"}`,
	}
	for path, body := range files {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func release(t *testing.T, d data) *Release {
	t.Helper()
	r, err := ReadRelease(writeRoot(t, d))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func compare(t *testing.T, old, cur data) Changes {
	t.Helper()
	before := release(t, old)
	path := filepath.Join(t.TempDir(), "release.json")
	if err := before.Save(path); err != nil {
		t.Fatal(err)
	}
	saved, err := LoadRelease(path)
	if err != nil {
		t.Fatal(err)
	}
	return Compare(saved, release(t, cur))
}

func TestANewNameWithTheSameDataChangesNothing(t *testing.T) {
	next := base
	next.version = "pigeon"
	next.acquire = strings.Replace(base.acquire, `"cloud"`, `"pigeon"`, 1)
	next.items = strings.ReplaceAll(base.items, "\n\t\t", "")
	if c := compare(t, base, next); !c.Empty() {
		t.Fatalf("a renamed, reformatted copy counts as changed: %+v", c)
	}
}

func TestNewItemsAreListedWithTheirSuits(t *testing.T) {
	next := base
	next.items = strings.Replace(base.items, `]
	]}`, `],
		[10003,"Rose Gown",1,1,3,5,7,8,"S","A","A","A","A",4,"Metallic Crisis"],
		[10004,"Tranquil Veil",0,1,3,5,7,8,"S","A","A","A","A",4,"Tranquil Rose (Hidden Suit)"],
		[10005,"Plain Socks",5,1,3,5,7,8,"S","A","A","A","A",1,""]
	]}`, 1)
	c := compare(t, base, next)
	if strings.Join(c.NewItems, "|") != "Rose Gown|Tranquil Veil|Plain Socks" {
		t.Errorf("new items %q", c.NewItems)
	}
	if strings.Join(c.NewSuits, "|") != "Metallic Crisis|Tranquil Rose" {
		t.Errorf("new suits %q", c.NewSuits)
	}
	if c.FixedItems != 0 || c.Touches("fixes") || c.Touches("stages") {
		t.Errorf("only items were added: %+v", c)
	}
}

func TestChangedAndRemovedItemsAndStagesAreFixes(t *testing.T) {
	next := base
	next.items = strings.Replace(base.items, `"Rose Bun",0,1,3,5,7,8,"S"`, `"Rose Bun",0,1,3,5,7,8,"SS"`, 1)
	c := compare(t, base, next)
	if c.FixedItems != 1 || !c.Touches("fixes") {
		t.Errorf("a changed grade: %+v", c)
	}

	next = base
	next.acquire = strings.Replace(base.acquire, "Available from the start", "Clothes Store", 1)
	if c := compare(t, base, next); c.FixedItems != 1 {
		t.Errorf("a changed how-to-get line: %+v", c)
	}

	next = base
	next.items = `{"items":[[10001,"Nikki's Pinky",0,1,3,5,7,8,"S","A","A","A","A",2,""]]}`
	if c := compare(t, base, next); c.FixedItems != 1 {
		t.Errorf("a removed item: %+v", c)
	}

	next = base
	next.stages = `[{"name":"1-1","mode":"Story","weights":[1,2,3,4,6],"attrs":[1,3,5,7,9]}]`
	if c := compare(t, base, next); c.FixedStages != 1 || len(c.NewStages) != 0 {
		t.Errorf("a changed stage: %+v", c)
	}

	next = base
	next.tags = `["Animal","Apron","Army"]`
	if c := compare(t, base, next); !c.FixedOther || !c.Touches("fixes") || c.FixedItems != 0 {
		t.Errorf("a changed tag list: %+v", c)
	}
}

func TestNewStagesAreNamedByModeAndName(t *testing.T) {
	next := base
	next.stages = `[{"name":"1-1","mode":"Story","weights":[1,2,3,4,5],"attrs":[1,3,5,7,9]},
		{"name":"3-4","mode":"Story","weights":[1,2,3,4,5],"attrs":[1,3,5,7,9]},
		{"name":"Beach Party","mode":"Arena","weights":[1,2,3,4,5],"attrs":[1,3,5,7,9]}]`
	c := compare(t, base, next)
	if strings.Join(c.NewStages, "|") != "Story 3-4|Arena Beach Party" {
		t.Errorf("new stages %q", c.NewStages)
	}
}

func TestReadReleaseRefusesAVersionThatLeavesTheDataFolder(t *testing.T) {
	root := writeRoot(t, base)
	if err := os.WriteFile(filepath.Join(root, "data", "index.json"), []byte(`{"version":"../cloud"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRelease(root); err == nil {
		t.Fatal("a version with ../ was accepted")
	}
}

func TestMessagesSayOnlyWhatEachPersonAskedFor(t *testing.T) {
	c := Changes{
		NewItems:   []string{"Rose Gown", "Tranquil Veil", "Plain Socks"},
		NewSuits:   []string{"Metallic Crisis"},
		NewStages:  []string{"Story 3-4"},
		FixedItems: 91,
	}
	cases := []struct {
		topics      []string
		title, body string
	}{
		{[]string{"items"}, "New in NikkiBase", "3 new items, including the Metallic Crisis suit."},
		{[]string{"stages"}, "New in NikkiBase", "New stage: Story 3-4."},
		{[]string{"fixes"}, "NikkiBase update", "Fixes to 91 items."},
		{[]string{"fixes", "items", "stages"}, "New in NikkiBase", "3 new items, including the Metallic Crisis suit. New stage: Story 3-4. Fixes to 91 items."},
	}
	for _, tc := range cases {
		m, ok := c.Message(tc.topics)
		if !ok || m.Title != tc.title || m.Body != tc.body || m.Tag != "nikkibase-data" || m.URL != "/" {
			t.Errorf("%v: %+v, %v", tc.topics, m, ok)
		}
	}
	if _, ok := (Changes{FixedItems: 3}).Message([]string{"items", "stages"}); ok {
		t.Error("a fixes-only release sent a message to someone who asked for new things only")
	}
}

func TestMessageWording(t *testing.T) {
	cases := []struct {
		c    Changes
		want string
	}{
		{Changes{NewItems: []string{"Rose Gown"}, NewSuits: []string{"Metallic Crisis"}}, "New item: Rose Gown."},
		{Changes{NewItems: []string{"a", "b"}}, "2 new items."},
		{Changes{NewItems: []string{"a", "b", "c"}, NewSuits: []string{"X", "Y"}}, "3 new items, including the X and Y suits."},
		{Changes{NewItems: []string{"a", "b", "c", "d"}, NewSuits: []string{"X", "Y", "Z"}}, "4 new items, including the X, Y and 1 more suit."},
		{Changes{NewItems: []string{"a", "b", "c", "d", "e"}, NewSuits: []string{"X", "Y", "Z", "W"}}, "5 new items, including the X, Y and 2 more suits."},
		{Changes{NewStages: []string{"Story 3-4", "Story 3-5", "Arena Beach Party"}}, "New stages: Story 3-4, Story 3-5 and Arena Beach Party."},
		{Changes{NewStages: []string{"a", "b", "c", "d"}}, "4 new stages."},
		{Changes{FixedItems: 1, FixedStages: 2}, "Fixes to 1 item and 2 stages."},
		{Changes{FixedOther: true}, "Fixes to the data."},
	}
	for _, tc := range cases {
		m, ok := tc.c.Message(Topics)
		if !ok || m.Body != tc.want {
			t.Errorf("%+v: %q, want %q", tc.c, m.Body, tc.want)
		}
	}
}
