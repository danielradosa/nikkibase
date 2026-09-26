package worth

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

func rebuilt(s *Session, br *branch, sd *side, c *coef, items []int32) *evaluator {
	l := s.l
	copied := sd.clone()
	for _, i := range items {
		l.addTo(copied, br, c, i)
	}
	e := l.newEvaluator()
	e.build(br, copied)
	return e
}

func TestManyPiecesMatchARebuiltSide(t *testing.T) {
	rng := rand.New(rand.NewPCG(61, 17))
	var checked, merged, moved int
	for trial := range 120 {
		layout := manyAccessories()
		if trial%3 == 0 {
			layout = places
		}
		var settings Settings
		if trial%2 == 1 {
			settings = Settings{Skills: true, Levels: scoring.MaxLevels}
		}
		fx := gradedFixture(rng, layout, 3, 4, settings)
		s := fx.session()
		var cands []int32
		for i := range s.l.items {
			if !s.l.owned[i] {
				cands = append(cands, int32(i))
			}
		}
		for vi := range fx.versions {
			st := s.stages[vi]
			if st.failing {
				continue
			}
			sc := s.scorer(st, &s.versions[vi])
			sides := []struct {
				evs   []*evaluator
				sides []*side
				c     *coef
			}{{sc.evU, st.u, st.cU}}
			if st.k != nil {
				sides = append(sides, struct {
					evs   []*evaluator
					sides []*side
					c     *coef
				}{sc.evK, st.k, st.cK})
			}
			for range 40 {
				n := 2 + rng.IntN(7)
				items := make([]int32, 0, n)
				for range n {
					items = append(items, cands[rng.IntN(len(cands))])
				}
				slices.Sort(items)
				items = slices.Compact(items)
				for _, side := range sides {
					var ps []piece
					for _, i := range items {
						v, f := s.l.value(side.c, i)
						ps = append(ps, piece{s.l.at[i], i, v, f})
					}
					for b, e := range side.evs {
						want := rebuilt(s, st.branches[b], side.sides[b], side.c, items)
						total, choice, worn := e.withMany(ps)
						if math.Abs(total-want.total) > 1e-9*(1+math.Abs(want.total)) || choice != want.accChoice || worn != want.choices[want.accChoice].worn {
							t.Fatalf("trial %d stage %d items %v: withMany gives %v (choice %d, %d worn), a rebuilt side %v (choice %d, %d worn)",
								trial, vi, items, total, choice, worn, want.total, want.accChoice, want.choices[want.accChoice].worn)
						}
						got := scoringIDs(e.outfitMany(nil, ps, choice, worn), fx.versions[vi].Stage)
						if wantItems := scoringIDs(want.outfit(nil, &change{}), fx.versions[vi].Stage); !slices.Equal(got, wantItems) {
							t.Fatalf("trial %d stage %d items %v: outfitMany wears %v, a rebuilt side %v", trial, vi, items, got, wantItems)
						}
						checked++
						if total != e.total {
							moved++
						}
						for c := range e.choices {
							e.gather(&e.choices[c], ps, false)
							if len(e.mg.offs) > 1 {
								merged++
							}
							e.unmark()
						}
					}
				}
			}
			sc.release()
		}
	}
	if checked < 20000 || merged < 10000 || moved < 10000 {
		t.Errorf("%d checks, %d merged several accessories, %d changed the total", checked, merged, moved)
	}
	t.Logf("%d checks, %d merged several accessories, %d changed the total", checked, merged, moved)
}
