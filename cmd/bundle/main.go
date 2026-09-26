package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/danielradosa/nikkibase/pipeline"
)

type config struct {
	itemsPath, dumpPath, knownPath, stagesPath            string
	stageValuesPath, stageNamesPath                       string
	stageScopePath, stageDifficultyPath, stageRulesPath   string
	packedPath, namesPath, keysPath, subgradesPath        string
	outDir, version                                       string
	sourcesPath, exceptionsPath                           string
	idCorrectionsPath, stageCorrectionsPath, coveragePath string
	acquisitionMapPath                                    string
	allowUnlicensed                                       bool
	builtAt                                               time.Time
}

func main() {
	var c config
	flag.StringVar(&c.itemsPath, "items", "", "path to lovenikkiusa's wardrobe.js, read only without -fandom")
	flag.StringVar(&c.dumpPath, "fandom", "", "path to a Fandom MediaWiki XML dump (preferred over -items)")
	flag.StringVar(&c.knownPath, "known", "", "path to a JSON array of valid game IDs, used to reject mistyped entries")
	flag.StringVar(&c.stagesPath, "stages", "", "path to the stage source's levels.js: the stage list, modes and rules, and weights and tag awards where -stage-values has none")
	flag.StringVar(&c.stageValuesPath, "stage-values", "", "path to a levels.js whose exact weights and tag awards replace -stages' on every stage it carries")
	flag.StringVar(&c.stageNamesPath, "stage-names", "", "path to a bilingual levels.js giving English names to stages -stages names in Chinese only")
	flag.StringVar(&c.stageScopePath, "stage-scope", "data/stage-scope.json", "the stages released on the Global server; empty keeps every stage")
	flag.StringVar(&c.stageDifficultyPath, "stage-difficulty", "data/stage-difficulty.json", "story stages whose Maiden numbers differ from Princess's; empty for none")
	flag.StringVar(&c.stageRulesPath, "stage-rules", "data/stage-rules.json", "the items particular stages require; empty for none")
	flag.StringVar(&c.packedPath, "packed", "", "path to the packed Chinese item table (aojiao wardrobe.js)")
	flag.StringVar(&c.namesPath, "names", "", "path to Nikki Calc's items JSON, for names the dump lacks and rarity no other source gives")
	flag.StringVar(&c.keysPath, "keys", "", "path to Nikki Calc's item-key JSON, required with -names and -subgrades")
	flag.StringVar(&c.subgradesPath, "subgrades", "", "glob of Nikki Calc's item-batch files, whose sub-grades set each stat within its letter grade, e.g. .../item-batch-v0.14-*.json")
	flag.StringVar(&c.outDir, "out", "web/public/data", "directory to write the bundle into")
	flag.StringVar(&c.version, "version", "", "version directory name, e.g. 2026.09")
	flag.StringVar(&c.sourcesPath, "sources", "data/sources.json", "the source registry")
	flag.StringVar(&c.idCorrectionsPath, "id-corrections", "data/id-corrections.json", "resolutions for item IDs that name two garments")
	flag.StringVar(&c.stageCorrectionsPath, "stage-corrections", "data/stage-corrections.json", "stage weight vectors the source states in the wrong unit")
	flag.StringVar(&c.coveragePath, "coverage", "data/coverage.json", "the coverage floor a bundle must clear")
	flag.StringVar(&c.exceptionsPath, "exceptions", "data/production-exceptions.json", "recorded exceptions for sources without a licence; empty for none")
	flag.BoolVar(&c.allowUnlicensed, "allow-unlicensed", false, "admit any source, marking the bundle research-only; deploy.sh refuses such a bundle")
	check := flag.String("check", "", "verify a built bundle's provenance.json may be deployed, then exit")
	flag.Parse()
	c.acquisitionMapPath = "data/acquisition-cn.json"

	var err error
	if *check != "" {
		err = runCheck(*check, c.sourcesPath, c.exceptionsPath)
	} else {
		c.builtAt, err = buildTime()
		if err == nil {
			err = run(c)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "bundle:", err)
		os.Exit(1)
	}
}

func buildTime() (time.Time, error) {
	if v := os.Getenv("SOURCE_DATE_EPOCH"); v != "" {
		sec, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("SOURCE_DATE_EPOCH: %w", err)
		}
		return time.Unix(sec, 0).UTC(), nil
	}
	return time.Now().UTC(), nil
}

