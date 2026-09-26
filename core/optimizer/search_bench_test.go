//go:build dataset

package optimizer_test

import (
	"bufio"
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/danielradosa/nikkibase/core/catalogue"
	"github.com/danielradosa/nikkibase/core/ideal"
	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
	"github.com/danielradosa/nikkibase/core/wardrobe"
)

var (
	benchData     = flag.String("data", "../../web/public/data", "the directory holding index.json and the versioned bundles")
	benchWardrobe = flag.String("wardrobe", "", "an @SEL selections file used as the owned pool (default: a fixed sample)")
	dumpTo        = flag.String("dump", "", "write every search result to this file")
)

var benchKeys = []string{
	"Arena/Beach Party", "Arena/Cloud Lady",
	"Co-op/Orlando - Evening Gown", "Co-op/Neva - Maiden", "Co-op/Kimi - Harajuku Style",
	"Commission/1-1", "Commission/1-4", "Commission/2-5", "Commission/20-7",
	"Story/1-1", "Story/1-2", "Story/4-1", "Story/4-1#maiden", "Story/7-7", "Story/9-4",
	"Story/2-7", "Story/11-3", "Story/14-8", "Story/15-9", "Story/19-9", "Story/III-3-Side 2",
}

type version struct {
	key     string
	st      scoring.Stage
	require [][]int
}

type pool struct {
	name      string
	owned     func(int32) bool
	positions []optimizer.Position
	posOf     map[int]int
	placeOf   map[int]uint16
}

type fixture struct {
	cat      *catalogue.Catalogue
	versions []version
	picked   []version
	pools    []pool
}

var (
	fixtureOnce sync.Once
	fixtureVal  *fixture
	fixtureErr  error
)

func loadFixture(tb testing.TB) *fixture {
	tb.Helper()
	fixtureOnce.Do(func() { fixtureVal, fixtureErr = buildFixture() })
	if fixtureErr != nil {
		tb.Fatal(fixtureErr)
	}
	return fixtureVal
}

func buildFixture() (*fixture, error) {
	raw, err := os.ReadFile(filepath.Join(*benchData, "index.json"))
	if err != nil {
		return nil, err
	}
	var index struct{ Version string }
	if err := json.Unmarshal(raw, &index); err != nil {
		return nil, err
	}
	dir := filepath.Join(*benchData, index.Version)
	if raw, err = os.ReadFile(filepath.Join(dir, "items.bin")); err != nil {
		return nil, err
	}
	f := &fixture{}
	if f.cat, err = catalogue.Read(raw); err != nil {
		return nil, err
	}
	if raw, err = os.ReadFile(filepath.Join(dir, "stages.json")); err != nil {
		return nil, err
	}
	stages, err := ideal.ReadStages(raw)
	if err != nil {
		return nil, err
	}
	byKey := map[string]version{}
	for _, s := range stages {
		v := version{s.Key(), s.Scoring, s.Require}
		f.versions = append(f.versions, v)
		byKey[v.key] = v
		for _, name := range slices.Sorted(maps.Keys(s.Variants)) {
			v := version{s.Key() + "#" + name, s.Variants[name], s.VariantRequire[name]}
			f.versions = append(f.versions, v)
			byKey[v.key] = v
		}
	}
	for _, k := range benchKeys {
		v, ok := byKey[k]
		if !ok {
			return nil, fmt.Errorf("no stage %q", k)
		}
		f.picked = append(f.picked, v)
	}

	sample := map[int32]bool{}
	for i, id := range f.cat.IDs {
		if (uint32(i)*2654435761)>>16%17 == 0 {
			sample[id] = true
		}
	}
	f.pools = []pool{{name: "all"}}
	if *benchWardrobe != "" {
		if raw, err = os.ReadFile(*benchWardrobe); err != nil {
			return nil, err
		}
		w, err := wardrobe.DecodeSelections(raw)
		if err != nil {
			return nil, err
		}
		have := make(map[int32]bool, len(w.Items))
		for _, id := range w.Items {
			have[int32(id)] = true
		}
		f.pools = append(f.pools, pool{name: "wardrobe", owned: func(id int32) bool { return have[id] }})
	}
	f.pools = append(f.pools, pool{name: "sample", owned: func(id int32) bool { return sample[id] }})
	for i := range f.pools {
		p := &f.pools[i]
		p.positions, p.posOf, p.placeOf = optimizer.FromCatalogue(f.cat, p.owned)
	}
	return f, nil
}

func (f *fixture) owned() pool {
	return f.pools[1]
}

func heaviest(st scoring.Stage) scoring.Placement {
	order := []int{0, 1, 2, 3, 4}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(st.Weights[b], st.Weights[a]) })
	return scoring.Placement{CharmSmile: int(st.Attrs[order[0]]), Smile: int(st.Attrs[order[1]])}
}

