package worth

import (
	"slices"

	"github.com/danielradosa/nikkibase/core/optimizer"
	"github.com/danielradosa/nikkibase/core/scoring"
)

type Version struct {
	Key     string
	Mode    string
	Stage   scoring.Stage
	Require [][]int
	Ideal   int
}

type Settings struct {
	Skills bool
	Levels scoring.Levels
	Suits  []Suit
}

type Suit struct {
	Key   string
	Items []int
}

type Filter struct {
	Modes     []string
	Skip      []string
	Slots     []scoring.Slot
	Positions []int
	Suits     bool
}

type Example struct {
	Key    string
	Points int
	Pct    float64
}

type Row struct {
	Suit     string
	Items    []int
	Worth    float64
	Stages   int
	Best     Example
	Examples []Example
}

type Needed struct {
	Key     string
	Missing [][]int
}

type Ranking struct {
	Rows   []Row
	Needed []Needed
}

const examples = 5

type Session struct {
	l        *layout
	owned    func(int32) bool
	versions []Version
	skills   bool
	levels   scoring.Levels
	stages   []*stage
	done     int
	pairs    [][2]int32
	paired   map[[2]int32]bool
	free     []*evaluator
	buf      []scoring.Item
	taken    []bool
	units    []unit
	unitsOf  [][]int32
	suits    []suitBase
	pieces   [3][]piece
	cons     []int32
	suitRun  *suitRun
}

func NewSession(positions []optimizer.Position, posOf map[int]int, owned func(int32) bool, versions []Version, s Settings) *Session {
	l := newLayout(positions, posOf, owned)
	if prunable(versions) {
		protected := make([]bool, len(l.items))
		for _, v := range versions {
			for _, set := range v.Require {
				for _, id := range set {
					if i, ok := l.index[id]; ok {
						protected[i] = true
					}
				}
			}
		}
		l.prune(func(i int32) bool { return protected[i] })
	}
	session := &Session{
		l:        l,
		owned:    owned,
		versions: slices.Clone(versions),
		skills:   s.Skills && s.Levels.Valid() && !s.Levels.None(),
		levels:   s.Levels,
		stages:   make([]*stage, len(versions)),
		paired:   map[[2]int32]bool{},
		taken:    make([]bool, len(l.items)),
	}
	session.addSuits(s.Suits)
	return session
}

func prunable(versions []Version) bool {
	for _, v := range versions {
		for _, w := range v.Stage.Weights {
			if !(w >= 0) {
				return false
			}
		}
		for _, a := range v.Stage.Tags {
			if a < 0 {
				return false
			}
		}
	}
	return true
}

func (s *Session) Run(n int) (done, total int) {
	for ; n > 0 && s.done < len(s.versions); n-- {
		s.stages[s.done] = s.process(s.done, nil)
		s.done++
	}
	return s.done, len(s.versions)
}

func (s *Session) owns(id int) bool {
	return s.owned == nil || s.owned(int32(id))
}

func (s *Session) effective(p scoring.Placement) scoring.Placement {
	if s.levels.Smile == 0 || p.Smile == p.CharmSmile {
		p.Smile = -1
	}
	if s.levels.Charming == 0 && p.Smile >= 0 && p.Smile < p.CharmSmile {
		p.CharmSmile, p.Smile = p.Smile, p.CharmSmile
	}
	return p
}

func (s *Session) acquire() *evaluator {
	if n := len(s.free); n > 0 {
		e := s.free[n-1]
		s.free = s.free[:n-1]
		return e
	}
	return s.l.newEvaluator()
}

func pct(points int32, ideal int) float64 {
	return float64(100*int64(points)) / float64(ideal)
}
