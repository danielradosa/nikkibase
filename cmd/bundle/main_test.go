package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/danielradosa/nikkibase/core/catalogue"
	"github.com/danielradosa/nikkibase/pipeline"
)

const (
	sources    = "../../data/sources.json"
	exceptions = "../../data/production-exceptions.json"
)

func testConfig(t *testing.T) config {
	t.Helper()
	return config{
		outDir: t.TempDir(), version: "test", sourcesPath: sources, exceptionsPath: exceptions,
		acquisitionMapPath: "../../data/acquisition-cn.json",
		builtAt:            time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC),
	}
}

func provenance(t *testing.T, c config) pipeline.Provenance {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(c.outDir, c.version, "provenance.json"))
	if err != nil {
		t.Fatalf("provenance.json was not written: %v", err)
	}
	var p pipeline.Provenance
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestBuildsFromLicensedSourcesAlone(t *testing.T) {
	c := testConfig(t)
	c.dumpPath = "testdata/wiki.xml"
	c.exceptionsPath = ""
	if err := run(c); err != nil {
		t.Fatalf("a wiki-only build failed: %v", err)
	}
	for _, f := range []string{"items.bin", "items.json", "stages.json", "tags.json", "provenance.json", "acquire.json"} {
		if _, err := os.Stat(filepath.Join(c.outDir, c.version, f)); err != nil {
			t.Errorf("%s missing: %v", f, err)
		}
	}
	raw, err := os.ReadFile(filepath.Join(c.outDir, c.version, "acquire.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"version":"test","items":{"10001":[{"k":"store","t":"Clothes Store"},{"k":"event","t":"Ribbon Party event","past":1}]}}`; string(raw) != want {
		t.Errorf("acquire.json = %s, want %s", raw, want)
	}
	p := provenance(t, c)
	if p.Mode != "production" || p.BuiltAt != "2026-09-23T00:00:00Z" || p.Version != "test" {
		t.Errorf("provenance header = %+v", p)
	}
	for _, s := range p.Sources {
		if s.Included != (s.ID == "love-nikki-wiki") {
			t.Errorf("%s included = %v", s.ID, s.Included)
		}
	}
	if err := runCheck(filepath.Join(c.outDir, c.version, "provenance.json"), sources, ""); err != nil {
		t.Errorf("the deploy check refused a licensed-only bundle: %v", err)
	}
}

func TestUnlicensedSourceFailsWithoutFlag(t *testing.T) {
	c := testConfig(t)
	c.dumpPath, c.stagesPath = "testdata/wiki.xml", "testdata/levels.js"
	c.exceptionsPath = ""
	err := run(c)
	if err == nil || !strings.Contains(err.Error(), "seal100x-nikkiup2u3-data") {
		t.Fatalf("err = %v, want a refusal naming the stage source", err)
	}
	if _, statErr := os.Stat(filepath.Join(c.outDir, c.version)); !os.IsNotExist(statErr) {
		t.Error("a refused build still wrote a bundle directory")
	}
}

func TestAllowUnlicensedBuildsResearchBundle(t *testing.T) {
	c := testConfig(t)
	c.dumpPath, c.stagesPath = "testdata/wiki.xml", "testdata/levels.js"
	c.exceptionsPath, c.allowUnlicensed = "", true
	if err := run(c); err != nil {
		t.Fatal(err)
	}
	if p := provenance(t, c); p.Mode != "research" {
		t.Errorf("mode = %q, want research", p.Mode)
	}
	if err := runCheck(filepath.Join(c.outDir, c.version, "provenance.json"), sources, exceptions); err == nil {
		t.Error("the deploy check accepted a research bundle")
	}
}

func TestProvenanceIdentifiesEveryIncludedSource(t *testing.T) {
	c := testConfig(t)
	c.dumpPath, c.stagesPath = "testdata/wiki.xml", "testdata/levels.js"
	if err := run(c); err != nil {
		t.Fatal(err)
	}
	want := map[string]pipeline.Basis{"love-nikki-wiki": pipeline.ByLicence, "seal100x-nikkiup2u3-data": pipeline.ByException}
	for _, s := range provenance(t, c).Sources {
		basis, in := want[s.ID]
		if s.Included != in || (in && s.Basis != basis) {
			t.Errorf("%s: included=%v basis=%q, want included=%v basis=%q", s.ID, s.Included, s.Basis, in, basis)
		}
		if in && (len(s.Inputs) != 1 || len(s.Inputs[0].SHA256) != 64) {
			t.Errorf("%s: inputs = %+v, want one hashed file", s.ID, s.Inputs)
		}
		if !in && !strings.Contains(s.Summary, "not included in production") {
			t.Errorf("%s: excluded source summary = %q", s.ID, s.Summary)
		}
	}
	if err := runCheck(filepath.Join(c.outDir, c.version, "provenance.json"), sources, exceptions); err != nil {
		t.Errorf("deploy check: %v", err)
	}
}

func TestCheckRefusesMissingProvenance(t *testing.T) {
	if err := runCheck(filepath.Join(t.TempDir(), "provenance.json"), sources, exceptions); err == nil {
		t.Error("a bundle without provenance passed the deploy check")
	}
}

func TestStageSourcesAreLayeredAndRecorded(t *testing.T) {
	c := testConfig(t)
	c.dumpPath, c.stagesPath = "testdata/wiki.xml", "testdata/stages.js"
	c.stageValuesPath, c.stageNamesPath = "testdata/values.js", "testdata/levels.js"
	c.coveragePath = filepath.Join(t.TempDir(), "coverage.json")
	if err := os.WriteFile(c.coveragePath, []byte(`{"maxUnwornTags": 1, "valuedStages": 1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(c); err != nil {
		t.Fatal(err)
	}
	inputs := map[string]string{}
	for _, s := range provenance(t, c).Sources {
		if s.Included {
			if s.Basis != pipeline.ByLicence && s.Basis != pipeline.ByPermission && s.Basis != pipeline.ByException {
				t.Errorf("%s included on basis %q", s.ID, s.Basis)
			}
			for _, in := range s.Inputs {
				inputs[in.File] = s.ID
			}
		}
	}
	for file, id := range map[string]string{
		"stages.js": "seal100x-nikkiup2u3-data", "values.js": "aojiao-nikkiup2u3", "levels.js": "lovenikkiusa-nikkiup2u",
	} {
		if inputs[file] != id {
			t.Errorf("provenance records %s under %q, want %q", file, inputs[file], id)
		}
	}

	raw, err := os.ReadFile(filepath.Join(c.outDir, c.version, "stages.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stages []struct {
		Name    string         `json:"name"`
		Mode    string         `json:"mode"`
		Weights []float64      `json:"weights"`
		Tags    map[string]int `json:"tags"`
	}
	if err := json.Unmarshal(raw, &stages); err != nil {
		t.Fatal(err)
	}
	animal, _ := pipeline.TagID("Animal")
	if len(stages) != 2 || stages[0].Name != "1-1" || stages[1].Name != "Beach Party" {
		t.Fatalf("stages = %+v, want 1-1 and Beach Party", stages)
	}
	if got := stages[0].Tags[strconv.Itoa(animal)]; got != 2680 || stages[0].Weights[1] != 44 {
		t.Errorf("1-1 = %+v, want the exact source's weights and an Animal award of 2680", stages[0])
	}
}

func TestAcquisitionIsHeldToItsFloor(t *testing.T) {
	c := testConfig(t)
	c.dumpPath = "testdata/wiki.xml"
	c.coveragePath = filepath.Join(t.TempDir(), "coverage.json")
	if err := os.WriteFile(c.coveragePath, []byte(`{"acquisitionItems": 2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(c); err == nil || !strings.Contains(err.Error(), "1 items say how to get them") {
		t.Errorf("err = %v, want a refusal naming the acquisition floor", err)
	}
	if _, statErr := os.Stat(filepath.Join(c.outDir, c.version)); !os.IsNotExist(statErr) {
		t.Error("a refused build still wrote a bundle directory")
	}
}

func TestStageLayersNeedStages(t *testing.T) {
	for _, set := range []func(*config){
		func(c *config) { c.stageValuesPath = "testdata/values.js" },
		func(c *config) { c.stageNamesPath = "testdata/levels.js" },
	} {
		c := testConfig(t)
		c.dumpPath = "testdata/wiki.xml"
		set(&c)
		if err := run(c); err == nil || !strings.Contains(err.Error(), "-stages") {
			t.Errorf("err = %v, want a refusal naming -stages", err)
		}
	}
}

func TestStageRulesReachStagesJSON(t *testing.T) {
	c := testConfig(t)
	c.dumpPath, c.stagesPath, c.stageNamesPath = "testdata/wiki.xml", "testdata/stages.js", "testdata/levels.js"
	dir := t.TempDir()
	c.coveragePath = filepath.Join(dir, "coverage.json")
	c.stageRulesPath = filepath.Join(dir, "stage-rules.json")
	rules := `{"rules": [{"stage": "1-1", "mode": "Story", "require": [[10001], [20002]],
		"source": "wiki quest field", "quote": "[[Test Ribbon Hair]] and [[Test Plain Dress]]"}]}`
	for path, doc := range map[string]string{c.coveragePath: `{"maxUnwornTags": 1}`, c.stageRulesPath: rules} {
		if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := run(c); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(c.outDir, c.version, "stages.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := `"rules":{"styles":[],"require":[[10001],[20002]]}`; !strings.Contains(string(raw), want) {
		t.Errorf("stages.json lacks %s:\n%s", want, raw)
	}

	c.outDir = t.TempDir()
	if err := os.WriteFile(c.stageRulesPath, []byte(strings.Replace(rules, "[[10001]", "[[10009]", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(c); err == nil || !strings.Contains(err.Error(), "10009") {
		t.Errorf("err = %v, want a refusal naming the item the catalogue lacks", err)
	}
	if _, statErr := os.Stat(filepath.Join(c.outDir, c.version)); !os.IsNotExist(statErr) {
		t.Error("a refused build still wrote a bundle directory")
	}
}

func subgradeConfig(t *testing.T, coverage string) config {
	t.Helper()
	c := testConfig(t)
	c.dumpPath, c.keysPath = "testdata/wiki.xml", "testdata/keys.json"
	c.subgradesPath = "testdata/subgrades/item-batch-*.json"
	c.coveragePath = filepath.Join(t.TempDir(), "coverage.json")
	if err := os.WriteFile(c.coveragePath, []byte(coverage), 0o644); err != nil {
		t.Fatal(err)
	}
	return c
}

func readCatalogue(t *testing.T, c config) *catalogue.Catalogue {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(c.outDir, c.version, "items.bin"))
	if err != nil {
		t.Fatal(err)
	}
	cat, err := catalogue.Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func statsByID(cat *catalogue.Catalogue) map[int32][5]int32 {
	out := map[int32][5]int32{}
	for i, id := range cat.IDs {
		out[id] = [5]int32(cat.Stats[i*5 : i*5+5])
	}
	return out
}

func TestSubgradesChangeOnlyStatsWithinTheirLetterAndSide(t *testing.T) {
	plain := testConfig(t)
	plain.dumpPath = "testdata/wiki.xml"
	if err := run(plain); err != nil {
		t.Fatal(err)
	}
	sub := subgradeConfig(t, `{"subgradedCells": 8, "maxSubgradeFallbacks": 2}`)
	if err := run(sub); err != nil {
		t.Fatal(err)
	}

	for _, f := range []string{"items.json", "stages.json", "tags.json", "positions.json"} {
		a, errA := os.ReadFile(filepath.Join(plain.outDir, plain.version, f))
		b, errB := os.ReadFile(filepath.Join(sub.outDir, sub.version, f))
		if errA != nil || errB != nil || !bytes.Equal(a, b) {
			t.Errorf("%s differs between the builds with and without -subgrades (%v, %v)", f, errA, errB)
		}
	}
	without, with := readCatalogue(t, plain), readCatalogue(t, sub)
	for id, want := range map[int32]struct{ without, with [5]int32 }{
		10001: {[5]int32{70, 56, 56, 56, 56}, [5]int32{64, 63, 57, 56, 57}},
		20002: {[5]int32{348, 225, 175, 109, 225}, [5]int32{400, 206, 163, 109, 225}},
	} {
		if got := statsByID(without)[id]; got != want.without {
			t.Errorf("%d without -subgrades: stats %v, want the letter's %v", id, got, want.without)
		}
		if got := statsByID(with)[id]; got != want.with {
			t.Errorf("%d with -subgrades: stats %v, want %v", id, got, want.with)
		}
	}
	without.Stats, with.Stats = nil, nil
	if !reflect.DeepEqual(without, with) {
		t.Errorf("items.bin differs beyond stats:\n%+v\n%+v", without, with)
	}
}

func TestSubgradeBatchesAreHashedUnderNikkiCalc(t *testing.T) {
	c := subgradeConfig(t, `{"subgradedCells": 8, "maxSubgradeFallbacks": 2}`)
	if err := run(c); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range provenance(t, c).Sources {
		if s.ID != "nikki-calc" {
			continue
		}
		found = true
		var files []string
		for _, in := range s.Inputs {
			if len(in.SHA256) != 64 {
				t.Errorf("%s: hash %q", in.File, in.SHA256)
			}
			files = append(files, in.File)
		}
		if want := []string{"keys.json", "item-batch-0.json", "item-batch-500.json"}; !slices.Equal(files, want) {
			t.Errorf("nikki-calc inputs %v, want %v", files, want)
		}
		if !s.Included || s.Basis != pipeline.ByPermission || s.Exception != nil || !slices.Contains(s.Flags, "subgrades") {
			t.Errorf("nikki-calc: included=%v basis=%q flags=%v", s.Included, s.Basis, s.Flags)
		}
	}
	if !found {
		t.Error("provenance has no nikki-calc record")
	}
	if err := runCheck(filepath.Join(c.outDir, c.version, "provenance.json"), sources, exceptions); err != nil {
		t.Errorf("deploy check: %v", err)
	}
}

func TestSubgradesAreRefusedOutsideTheirFloorAndCeiling(t *testing.T) {
	for coverage, want := range map[string]string{
		`{"subgradedCells": 9, "maxSubgradeFallbacks": 2}`: "committed floor is 9",
		`{"subgradedCells": 8, "maxSubgradeFallbacks": 1}`: "committed ceiling is 1",
	} {
		c := subgradeConfig(t, coverage)
		if err := run(c); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v, want one naming %q", coverage, err, want)
		}
		if _, statErr := os.Stat(filepath.Join(c.outDir, c.version)); !os.IsNotExist(statErr) {
			t.Errorf("%s: a refused build still wrote a bundle directory", coverage)
		}
	}
	for want, set := range map[string]func(*config){
		"-keys":            func(c *config) { c.keysPath = "" },
		"matches no files": func(c *config) { c.subgradesPath = "testdata/subgrades/none-*.json" },
	} {
		c := subgradeConfig(t, `{}`)
		set(&c)
		if err := run(c); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("err = %v, want one naming %q", err, want)
		}
	}
}

func itemRows(t *testing.T, c config) map[int][]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(c.outDir, c.version, "items.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Items [][]any `json:"items"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	out := map[int][]any{}
	for _, row := range doc.Items {
		out[int(row[0].(float64))] = row
	}
	return out
}

func evolvedConfig(t *testing.T) config {
	t.Helper()
	c := testConfig(t)
	raw, err := os.ReadFile("testdata/wiki.xml")
	if err != nil {
		t.Fatal(err)
	}
	pages := "<page>\n  <title>Honey-Soaked Song</title>\n  <ns>0</ns>\n  <revision><text>{{Clothing\n" +
		"|type = Hair\n|wardrobe nr = 2\n}}\n" +
		"{{Attributes|Gorgeous|A|Elegant|B|Mature|C|Sexy|S|Warm|A}}</text></revision>\n</page>\n" +
		"<page>\n  <title>Test Evolved Hair</title>\n  <ns>0</ns>\n  <revision><text>{{Clothing\n" +
		"|type = Hair\n|wardrobe nr = 3\n|how to obtain = [[Evolution]]\n}}\n" +
		"{{Attributes|Simple|A|Lively|A|Cute|A|Pure|A|Warm|A}}\n" +
		"== Evolution ==\n=== Evolved from: ===\n* {{IconItem|Honey-Soaked Song|quantity=2}}</text></revision>\n</page>\n"
	c.dumpPath = filepath.Join(t.TempDir(), "wiki.xml")
	dump := strings.Replace(string(raw), "</mediawiki>", pages+"</mediawiki>", 1)
	if err := os.WriteFile(c.dumpPath, []byte(dump), 0o644); err != nil {
		t.Fatal(err)
	}
	c.namesPath, c.keysPath = "testdata/names.json", "testdata/names-keys.json"
	return c
}

func TestNikkiCalcSpellingKeepsHyphensJoined(t *testing.T) {
	c := evolvedConfig(t)
	if err := run(c); err != nil {
		t.Fatal(err)
	}
	if got := itemRows(t, c)[10002][1]; got != "Honey-Soaked Song" {
		t.Errorf("items.json names 10002 %q, want Honey-Soaked Song as Nikki Calc spells it", got)
	}
	acq, err := os.ReadFile(filepath.Join(c.outDir, c.version, "acquire.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := `"10003":[{"k":"evolve","t":"Evolve: 2× Honey-Soaked Song","from":[[10002,2]]}]`; !strings.Contains(string(acq), want) {
		t.Errorf("acquire.json = %s, want it to hold %s", acq, want)
	}
}

func TestWikiSpellingIsSpacedWithoutNikkiCalc(t *testing.T) {
	c := evolvedConfig(t)
	c.namesPath, c.keysPath = "", ""
	if err := run(c); err != nil {
		t.Fatal(err)
	}
	if got := itemRows(t, c)[10002][1]; got != "Honey - Soaked Song" {
		t.Errorf("items.json names 10002 %q, want Honey - Soaked Song with no source spelling it joined", got)
	}
}

func TestNikkiCalcGivesEveryItemItsRarity(t *testing.T) {
	c := evolvedConfig(t)
	if err := run(c); err != nil {
		t.Fatal(err)
	}
	rows := itemRows(t, c)
	for id, want := range map[int]float64{10001: 3, 20002: 5, 30003: 4, 10002: 1} {
		if got := rows[id][13]; got != want {
			t.Errorf("%d: rarity %v, want %v", id, got, want)
		}
	}
}

func TestBlankRarityRefusesTheBuild(t *testing.T) {
	c := testConfig(t)
	dir := t.TempDir()
	c.dumpPath, c.keysPath = "testdata/wiki.xml", "testdata/names-keys.json"
	c.namesPath, c.knownPath = filepath.Join(dir, "names.json"), filepath.Join(dir, "known.json")
	c.allowUnlicensed = true
	for path, doc := range map[string]string{
		c.namesPath: `["001", 0, 0, 3, "Test Ribbon Hair"]`, c.knownPath: `[10001, 20002]`,
	} {
		if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := run(c); err == nil || !strings.Contains(err.Error(), "20002 (Test Plain Dress) rarity 0") {
		t.Errorf("err = %v, want a refusal naming 20002's blank rarity", err)
	}
	if _, statErr := os.Stat(filepath.Join(c.outDir, c.version)); !os.IsNotExist(statErr) {
		t.Error("a refused build still wrote a bundle directory")
	}
}

func withCorrections(t *testing.T, c config, doc string) config {
	t.Helper()
	c.idCorrectionsPath = filepath.Join(t.TempDir(), "id-corrections.json")
	if err := os.WriteFile(c.idCorrectionsPath, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestDisplayNamesChangeOnlyWhatIsPrinted(t *testing.T) {
	c := withCorrections(t, evolvedConfig(t), `{"displayNames": [
		{"id": 10002, "name": "Honey-Soaked Ballad", "was": "Honey-Soaked Song", "basis": "b"}]}`)
	if err := run(c); err != nil {
		t.Fatal(err)
	}
	if got := itemRows(t, c)[10002][1]; got != "Honey-Soaked Ballad" {
		t.Errorf("items.json names 10002 %q, want Honey-Soaked Ballad", got)
	}
	acq, err := os.ReadFile(filepath.Join(c.outDir, c.version, "acquire.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := `"10003":[{"k":"evolve","t":"Evolve: 2× Honey-Soaked Ballad","from":[[10002,2]]}]`; !strings.Contains(string(acq), want) {
		t.Errorf("acquire.json = %s, want it to hold %s", acq, want)
	}
}

func TestDisplayNamesMustMatchWhatTheSourcesGive(t *testing.T) {
	for doc, want := range map[string]string{
		`{"displayNames": [{"id": 10002, "name": "Honey-Soaked Ballad", "was": "Honey Soaked Song", "basis": "b"}]}`: "10002",
		`{"displayNames": [{"id": 10009, "name": "Gone Again", "was": "Gone", "basis": "b"}]}`:                       "10009",
	} {
		c := withCorrections(t, evolvedConfig(t), doc)
		if err := run(c); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("err = %v, want a refusal naming %s", err, want)
		}
		if _, statErr := os.Stat(filepath.Join(c.outDir, c.version)); !os.IsNotExist(statErr) {
			t.Error("a refused build still wrote a bundle directory")
		}
	}
}

func packedConfig(t *testing.T, pages string) config {
	t.Helper()
	c := testConfig(t)
	raw, err := os.ReadFile("testdata/wiki.xml")
	if err != nil {
		t.Fatal(err)
	}
	c.dumpPath = filepath.Join(t.TempDir(), "wiki.xml")
	dump := strings.Replace(string(raw), "</mediawiki>", pages+"</mediawiki>", 1)
	if err := os.WriteFile(c.dumpPath, []byte(dump), 0o644); err != nil {
		t.Fatal(err)
	}
	c.packedPath, c.acquisitionMapPath = "testdata/wardrobe.js", "testdata/acquisition-cn.json"
	c.namesPath, c.keysPath = "testdata/names.json", "testdata/names-keys.json"
	return c
}

func grades(row []any) [5]any {
	return [5]any(row[8:13])
}

func TestPackedTableGoesUnderTheWiki(t *testing.T) {
	c := packedConfig(t, "")
	if err := run(c); err != nil {
		t.Fatal(err)
	}
	rows := itemRows(t, c)
	if got, want := grades(rows[10001]), [5]any{"S", "A", "A", "A", "A"}; got != want {
		t.Errorf("10001 grades %v, want the wiki's %v over the packed table's", got, want)
	}
	if got, want := grades(rows[30003]), [5]any{"C", "C", "C", "C", "C"}; got != want || rows[30003][1] != "Test-only Coat" {
		t.Errorf("30003 = %v, want the packed table's grades under Nikki Calc's name", rows[30003])
	}
	for _, p := range provenance(t, c).Sources {
		if p.ID == "aojiao-nikkiup2u3" && !p.Included {
			t.Error("the packed table is not recorded as included")
		}
	}
}

func TestMisnumberedWikiPageNeedsASettlement(t *testing.T) {
	page := "<page>\n  <title>Wrong Page Hair</title>\n  <ns>0</ns>\n  <revision><text>{{Clothing\n" +
		"|type = Hair\n|wardrobe nr = 2\n}}\n" +
		"{{Attributes|Simple|SS|Lively|SS|Cute|SS|Pure|SS|Warm|SS}}</text></revision>\n</page>\n"
	c := packedConfig(t, page)
	err := run(c)
	if err == nil || !strings.Contains(err.Error(), "packed table and Nikki Calc") || !strings.Contains(err.Error(), "10002") {
		t.Fatalf("err = %v, want the misnumbered page at 10002 refused", err)
	}
	if _, statErr := os.Stat(filepath.Join(c.outDir, c.version)); !os.IsNotExist(statErr) {
		t.Error("a refused build still wrote a bundle directory")
	}
	c = withCorrections(t, packedConfig(t, page), `{"duplicates": [{"id": 10002, "keep": "Honey-Soaked Song",
		"drop": "Wrong Page Hair", "basis": "b"}]}`)
	if err := run(c); err != nil {
		t.Fatalf("a settled page still fails: %v", err)
	}
	row := itemRows(t, c)[10002]
	if got, want := grades(row), [5]any{"C", "C", "C", "C", "C"}; got != want || row[1] != "Honey-Soaked Song" {
		t.Errorf("10002 = %v, want the packed row under Nikki Calc's name", row)
	}
}

func TestSuitsComeFromTheWikiThenThePackedTable(t *testing.T) {
	page := "<page>\n  <title>Test Suit</title>\n  <ns>0</ns>\n  <revision><text>{{Suit Infobox\n" +
		"|type = Collection Suit\n|cnwiki = 测试套装\n|how to obtain = [[Recharge]]\n}}\n==Wardrobe==\n" +
		"{{Suit Part|Test Ribbon Hair|type=Hair}}\n{{Suit Part|Test Evolved Hair|type=Hair}}</text></revision>\n</page>\n"
	c := packedConfig(t, page)
	if err := run(c); err != nil {
		t.Fatal(err)
	}
	rows := itemRows(t, c)
	for id, want := range map[int]string{10001: "Test Suit", 10002: "Test Suit", 10003: "Test Suit", 20002: "", 30003: ""} {
		if got := rows[id][14]; got != want {
			t.Errorf("%d: suit %q, want %q", id, got, want)
		}
	}
	for _, p := range provenance(t, c).Sources {
		if (p.ID == "love-nikki-wiki" || p.ID == "aojiao-nikkiup2u3") &&
			!slices.ContainsFunc(p.Fields, func(f string) bool { return strings.Contains(f, "suit each item") }) {
			t.Errorf("%s's recorded fields %v do not say it gives items their suit", p.ID, p.Fields)
		}
	}
}

func TestSuitsAreHeldToTheirFloor(t *testing.T) {
	c := packedConfig(t, "")
	c.coveragePath = filepath.Join(t.TempDir(), "coverage.json")
	if err := os.WriteFile(c.coveragePath, []byte(`{"suitItems": 1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	err := run(c)
	if err == nil || !strings.Contains(err.Error(), "suit") {
		t.Fatalf("err = %v, want a build with no item in a suit refused", err)
	}
}