func run(c config) error {
	if c.version == "" {
		return fmt.Errorf("a -version is required; it names an immutable directory")
	}
	if (c.stageValuesPath != "" || c.stageNamesPath != "") && c.stagesPath == "" {
		return fmt.Errorf("-stage-values and -stage-names refine the stages -stages reads, so they need -stages")
	}
	if c.subgradesPath != "" && c.keysPath == "" {
		return fmt.Errorf("-subgrades needs -keys, which maps each item-batch record to its item")
	}
	reg, ex, err := readTerms(c.sourcesPath, c.exceptionsPath)
	if err != nil {
		return err
	}
	used, inputs, err := sourcesInUse(c, reg)
	if err != nil {
		return err
	}
	basis, err := pipeline.Gate(used, ex, c.allowUnlicensed)
	if err != nil {
		return err
	}
	if c.allowUnlicensed {
		fmt.Println("RESEARCH BUNDLE: -allow-unlicensed is set; this bundle must not be deployed")
	}
	prov := pipeline.NewProvenance(c.version, c.builtAt.Format(time.RFC3339), reg, ex, basis, inputs, c.allowUnlicensed)

	itemsPath, dumpPath, knownPath := c.itemsPath, c.dumpPath, c.knownPath
	packedPath, namesPath, keysPath := c.packedPath, c.namesPath, c.keysPath
	names := map[int]string{}
	var rarity map[int]int
	if namesPath != "" {
		if keysPath == "" {
			return fmt.Errorf("-names needs -keys, which maps item keys to the name array")
		}
		rawNames, err := os.ReadFile(namesPath)
		if err != nil {
			return err
		}
		rawKeys, err := os.ReadFile(keysPath)
		if err != nil {
			return err
		}
		if names, rarity, err = pipeline.ParseNikkicalc(rawNames, rawKeys); err != nil {
			return err
		}
	}
	known, err := readKnown(knownPath)
	if err != nil {
		return err
	}
	if len(known) == 0 {
		known = make(map[int]bool, len(names))
		for id := range names {
			known[id] = true
		}
	}
	var corrections *pipeline.IDCorrections
	if c.idCorrectionsPath != "" {
		raw, err := os.ReadFile(c.idCorrectionsPath)
		if err != nil {
			return err
		}
		if corrections, err = pipeline.ReadIDCorrections(raw); err != nil {
			return err
		}
	}
	entries, skipped, err := readItems(itemsPath, dumpPath, known)
	if err != nil {
		return err
	}
	if err := pipeline.Canonicalise(entries); err != nil {
		return fmt.Errorf("items: %w", err)
	}
	if corrections != nil {
		entries = corrections.DropDuplicates(entries)
	}
	var packedPlaces map[int]pipeline.Placed
	if packedPath != "" {
		raw, err := os.ReadFile(packedPath)
		if err != nil {
			return err
		}
		packed, pstats, err := pipeline.ParsePacked(raw, known)
		if err != nil {
			return err
		}
		if err := pipeline.Canonicalise(packed); err != nil {
			return fmt.Errorf("packed: %w", err)
		}
		packedPlaces = pipeline.PlacesOf(packed)
		before := len(entries)
		entries = pipeline.MergeEntries(entries, packed)
		fmt.Printf("packed: %d rows, %d parsed, %d rejected, %d not in the global game, %d spirit bonuses, %d unknown tags -> catalogue %d to %d\n",
			pstats.Rows, pstats.Parsed, pstats.UnknownAny, pstats.NotGlobal, pstats.Bonuses,
			pstats.UnknownTag, before, len(entries))
		if dumpPath != "" {
			if err := checkSpiritBonuses(dumpPath, known, packed); err != nil {
				return err
			}
		}
	}
	if corrections != nil {
		before := len(entries)
		if entries, err = corrections.Apply(entries); err != nil {
			return err
		}
		fmt.Printf("identity: %d duplicate IDs resolved, %d slots restored, %d names corrected, %d items left out -> catalogue %d to %d\n",
			len(corrections.Owner), len(corrections.Slot), len(corrections.Name), len(corrections.Excluded), before, len(entries))
	}
	if packedPlaces != nil && namesPath != "" {
		if wrong := pipeline.CheckGarments(entries, packedPlaces, names); len(wrong) > 0 {
			return pipeline.Violations{fmt.Sprintf(
				"%d item IDs carry a different garment from the one the packed table and Nikki Calc name; settle each in %s: %s",
				len(wrong), c.idCorrectionsPath, strings.Join(wrong, "; "))}
		}
	}
	var sub *pipeline.SubgradeStats
	if c.subgradesPath != "" {
		if sub, err = applySubgrades(c.subgradesPath, keysPath, entries); err != nil {
			return err
		}
	}
	if namesPath != "" {
		filled := pipeline.FillRarity(entries, names, rarity)
		if v := pipeline.CheckRarity(entries, names, rarity); len(v) > 0 {
			return v
		}
		fmt.Printf("rarity: %d items with no rarity from the other sources take Nikki Calc's\n", filled)
	}
	itemNames := pipeline.ItemNames{Calc: names}
	if corrections != nil {
		if itemNames.Shown, err = corrections.DisplayNames(entries, itemNames); err != nil {
			return err
		}
		fmt.Printf("names: %d items shown under a corrected name\n", len(itemNames.Shown))
	}
	return finish(c, entries, skipped, itemNames, rarity, prov, sub, known, corrections)
}

