package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/danielradosa/nikkibase/core/catalogue"
	"github.com/danielradosa/nikkibase/core/ideal"
	"github.com/danielradosa/nikkibase/core/optimizer"
)

func main() {
	data := flag.String("data", "web/public/data", "the directory holding index.json and the versioned bundles")
	out := flag.String("out", "web/src/generated/ideal.json", "the file to write every stage's best possible outfit into")
	flag.Parse()

	if err := run(*data, *out, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "ideal:", err)
		os.Exit(1)
	}
}

func run(data, out string, log io.Writer) error {
	start := time.Now()
	indexPath := filepath.Join(data, "index.json")
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("no bundle to precompute from (build one with cmd/bundle, see SOURCES.md): %w", err)
	}
	var index struct{ Version string }
	if err := json.Unmarshal(raw, &index); err != nil {
		return fmt.Errorf("%s: %w", indexPath, err)
	}
	if index.Version == "" {
		return fmt.Errorf("%s names no version", indexPath)
	}

	dir := filepath.Join(data, index.Version)
	if raw, err = os.ReadFile(filepath.Join(dir, "items.bin")); err != nil {
		return err
	}
	c, err := catalogue.Read(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Join(dir, "items.bin"), err)
	}
	if raw, err = os.ReadFile(filepath.Join(dir, "stages.json")); err != nil {
		return err
	}
	stages, err := ideal.ReadStages(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Join(dir, "stages.json"), err)
	}

	positions, posOf, placeOf := optimizer.FromCatalogue(c, nil)
	ideals, err := ideal.All(positions, posOf, placeOf, stages, 0)
	if err != nil {
		return err
	}
	encoded := ideal.Encode(index.Version, ideals)

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(out, encoded, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(log, "ideal: %d stage versions (%d stages) from bundle %s -> %s, %d bytes in %s\n",
		ideal.Versions(ideals), len(ideals), index.Version, out, len(encoded),
		time.Since(start).Round(time.Millisecond))
	return nil
}