func search(positions []optimizer.Position, posOf map[int]int, v version, skills string) (optimizer.Result, scoring.Skills, scoring.Placement) {
	space := optimizer.Require(positions, posOf, v.require)
	switch skills {
	case "none":
		return space.Best(v.st, scoring.Skills{}), scoring.Skills{}, scoring.Placement{CharmSmile: -1, Smile: -1}
	case "auto":
		r, p := space.BestPlaced(v.st)
		return r, p.Skills(), p
	case "levels":
		r, p := space.BestPlacedAt(v.st, lowLevels)
		return r, p.SkillsAt(lowLevels), p
	default:
		p := heaviest(v.st)
		sk := p.Skills()
		return space.Best(v.st, sk), sk, p
	}
}

var lowLevels = scoring.Levels{Charming: 4, Smile: 6}

var skillModes = []string{"none", "auto", "manual", "levels"}

var sink int

func BenchmarkSearch(b *testing.B) {
	f := loadFixture(b)
	for _, p := range []pool{f.pools[0], f.owned()} {
		for _, skills := range skillModes {
			b.Run(p.name+"/"+skills, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					for _, v := range f.picked {
						r, _, _ := search(p.positions, p.posOf, v, skills)
						sink += r.Score
					}
				}
				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*len(f.picked))/1e6, "ms/stage")
			})
		}
	}
}

func BenchmarkBrowserCall(b *testing.B) {
	f := loadFixture(b)
	for _, p := range []pool{f.pools[0], f.owned()} {
		for _, skills := range skillModes {
			b.Run(p.name+"/"+skills, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					for _, v := range f.picked {
						positions, posOf, _ := optimizer.FromCatalogue(f.cat, p.owned)
						r, sk, _ := search(positions, posOf, v, skills)
						for _, it := range r.Items {
							sink += len(optimizer.Ranked(positions[posOf[it.ID]].Items, v.st, sk))
						}
					}
				}
				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*len(f.picked))/1e6, "ms/stage")
			})
		}
	}
}

func BenchmarkFromCatalogue(b *testing.B) {
	f := loadFixture(b)
	for _, p := range []pool{f.pools[0], f.owned()} {
		b.Run(p.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				positions, _, _ := optimizer.FromCatalogue(f.cat, p.owned)
				sink += len(positions)
			}
		})
	}
}

func BenchmarkRanked(b *testing.B) {
	f := loadFixture(b)
	p := f.pools[0]
	type worn struct {
		v  version
		sk scoring.Skills
		r  optimizer.Result
	}
	var picks []worn
	for _, v := range f.picked {
		r, sk, _ := search(p.positions, p.posOf, v, "auto")
		picks = append(picks, worn{v, sk, r})
	}
	b.ReportAllocs()
	for b.Loop() {
		for _, w := range picks {
			for _, it := range w.r.Items {
				sink += len(optimizer.Ranked(p.positions[p.posOf[it.ID]].Items, w.v.st, w.sk))
			}
		}
	}
}

func TestDumpSearch(t *testing.T) {
	if *dumpTo == "" {
		t.Skip("no -dump file")
	}
	f := loadFixture(t)
	type job struct {
		pool   pool
		v      version
		skills string
	}
	var jobs []job
	for _, p := range f.pools {
		for _, v := range f.versions {
			for _, s := range skillModes {
				jobs = append(jobs, job{p, v, s})
			}
		}
	}
	lines := make([]string, len(jobs))
	next := make(chan int)
	var wg sync.WaitGroup
	for range 6 {
		wg.Go(func() {
			for j := range next {
				jb := jobs[j]
				r, sk, pl := search(jb.pool.positions, jb.pool.posOf, jb.v, jb.skills)
				var b strings.Builder
				fmt.Fprintf(&b, "%s|%s|%s score=%d dress=%v sep=%v place=%d,%d items=", jb.pool.name, jb.v.key, jb.skills,
					r.Score, r.Dress, r.Separates, pl.CharmSmile, pl.Smile)
				for i, it := range r.Items {
					if i > 0 {
						b.WriteByte(',')
					}
					b.WriteString(strconv.Itoa(it.ID))
				}
				b.WriteString(" alts=")
				for _, it := range r.Items {
					ranked := optimizer.Ranked(jb.pool.positions[jb.pool.posOf[it.ID]].Items, jb.v.st, sk)
					b.WriteByte('[')
					for i, a := range ranked[:min(6, len(ranked))] {
						if i > 0 {
							b.WriteByte(',')
						}
						b.WriteString(strconv.Itoa(a.ID))
					}
					b.WriteByte(']')
				}
				lines[j] = b.String()
			}
		})
	}
	for j := range jobs {
		next <- j
	}
	close(next)
	wg.Wait()
	out, err := os.Create(*dumpTo)
	if err != nil {
		t.Fatal(err)
	}
	w := bufio.NewWriter(out)
	for _, l := range lines {
		w.WriteString(l)
		w.WriteByte('\n')
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	out.Close()
}