func applySubgrades(glob, keysPath string, entries []pipeline.Entry) (*pipeline.SubgradeStats, error) {
	paths, err := subgradeFiles(glob)
	if err != nil {
		return nil, err
	}
	batches := make([][]byte, 0, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		batches = append(batches, raw)
	}
	keys, err := os.ReadFile(keysPath)
	if err != nil {
		return nil, err
	}
	subs, stats, err := pipeline.ParseSubgrades(batches, keys)
	if err != nil {
		return nil, err
	}
	pipeline.ApplySubgrades(entries, subs, &stats)
	fmt.Printf("subgrades: %d files, %d records, %d without a key, %d not in the catalogue -> %d stats set by sub-grade; %d keep their letter's stat: %d on the other side, %d under another letter, %d with no sub-grade\n",
		stats.Files, stats.Records, stats.Unkeyed, stats.NotInCatalogue, stats.Applied,
		stats.OtherSide+stats.OtherLetter+stats.Missing, stats.OtherSide, stats.OtherLetter, stats.Missing)
	return &stats, nil
}

func readTerms(sourcesPath, exceptionsPath string) (*pipeline.Registry, *pipeline.Exceptions, error) {
	raw, err := os.ReadFile(sourcesPath)
	if err != nil {
		return nil, nil, fmt.Errorf("the source registry is required: %w", err)
	}
	reg, err := pipeline.ReadRegistry(raw)
	if err != nil {
		return nil, nil, err
	}
	if exceptionsPath == "" {
		return reg, nil, nil
	}
	raw, err = os.ReadFile(exceptionsPath)
	if err != nil {
		return nil, nil, err
	}
	ex, err := pipeline.ReadExceptions(raw, reg)
	if err != nil {
		return nil, nil, err
	}
	return reg, ex, nil
}

func sourcesInUse(c config, reg *pipeline.Registry) ([]pipeline.Source, map[string][]pipeline.Input, error) {
	given := []struct{ flag, path string }{
		{"fandom", c.dumpPath}, {"items", c.itemsPath}, {"known", c.knownPath},
		{"stages", c.stagesPath}, {"stage-values", c.stageValuesPath}, {"stage-names", c.stageNamesPath},
		{"packed", c.packedPath},
		{"names", c.namesPath}, {"keys", c.keysPath}, {"subgrades", c.subgradesPath},
	}
	var used []pipeline.Source
	inputs := map[string][]pipeline.Input{}
	for _, g := range given {
		if g.path == "" {
			continue
		}
		src, ok := reg.ForFlag(g.flag)
		if !ok {
			return nil, nil, fmt.Errorf("-%s has no source in the registry; record where its data comes from first", g.flag)
		}
		if !slices.ContainsFunc(used, func(s pipeline.Source) bool { return s.ID == src.ID }) {
			used = append(used, src)
		}
		paths := []string{g.path}
		var err error
		if g.flag == "subgrades" {
			paths, err = subgradeFiles(g.path)
		}
		if err != nil {
			return nil, nil, err
		}
		for _, p := range paths {
			in, err := hashFile(p)
			if err != nil {
				return nil, nil, err
			}
			inputs[src.ID] = append(inputs[src.ID], in)
		}
	}
	return used, inputs, nil
}

func hashFile(path string) (pipeline.Input, error) {
	f, err := os.Open(path)
	if err != nil {
		return pipeline.Input{}, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return pipeline.Input{}, err
	}
	return pipeline.Input{File: filepath.Base(path), SHA256: hex.EncodeToString(h.Sum(nil)), Bytes: n}, nil
}

