package ideal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"runtime"
	"slices"
	"strconv"
	"sync"

	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
)

type Outfit struct {
	Score int
	Items [][2]int
	Auto  Auto
}

type Auto struct {
	Placement scoring.Placement
	Score     int
	Items     [][2]int
}

type Stage struct {
	Mode, Name     string
	Scoring        scoring.Stage
	Require        [][]int
	Variants       map[string]scoring.Stage
	VariantRequire map[string][][]int
}

func (s Stage) Key() string { return s.Mode + "/" + s.Name }

type Ideal struct {
	Key      string
	Outfit   Outfit
	Variants map[string]Outfit
}

type numbers struct {
	Weights []float64      `json:"weights"`
	Attrs   []int8         `json:"attrs"`
	Tags    map[string]int `json:"tags"`
	Rules   *rules         `json:"rules"`
}

type rules struct {
	Require [][]int `json:"require"`
}

func ReadStages(raw []byte) ([]Stage, error) {
	var rows []struct {
		Name string `json:"name"`
		Mode string `json:"mode"`
		numbers
		Variants map[string]numbers `json:"variants"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("stages: %w", err)
	}
	stages := make([]Stage, 0, len(rows))
	for _, r := range rows {
		s := Stage{Mode: r.Mode, Name: r.Name}
		var err error
		if s.Scoring, err = r.numbers.scoring(); err != nil {
			return nil, fmt.Errorf("stages: %s: %w", s.Key(), err)
		}
		if s.Require, err = r.numbers.required(); err != nil {
			return nil, fmt.Errorf("stages: %s: %w", s.Key(), err)
		}
		for _, name := range slices.Sorted(maps.Keys(r.Variants)) {
			v := r.Variants[name]
			st, err := v.scoring()
			if err != nil {
				return nil, fmt.Errorf("stages: %s (%s): %w", s.Key(), name, err)
			}
			require, err := v.required()
			if err != nil {
				return nil, fmt.Errorf("stages: %s (%s): %w", s.Key(), name, err)
			}
			if v.Rules == nil {
				require = s.Require
			}
			if s.Variants == nil {
				s.Variants = make(map[string]scoring.Stage, len(r.Variants))
			}
			s.Variants[name] = st
			if len(require) > 0 {
				if s.VariantRequire == nil {
					s.VariantRequire = make(map[string][][]int, len(r.Variants))
				}
				s.VariantRequire[name] = require
			}
		}
		stages = append(stages, s)
	}
	return stages, nil
}

func (n numbers) scoring() (scoring.Stage, error) {
	var st scoring.Stage
	if len(n.Weights) != len(st.Weights) || len(n.Attrs) != len(st.Attrs) {
		return st, fmt.Errorf("%d weights and %d attrs, want %d of each", len(n.Weights), len(n.Attrs), len(st.Weights))
	}
	copy(st.Weights[:], n.Weights)
	copy(st.Attrs[:], n.Attrs)
	if len(n.Tags) > 0 {
		st.Tags = make(map[int]int, len(n.Tags))
		for k, v := range n.Tags {
			id, err := strconv.Atoi(k)
			if err != nil {
				return st, fmt.Errorf("tag %q is not a tag number", k)
			}
			st.Tags[id] = v
		}
	}
	return st, nil
}

func (n numbers) required() ([][]int, error) {
	if n.Rules == nil {
		return nil, nil
	}
	for _, set := range n.Rules.Require {
		if len(set) == 0 {
			return nil, fmt.Errorf("a required set names no item")
		}
	}
	return n.Rules.Require, nil
}

func Of(positions []optimizer.Position, posOf map[int]int, placeOf map[int]uint16, st scoring.Stage, require [][]int) Outfit {
	space := optimizer.Require(positions, posOf, require)
	best := space.Best(st, nil)
	auto, placement := space.BestPlacedFrom(st, best)
	return Outfit{Score: best.Score, Items: placed(best, placeOf),
		Auto: Auto{Placement: placement, Score: auto.Score, Items: placed(auto, placeOf)}}
}

func placed(r optimizer.Result, placeOf map[int]uint16) [][2]int {
	items := make([][2]int, len(r.Items))
	for i, it := range r.Items {
		items[i] = [2]int{it.ID, int(placeOf[it.ID])}
	}
	return items
}

func All(positions []optimizer.Position, posOf map[int]int, placeOf map[int]uint16, stages []Stage, workers int) ([]Ideal, error) {
	type job struct {
		st      scoring.Stage
		require [][]int
	}
	seen := make(map[string]bool, len(stages))
	var jobs []job
	for _, s := range stages {
		if seen[s.Key()] {
			return nil, fmt.Errorf("two stages share the key %q", s.Key())
		}
		seen[s.Key()] = true
		jobs = append(jobs, job{s.Scoring, s.Require})
		for _, name := range slices.Sorted(maps.Keys(s.Variants)) {
			jobs = append(jobs, job{s.Variants[name], s.VariantRequire[name]})
		}
	}

	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	outfits := make([]Outfit, len(jobs))
	next := make(chan int)
	var wg sync.WaitGroup
	for range min(workers, len(jobs)) {
		wg.Go(func() {
			for j := range next {
				outfits[j] = Of(positions, posOf, placeOf, jobs[j].st, jobs[j].require)
			}
		})
	}
	for j := range jobs {
		next <- j
	}
	close(next)
	wg.Wait()

	ideals := make([]Ideal, len(stages))
	j := 0
	for i, s := range stages {
		ideals[i] = Ideal{Key: s.Key(), Outfit: outfits[j]}
		j++
		for _, name := range slices.Sorted(maps.Keys(s.Variants)) {
			if ideals[i].Variants == nil {
				ideals[i].Variants = make(map[string]Outfit, len(s.Variants))
			}
			ideals[i].Variants[name] = outfits[j]
			j++
		}
	}
	return ideals, nil
}

func Versions(ideals []Ideal) int {
	n := 0
	for _, id := range ideals {
		n += 1 + len(id.Variants)
	}
	return n
}

func Encode(version string, ideals []Ideal) []byte {
	var b bytes.Buffer
	b.WriteString(`{"version":`)
	writeString(&b, version)
	b.WriteString(`,"stages":{`)
	for i, id := range ideals {
		if i > 0 {
			b.WriteByte(',')
		}
		writeString(&b, id.Key)
		b.WriteByte(':')
		writeOutfit(&b, id.Outfit, id.Variants)
	}
	b.WriteString("}}")
	return b.Bytes()
}

func writeOutfit(b *bytes.Buffer, o Outfit, variants map[string]Outfit) {
	b.WriteByte('{')
	writeScored(b, o.Score, o.Items)
	b.WriteString(`,"auto":{"charmSmile":`)
	b.WriteString(strconv.Itoa(o.Auto.Placement.CharmSmile))
	b.WriteString(`,"smile":`)
	b.WriteString(strconv.Itoa(o.Auto.Placement.Smile))
	b.WriteByte(',')
	writeScored(b, o.Auto.Score, o.Auto.Items)
	b.WriteByte('}')
	if len(variants) > 0 {
		b.WriteString(`,"variants":{`)
		for i, name := range slices.Sorted(maps.Keys(variants)) {
			if i > 0 {
				b.WriteByte(',')
			}
			writeString(b, name)
			b.WriteByte(':')
			writeOutfit(b, variants[name], nil)
		}
		b.WriteByte('}')
	}
	b.WriteByte('}')
}

func writeScored(b *bytes.Buffer, score int, items [][2]int) {
	b.WriteString(`"score":`)
	b.WriteString(strconv.Itoa(score))
	b.WriteString(`,"items":[`)
	for i, it := range items {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('[')
		b.WriteString(strconv.Itoa(it[0]))
		b.WriteByte(',')
		b.WriteString(strconv.Itoa(it[1]))
		b.WriteByte(']')
	}
	b.WriteByte(']')
}

func writeString(b *bytes.Buffer, s string) {
	e := json.NewEncoder(b)
	e.SetEscapeHTML(false)
	if err := e.Encode(s); err != nil {
		panic(err)
	}
	b.Truncate(b.Len() - 1)
}
