package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/catalogue"
	"github.com/danielradosa/nikkibase/core/scoring"
)

const version = "2026-09-23"

var everyPair = [5]int8{scoring.Simple, scoring.Lively, scoring.Cute, scoring.Sexy, scoring.Cool}

const stages = `[{"name":"1-1","mode":"Story","weights":[2,1,1,1,1],"attrs":[1,3,5,7,9],"tags":{"7":100},"rules":{"styles":["Unisex"]},` +
	`"variants":{"maiden":{"weights":[3,1,1,1,1],"attrs":[1,3,5,7,9],"tags":{"7":300}}}},` +
	`{"name":"Beach Party","mode":"Arena","weights":[0.5,0,0,0,0],"attrs":[1,3,5,7,9]}]`

func writeBundle(t *testing.T) string {
	t.Helper()
	data := t.TempDir()
	dir := filepath.Join(data, version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	items := catalogue.Write([]catalogue.Item{
		{ID: 10001, Slot: uint8(scoring.Hair), Position: 0, Attrs: everyPair, Stats: [5]int32{10}, Tags: []int32{7}},
		{ID: 20001, Slot: uint8(scoring.Dress), Position: 1, Attrs: everyPair, Stats: [5]int32{30}},
	})
	files := map[string]string{
		filepath.Join(data, "index.json"): `{"version":"` + version + `"}` + "\n",
		filepath.Join(dir, "items.bin"):   string(items),
		filepath.Join(dir, "stages.json"): stages,
	}
	for path, body := range files {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return data
}

func TestWritesEveryStageVersion(t *testing.T) {
	data := writeBundle(t)
	out := filepath.Join(t.TempDir(), "src", "generated", "ideal.json")
	var log bytes.Buffer
	if err := run(data, out, &log); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"version":"2026-09-23","stages":{` +
		`"Story/1-1":{"score":130,"items":[[10001,0],[20001,1]],"auto":{"charmSmile":1,"smile":3,"score":192,"items":[[10001,0],[20001,1]]},` +
		`"variants":{"maiden":{"score":270,"items":[[10001,0],[20001,1]],"auto":{"charmSmile":1,"smile":3,"score":363,"items":[[10001,0],[20001,1]]}}}},` +
		`"Arena/Beach Party":{"score":20,"items":[[10001,0],[20001,1]],"auto":{"charmSmile":1,"smile":-1,"score":35,"items":[[10001,0],[20001,1]]}}}}`
	if string(got) != want {
		t.Errorf("wrote\n%s\nwant\n%s", got, want)
	}
	if !strings.Contains(log.String(), "3 stage versions (2 stages) from bundle 2026-09-23") {
		t.Errorf("summary = %q", log.String())
	}
}

func TestWearsWhatEachStageVersionRequires(t *testing.T) {
	data := writeBundle(t)
	items := catalogue.Write([]catalogue.Item{
		{ID: 10001, Slot: uint8(scoring.Hair), Position: 0, Attrs: everyPair, Stats: [5]int32{10}, Tags: []int32{7}},
		{ID: 10002, Slot: uint8(scoring.Hair), Position: 0, Attrs: everyPair, Stats: [5]int32{1}},
		{ID: 20001, Slot: uint8(scoring.Dress), Position: 1, Attrs: everyPair, Stats: [5]int32{30}},
	})
	required := `[{"name":"1-1","mode":"Story","weights":[2,1,1,1,1],"attrs":[1,3,5,7,9],"tags":{"7":100},"rules":{"styles":[],"require":[[10002]]},` +
		`"variants":{"maiden":{"weights":[3,1,1,1,1],"attrs":[1,3,5,7,9],"tags":{"7":300},"rules":{"styles":[],"require":[[10001]]}}}}]`
	for name, body := range map[string][]byte{"items.bin": items, "stages.json": []byte(required)} {
		if err := os.WriteFile(filepath.Join(data, version, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out := filepath.Join(t.TempDir(), "ideal.json")
	if err := run(data, out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	type outfit struct {
		Items [][2]int `json:"items"`
		Auto  struct {
			Items [][2]int `json:"items"`
		} `json:"auto"`
	}
	var got struct {
		Stages map[string]struct {
			outfit
			Variants map[string]outfit `json:"variants"`
		} `json:"stages"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	stage := got.Stages["Story/1-1"]
	for _, c := range []struct {
		label string
		o     outfit
		hair  int
	}{{"Princess", stage.outfit, 10002}, {"Maiden", stage.Variants["maiden"], 10001}} {
		for _, items := range [][][2]int{c.o.Items, c.o.Auto.Items} {
			if len(items) == 0 || items[0] != [2]int{c.hair, 0} {
				t.Errorf("%s wears %v, want the required hair %d", c.label, items, c.hair)
			}
		}
	}
}

func TestRefusesAMissingBundle(t *testing.T) {
	out := filepath.Join(t.TempDir(), "ideal.json")
	err := run(t.TempDir(), out, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "cmd/bundle") {
		t.Errorf("err = %v, want one pointing at cmd/bundle", err)
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Error("a failed run still wrote the file")
	}
}

func TestRefusesBrokenStages(t *testing.T) {
	data := writeBundle(t)
	broken := `[{"name":"1-1","mode":"Story","weights":[1,2],"attrs":[1,3,5,7,9]}]`
	if err := os.WriteFile(filepath.Join(data, version, "stages.json"), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	err := run(data, filepath.Join(t.TempDir(), "ideal.json"), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "stages.json") || !strings.Contains(err.Error(), "Story/1-1") {
		t.Errorf("err = %v, want one naming stages.json and the stage", err)
	}
}