func subgradeFiles(glob string) ([]string, error) {
	paths, err := filepath.Glob(glob)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("-subgrades %q matches no files", glob)
	}
	return paths, nil
}

func runCheck(provenancePath, sourcesPath, exceptionsPath string) error {
	reg, ex, err := readTerms(sourcesPath, exceptionsPath)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(provenancePath)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s is missing; a bundle without provenance cannot be deployed", provenancePath)
	}
	if err != nil {
		return err
	}
	var p pipeline.Provenance
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("%s: %w", provenancePath, err)
	}
	if err := pipeline.CheckDeployable(p, reg, ex); err != nil {
		return err
	}
	for _, s := range p.Sources {
		if s.Included {
			fmt.Printf("  %-24s %s\n", s.ID, s.Basis)
		}
	}
	fmt.Printf("provenance ok: bundle %s may be deployed\n", p.Version)
	return nil
}

func checkSpiritBonuses(dumpPath string, known map[int]bool, packed []pipeline.Entry) error {
	f, err := os.Open(dumpPath)
	if err != nil {
		return err
	}
	defer f.Close()

	wiki, stats, err := pipeline.ParseSpiritBonuses(f, known)
	if err != nil {
		return err
	}
	check := pipeline.CompareSpiritBonuses(wiki, packed)
	fmt.Printf("spirits: %d wiki pages, %d with a bonus, %d without a Skill Bonus section, %d without a level, %d rejected -> %d agree, %d disagree %v, %d wiki-only, %d packed-only\n",
		stats.Pages, stats.Parsed, stats.NoSection, stats.NoLevels, stats.Rejected,
		check.Agree, len(check.Disagree), check.Disagree,
		len(check.WikiOnly), len(check.PackedOnly))
	return nil
}

func finish(c config, entries []pipeline.Entry, skipped int, names pipeline.ItemNames, rarity map[int]int,
	prov pipeline.Provenance, sub *pipeline.SubgradeStats, known map[int]bool, corrections *pipeline.IDCorrections) error {
	stagesPath, outDir, version := c.stagesPath, c.outDir, c.version
	var stages []pipeline.Stage
	var stageStats pipeline.StageStats
	if stagesPath == "" {
		fmt.Println("WARNING: no -stages: the bundle has no stages, so the site cannot score any outfit")
	} else {
		raw, err := os.ReadFile(stagesPath)
		if err != nil {
			return err
		}
		corrections, err := readStageCorrections(c.stageCorrectionsPath)
		if err != nil {
			return err
		}
		if stages, stageStats, err = readStages(c, raw, corrections, entries); err != nil {
			return err
		}
	}

	cat := pipeline.NewAcquisitionCatalogue(entries, names, stages)
	acq, wikiStats, suits, err := readAcquisition(c, known, corrections, cat)
	if err != nil {
		return err
	}
	pipeline.ApplySuits(entries, suits)
	if err := checkBundle(c, entries, stages, stageStats, sub, acq, cat, wikiStats, suits); err != nil {
		return err
	}

	packed, err := pipeline.WriteCatalogue(entries)
	if err != nil {
		return err
	}
	provenance, err := json.MarshalIndent(prov, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Join(outDir, version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	files := map[string][]byte{
		"items.bin":       packed,
		"items.json":      pipeline.WriteItems(entries, names, rarity, suits),
		"stages.json":     pipeline.WriteStages(stages),
		"tags.json":       pipeline.WriteTags(),
		"positions.json":  pipeline.WritePositions(),
		"acquire.json":    pipeline.WriteAcquisition(version, acq),
		"provenance.json": append(provenance, '\n'),
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return err
		}
		fmt.Printf("%-12s %8d bytes\n", name, len(data))
	}
	index := fmt.Appendf(nil, "{\"version\":%q}\n", version)
	if err := os.WriteFile(filepath.Join(outDir, "index.json"), index, 0o644); err != nil {
		return err
	}

	fmt.Printf("\n%d items (%d source rows skipped), %d stages -> %s\n",
		len(entries), skipped, len(stages), dir)
	fmt.Printf("%d stages carry a tag bonus (%d awards); %d unknown tags, %d unknown grades, %d bonuses with no stage\n",
		stageStats.WithBonus, stageStats.Awards, stageStats.UnknownTag,
		stageStats.UnknownGrade, stageStats.OrphanBonus)
	fmt.Printf("%d stages took exact weights and awards from -stage-values (%d it carries are not in the bundle); %d have Maiden numbers of their own\n",
		stageStats.Valued, stageStats.Beyond, stageStats.Variants)
	fmt.Printf("%d stages require particular items\n", stageStats.Required)
	return nil
}

func readKnown(path string) (map[int]bool, error) {
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ids []json.Number
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, err
	}
	known := make(map[int]bool, len(ids))
	for _, id := range ids {
		n, err := id.Int64()
		if err != nil {
			return nil, err
		}
		known[int(n)] = true
	}
	return known, nil
}

