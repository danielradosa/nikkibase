package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"slices"
	"unicode"

	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
	"github.com/danielradosa/nikkibase/core/wardrobe"
	"github.com/danielradosa/nikkibase/pipeline"
)

func main() {
	var (
		wardrobePath  = flag.String("wardrobe", "", "path to a clothes_date file, or a Nikki Calc selections (@SEL) file")
		keystreamPath = flag.String("keystream", "web/public/keystream.bin", "keystream for clothes_date files")
		itemsPath     = flag.String("items", "", "path to the community wardrobe.js")
		stagesPath    = flag.String("stages", "", "path to the community levels.js")
		stageName     = flag.String("stage", "1-1", "stage to dress for")
	)
	flag.Parse()

	if err := run(*wardrobePath, *keystreamPath, *itemsPath, *stagesPath, *stageName); err != nil {
		fmt.Fprintln(os.Stderr, "outfit:", err)
		os.Exit(1)
	}
}

func run(wardrobePath, keystreamPath, itemsPath, stagesPath, stageName string) error {
	owned, unresolved, err := readWardrobe(wardrobePath, keystreamPath)
	if err != nil {
		return err
	}

	raw, err := os.ReadFile(itemsPath)
	if err != nil {
		return err
	}
	entries, skipped, err := pipeline.ParseWardrobe(raw)
	if err != nil {
		return err
	}

	raw, err = os.ReadFile(stagesPath)
	if err != nil {
		return err
	}
	stages, _, err := pipeline.ParseStages(raw)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(stages, func(s pipeline.Stage) bool { return s.Name == stageName })
	if i < 0 {
		return fmt.Errorf("stage %q not found among %d stages", stageName, len(stages))
	}
	stage := stages[i]

	positions, unplaced := pipeline.Positions(entries, owned)
	best := optimizer.Best(positions, stage.Stage, nil)

	names := map[int]string{}
	for _, e := range entries {
		names[e.Item.ID] = e.Name
	}

	fmt.Printf("wardrobe: %d items", len(owned))
	if unresolved > 0 {
		fmt.Printf(" (%d could not be read)", unresolved)
	}
	known := 0
	for _, p := range positions {
		known += len(p.Items)
	}
	fmt.Printf("; %d of them are in the catalogue, across %d positions\n", known, len(positions))
	if unplaced > 0 {
		fmt.Printf("WARNING: %d owned rows were left out: no wearable place the game names, or an ID already placed\n", unplaced)
	}
	fmt.Printf("catalogue: %d items, %d source rows skipped\n", len(entries), skipped)
	fmt.Printf("stage %s weights: %v\n\n", stage.Name, stage.Stage.Weights)

	slices.SortFunc(best.Items, func(a, b scoring.Item) int { return int(a.Slot) - int(b.Slot) })
	for _, it := range best.Items {
		fmt.Printf("  %-10s %-34s %d\n", slotName(it.Slot), names[it.ID], it.ID)
	}
	fmt.Printf("\nscore: %d\n", best.Score)
	return nil
}

func readWardrobe(path, keystreamPath string) (owned []int, unresolved int, err error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	var w *wardrobe.Wardrobe
	if bytes.HasPrefix(bytes.TrimLeftFunc(src, unicode.IsSpace), []byte("@SEL")) {
		w, err = wardrobe.DecodeSelections(src)
	} else {
		var raw []byte
		if raw, err = os.ReadFile(keystreamPath); err != nil {
			return nil, 0, err
		}
		var ks *wardrobe.Keystream
		if ks, err = wardrobe.ParseKeystream(raw); err != nil {
			return nil, 0, err
		}
		w, err = wardrobe.Decode(src, ks)
	}
	if err != nil {
		return nil, 0, err
	}
	return w.Items, w.Unresolved, nil
}

func slotName(s scoring.Slot) string {
	names := map[scoring.Slot]string{
		scoring.Hair: "hair", scoring.Dress: "dress", scoring.Coat: "coat",
		scoring.Top: "top", scoring.Bottom: "bottom", scoring.Hosiery: "hosiery",
		scoring.Shoes: "shoes", scoring.Makeup: "makeup", scoring.Accessory: "accessory",
		scoring.Spirit: "spirit",
	}
	return names[s]
}