func readItems(itemsPath, dumpPath string, known map[int]bool) ([]pipeline.Entry, int, error) {
	if dumpPath == "" {
		raw, err := os.ReadFile(itemsPath)
		if err != nil {
			return nil, 0, err
		}
		return pipeline.ParseWardrobe(raw)
	}

	f, err := os.Open(dumpPath)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	entries, stats, err := pipeline.ParseFandomDump(f, known)
	if err != nil {
		return nil, 0, err
	}
	fmt.Printf("fandom: %d item pages, %d parsed, %d without grades, %d rejected, %d unknown styles\n",
		stats.Pages, stats.Parsed, stats.NoGrades, stats.UnknownAny, stats.UnknownStyle)
	return entries, stats.NoGrades + stats.UnknownAny, nil
}

func readStages(c config, raw []byte, corrections map[string]pipeline.StageCorrection,
	entries []pipeline.Entry) ([]pipeline.Stage, pipeline.StageStats, error) {
	stages, stats, err := pipeline.ParseStagesWith(raw, corrections)
	if err != nil {
		return nil, stats, err
	}
	if c.stageScopePath != "" {
		b, err := os.ReadFile(c.stageScopePath)
		if err != nil {
			return nil, stats, err
		}
		scope, err := pipeline.ReadStageScope(b)
		if err != nil {
			return nil, stats, err
		}
		var out map[string]int
		if stages, out, err = pipeline.ApplyScope(stages, scope); err != nil {
			return nil, stats, err
		}
		fmt.Printf("scope: %d stages released on Global; not released: %v\n", len(stages), out)
	}
	if c.stageValuesPath != "" {
		b, err := os.ReadFile(c.stageValuesPath)
		if err != nil {
			return nil, stats, err
		}
		if err := pipeline.ApplyStageValues(stages, b, corrections, &stats); err != nil {
			return nil, stats, err
		}
	}
	if c.stageNamesPath != "" {
		b, err := os.ReadFile(c.stageNamesPath)
		if err != nil {
			return nil, stats, err
		}
		if _, err := pipeline.ApplyStageNames(stages, b); err != nil {
			return nil, stats, err
		}
	}
	if c.stageDifficultyPath != "" {
		b, err := os.ReadFile(c.stageDifficultyPath)
		if err != nil {
			return nil, stats, err
		}
		variants, err := pipeline.ReadStageVariants(b)
		if err != nil {
			return nil, stats, err
		}
		if err := pipeline.ApplyVariants(stages, variants, raw, &stats); err != nil {
			return nil, stats, err
		}
	}
	if c.stageRulesPath != "" {
		b, err := os.ReadFile(c.stageRulesPath)
		if err != nil {
			return nil, stats, err
		}
		rules, err := pipeline.ReadStageRules(b)
		if err != nil {
			return nil, stats, err
		}
		if err := pipeline.ApplyStageRules(stages, rules, entries, raw, &stats); err != nil {
			return nil, stats, err
		}
	}
	stats.Recount(stages)
	return stages, stats, nil
}

func readStageCorrections(path string) (map[string]pipeline.StageCorrection, error) {
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return pipeline.ReadStageCorrections(raw)
}

func readAcquisition(c config, known map[int]bool, corrections *pipeline.IDCorrections,
	cat pipeline.AcquisitionCatalogue) (map[int][]pipeline.Acquisition, *pipeline.WikiAcquisitionStats, map[int]string, error) {
	var wiki pipeline.WikiAcquisition
	var wikiStats *pipeline.WikiAcquisitionStats
	if c.dumpPath != "" {
		f, err := os.Open(c.dumpPath)
		if err != nil {
			return nil, nil, nil, err
		}
		defer f.Close()
		var stats pipeline.WikiAcquisitionStats
		if wiki, stats, err = pipeline.ParseFandomAcquisition(f, known, corrections, cat); err != nil {
			return nil, nil, nil, err
		}
		wikiStats = &stats
		cat.Suits = wiki.SuitOf
		fmt.Printf("acquisition: %d wiki item pages, %d say how to get the item, %d misnumbered read at their own ID, %d misnumbered or repeated left out; %d suit pages cover %d items; %d items in a suit the wiki names; %d names and %d suit parts unmatched, %d currencies unmatched, %d sources unread\n",
			stats.Pages, stats.WithEntries, stats.Moved, stats.Dropped, stats.SuitPages, stats.SuitItems,
			len(wiki.SuitOf), stats.Unresolved, stats.UnknownParts, stats.UnknownUnits, stats.Unclassified)
	}
	var packed map[int][]pipeline.Acquisition
	var packedSuits map[int]pipeline.PackedSuit
	if c.packedPath != "" {
		raw, err := os.ReadFile(c.packedPath)
		if err != nil {
			return nil, nil, nil, err
		}
		sources, err := pipeline.ParsePackedSources(raw, known)
		if err != nil {
			return nil, nil, nil, err
		}
		if packedSuits, err = pipeline.ParsePackedSuits(raw, known); err != nil {
			return nil, nil, nil, err
		}
		raw, err = os.ReadFile(c.acquisitionMapPath)
		if err != nil {
			return nil, nil, nil, err
		}
		m, err := pipeline.ReadAcquisitionMap(raw)
		if err != nil {
			return nil, nil, nil, err
		}
		if packed, err = m.Translate(sources, cat); err != nil {
			return nil, nil, nil, err
		}
		if c.dumpPath != "" {
			if err := m.CheckBasis(sources, wiki.Items); err != nil {
				return nil, nil, nil, err
			}
		}
	}
	acq, stats := pipeline.MergeAcquisition(cat, wiki.Items, packed, wiki.Suits)
	fmt.Printf("acquisition: %d of %d items say how to get them: %d from their wiki page, %d from the packed table, %d from their suit's wiki page\n",
		stats.Covered, stats.Catalogue, stats.FromWiki, stats.FromPacked, stats.FromSuits)
	ids := make([]int, 0, len(cat.Names))
	for id := range cat.Names {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	suits, suitStats := pipeline.LayerSuits(ids, wiki, packedSuits)
	shared := 0
	if wikiStats != nil {
		shared = wikiStats.SharedParts
	}
	fmt.Printf("suits: %d of %d items in %d suits: %d from the wiki's suit and item pages (%d on more than one suit page), %d from the packed table under the wiki's English name, %d from a suit page whose part name fits several items once the others are placed; %d the wiki lists only on a pack page, which is no suit, left to the packed table; not in a suit: %d the packed table files as a suit's base, %d whose packed suit the wiki gives no English name, %d with no suit anywhere\n",
		len(suits), len(ids), suitStats.Suits, suitStats.Wiki, shared, suitStats.Packed, suitStats.Late,
		suitStats.Packs, suitStats.Bases, suitStats.Unnamed, suitStats.None)
	return acq, wikiStats, suits, nil
}

func checkBundle(c config, entries []pipeline.Entry, stages []pipeline.Stage,
	stats pipeline.StageStats, sub *pipeline.SubgradeStats,
	acq map[int][]pipeline.Acquisition, cat pipeline.AcquisitionCatalogue,
	wikiStats *pipeline.WikiAcquisitionStats, suits map[int]string) error {
	var cov pipeline.Coverage
	if c.coveragePath != "" {
		raw, err := os.ReadFile(c.coveragePath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &cov); err != nil {
			return fmt.Errorf("%s: %w", c.coveragePath, err)
		}
	}
	var acknowledged map[string]pipeline.Acknowledged
	if c.stageCorrectionsPath != "" {
		raw, err := os.ReadFile(c.stageCorrectionsPath)
		if err != nil {
			return err
		}
		if acknowledged, err = pipeline.ReadAcknowledged(raw); err != nil {
			return err
		}
	}
	v := pipeline.CheckAcquisition(acq, cat, cov)
	v = append(v, pipeline.CheckSuits(suits, cov)...)
	if wikiStats != nil {
		v = append(v, pipeline.CheckAcquisitionNames(*wikiStats, cov)...)
	}
	if err := pipeline.CheckInvariants(entries, stages, stats, cov, acknowledged, sub); err != nil {
		var found pipeline.Violations
		if !errors.As(err, &found) {
			return err
		}
		v = append(found, v...)
	}
	if len(v) > 0 {
		return v
	}
	return nil
}
