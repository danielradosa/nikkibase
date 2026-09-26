package scoring

import (
	"math"
	"testing"
)

func hair(lively int, tags ...int) Item {
	it := Item{Slot: Hair, Tags: tags}
	it.Attrs = [pairs]int8{Gorgeous, Lively, Mature, Sexy, Warm}
	it.Stats[1] = lively
	return it
}

func TestSingleItemAndSkills(t *testing.T) {
	stage := CustomStage([pairs]int{0, 100, 0, 0, 0}, nil)
	cuteSmile := hair(92)

	for _, tc := range []struct {
		name   string
		skills Skills
		want   int
	}{
		{"no skills", nil, 920},
		{"charming+smile on Lively", Skills{Lively: CharmingSmile}, 1635},
		{"smile on Lively", Skills{Lively: SmileOnly}, 1168},
		{"skill on an unrelated attribute", Skills{Cute: CharmingSmile}, 920},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Score([]Item{cuteSmile}, stage, tc.skills); got != tc.want {
				t.Errorf("Score = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestTagIsExemptFromSkills(t *testing.T) {
	const modernChina = 7
	stage := CustomStage([pairs]int{0, 100, 0, 0, 0}, map[int]int{modernChina: 100})
	item := hair(83, modernChina)

	if got := Score([]Item{item}, stage, nil); got != 880 {
		t.Errorf("without skills = %d, want 880", got)
	}
	if got := Score([]Item{item}, stage, Skills{Lively: CharmingSmile}); got != 1525 {
		t.Errorf("with charming+smile = %d, want 1525", got)
	}
}

func TestMismatchedSideScoresZero(t *testing.T) {
	stage := CustomStage([pairs]int{100, 100, 0, 0, 0}, nil)
	item := hair(83)
	item.Stats[0] = 500

	if got := Score([]Item{item}, stage, nil); got != 415 {
		t.Errorf("Score = %d, want 415", got)
	}
}

func TestSpiritFlatBonus(t *testing.T) {
	spirit := Item{Slot: Spirit, FlatBonus: 1500}
	spirit.Attrs = [pairs]int8{Gorgeous, Lively, Cute, Sexy, Warm}
	spirit.Stats[1], spirit.Stats[2] = 37, 25

	lively := CustomStage([pairs]int{0, 100, 0, 0, 0}, nil)
	if got := Score([]Item{spirit}, lively, Skills{Lively: CharmingSmile}); got != 2157 {
		t.Errorf("boosted = %d, want 2157", got)
	}
	cute := CustomStage([pairs]int{0, 0, 100, 0, 0}, nil)
	if got := Score([]Item{spirit}, cute, nil); got != 1750 {
		t.Errorf("on a Cute stage = %d, want 1750", got)
	}
}

func TestAccessoryPenalty(t *testing.T) {
	stage := CustomStage([pairs]int{0, 100, 0, 0, 0}, nil)
	acc := func() Item {
		it := Item{Slot: Accessory}
		it.Attrs = [pairs]int8{Gorgeous, Lively, Mature, Sexy, Warm}
		it.Stats[1] = 30
		return it
	}
	score := func(n int) int {
		outfit := make([]Item, n)
		for i := range outfit {
			outfit[i] = acc()
		}
		return Score(outfit, stage, nil)
	}

	if got := score(3); got != 900 {
		t.Errorf("3 accessories = %d, want 900", got)
	}
	if got := score(4); got != 1140 {
		t.Errorf("4 accessories = %d, want 1140", got)
	}
	if ratio := AccessoryPenalty(4) / AccessoryPenalty(5); ratio < 1.0555 || ratio > 1.0556 {
		t.Errorf("r(4)/r(5) = %v, want 0.95/0.90", ratio)
	}
	if AccessoryPenalty(16) != AccessoryPenalty(23) {
		t.Error("penalty should plateau at 16")
	}
}

func TestAccessoryTagEscapesThePenalty(t *testing.T) {
	const tag = 3
	stage := CustomStage([pairs]int{0, 100, 0, 0, 0}, map[int]int{tag: 50000})
	outfit := make([]Item, 23)
	for i := range outfit {
		outfit[i] = Item{Slot: Accessory, Attrs: [pairs]int8{Gorgeous, Lively, Mature, Sexy, Warm}}
	}
	outfit[0].Tags = []int{tag}

	if got := Score(outfit, stage, nil); got != 10000 {
		t.Errorf("Score = %d, want 10000", got)
	}
}

func TestCustomStageWeights(t *testing.T) {
	for _, tc := range []struct {
		name    string
		sliders [pairs]int
		want    [pairs]float64
	}{
		{"single attribute", [pairs]int{0, 100, 0, 0, 0}, [pairs]float64{0, 10, 0, 0, 0}},
		{"half strength scores the same", [pairs]int{0, 50, 0, 0, 0}, [pairs]float64{0, 10, 0, 0, 0}},
		{"two attributes split evenly", [pairs]int{100, 100, 0, 0, 0}, [pairs]float64{5, 5, 0, 0, 0}},
		{"all sliders zero", [pairs]int{}, [pairs]float64{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := CustomStage(tc.sliders, nil).Weights; got != tc.want {
				t.Errorf("weights = %v, want %v", got, tc.want)
			}
		})
	}

	if got := CustomStage([pairs]int{-100, 100, 0, 0, 0}, nil).Attrs; got[0] != Gorgeous || got[1] != Lively {
		t.Errorf("attrs = %v, want Gorgeous then Lively", got)
	}
}

func TestFloorForgivesOnlyFloatError(t *testing.T) {
	below := func(x float64, ulps int) float64 {
		for range ulps {
			x = math.Nextafter(x, math.Inf(-1))
		}
		return x
	}
	for _, c := range []struct {
		x    float64
		want int
	}{
		{below(201, 1), 201},
		{below(54158, 16), 54158},
		{below(225000, 32), 225000},
		{54157.999995, 54157},
		{119687.99999, 119687},
		{below(1, 1) - 1e-7, 0},
		{0, 0},
		{12.5, 12},
	} {
		if got := floor(c.x); got != c.want {
			t.Errorf("floor(%.12f) = %d, want %d", c.x, got, c.want)
		}
	}
}

func TestPlacementSkills(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    Placement
		want Skills
	}{
		{"two attributes", Placement{Lively, Cute}, Skills{Lively: CharmingSmile, Cute: SmileOnly}},
		{"the same attribute keeps Charming", Placement{Lively, Lively}, Skills{Lively: CharmingSmile}},
		{"no Smile", Placement{Lively, -1}, Skills{Lively: CharmingSmile}},
		{"no Charming", Placement{-1, Cute}, Skills{Cute: SmileOnly}},
		{"nothing placed", Placement{-1, -1}, Skills{}},
		{"Gorgeous is code zero, not nothing", Placement{Gorgeous, Simple}, Skills{Gorgeous: CharmingSmile, Simple: SmileOnly}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.p.Skills()
			if len(got) != len(tc.want) {
				t.Fatalf("Skills = %v, want %v", got, tc.want)
			}
			for code, m := range tc.want {
				if got[code] != m {
					t.Errorf("Skills = %v, want %v", got, tc.want)
				}
			}
		})
	}

	stage := CustomStage([pairs]int{0, 100, 0, 0, 0}, nil)
	if got := Score([]Item{hair(92)}, stage, Placement{Lively, Lively}.Skills()); got != 1635 {
		t.Errorf("Charming and Smile named on one attribute score %d, want the Charming 1635", got)
	}
}

func TestSkillsAtMaxAreTheLiterals(t *testing.T) {
	sk := Placement{Lively, Cute}.SkillsAt(MaxLevels)
	if math.Float64bits(sk[Lively]) != math.Float64bits(CharmingSmile) || math.Float64bits(sk[Cute]) != math.Float64bits(SmileOnly) {
		t.Errorf("max levels give %v, want exactly %v and %v", sk, CharmingSmile, SmileOnly)
	}
}

func TestSkillLevelsMatchTheWiki(t *testing.T) {
	if charmingPercent != [...]int{0, 24, 26, 28, 30, 32, 34, 36, 38, 40} {
		t.Errorf("Charming percents %v", charmingPercent)
	}
	if smilePercent != [...]int{0, 15, 17, 19, 20, 22, 24, 25, 26, 27} {
		t.Errorf("Smile percents %v", smilePercent)
	}
}

func TestSkillsAtLevels(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    Placement
		l    Levels
		want Skills
	}{
		{"nothing taken", Placement{Lively, Cute}, Levels{0, 0}, Skills{}},
		{"Charming locked leaves two Smiles", Placement{Lively, Cute}, Levels{0, 9}, Skills{Lively: 1.27, Cute: 1.27}},
		{"no Smile leaves Charming alone", Placement{Lively, Cute}, Levels{9, 0}, Skills{Lively: 1.4}},
		{"level 5 each", Placement{Lively, Cute}, Levels{5, 5}, Skills{Lively: 1.6104, Cute: 1.22}},
		{"level 1 each", Placement{Lively, Cute}, Levels{1, 1}, Skills{Lively: 1.426, Cute: 1.15}},
		{"the same attribute keeps Charming", Placement{Lively, Lively}, Levels{3, 4}, Skills{Lively: 1.536}},
		{"nothing placed", Placement{-1, -1}, Levels{4, 4}, Skills{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.p.SkillsAt(tc.l)
			if len(got) != len(tc.want) {
				t.Fatalf("SkillsAt = %v, want %v", got, tc.want)
			}
			for code, m := range tc.want {
				if math.Float64bits(got[code]) != math.Float64bits(m) {
					t.Errorf("SkillsAt = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestSkillsGrowWithEveryLevel(t *testing.T) {
	at := func(c, s int) (float64, float64) {
		sk := Placement{Lively, Cute}.SkillsAt(Levels{c, s})
		return sk.mult(Lively), sk.mult(Cute)
	}
	for c := range 10 {
		for s := range 10 {
			a, b := at(c, s)
			if c < 9 {
				if na, _ := at(c+1, s); na <= a {
					t.Errorf("Charming %d->%d at Smile %d: %v -> %v", c, c+1, s, a, na)
				}
			}
			if s < 9 {
				if na, nb := at(c, s+1); na <= a || nb <= b {
					t.Errorf("Smile %d->%d at Charming %d: %v,%v -> %v,%v", s, s+1, c, a, b, na, nb)
				}
			}
		}
	}
}

func TestLevelsValid(t *testing.T) {
	for _, tc := range []struct {
		l    Levels
		want bool
	}{
		{Levels{0, 0}, true}, {Levels{9, 9}, true}, {Levels{4, 7}, true},
		{Levels{-1, 5}, false}, {Levels{5, -1}, false}, {Levels{10, 0}, false}, {Levels{0, 10}, false},
	} {
		if got := tc.l.Valid(); got != tc.want {
			t.Errorf("%+v.Valid() = %v", tc.l, got)
		}
	}
}

func TestPointsAndFixedMakeTheScore(t *testing.T) {
	const paid = 5
	st := Stage{
		Attrs:   [pairs]int8{Simple, Lively, Cute, Sexy, Cool},
		Weights: [pairs]float64{50.625, 25, 0, 8.5, 15},
		Tags:    map[int]int{paid: 8002},
	}
	matches := [pairs]int8{Simple, Lively, Cute, Sexy, Cool}
	outfit := []Item{
		{Slot: Hair, Attrs: matches, Stats: [pairs]int{312, 207, 190, 44, 81}, Tags: []int{paid}},
		{Slot: Dress, Attrs: [pairs]int8{Gorgeous, Lively, Cute, Sexy, Warm}, Stats: [pairs]int{1570, 2210, 900, 330, 480}},
		{Slot: Spirit, Attrs: matches, Stats: [pairs]int{37, 25, 19, 11, 9}, FlatBonus: 1500},
	}
	for i := range 6 {
		outfit = append(outfit, Item{Slot: Accessory, Attrs: matches,
			Stats: [pairs]int{41 + 7*i, 33, 12 + i, 9, 27 - 3*i}, Tags: []int{paid}[:i%2]})
	}

	var fixed float64
	for _, it := range outfit {
		_, f := Contribution(it, st, nil)
		fixed += f
	}
	if fixed == 0 {
		t.Fatal("the fixture pays no tags or flat bonus, so the fixed part is untested")
	}
	points := Points(outfit, st)
	sum := fixed
	for _, p := range points {
		sum += p
	}
	want := unfloored(outfit, st, nil)
	if math.Abs(sum-want) > 1e-9*want {
		t.Errorf("points %v plus fixed %v = %v, and the unfloored score is %v", points, fixed, sum, want)
	}
	if got := Score(outfit, st, nil); floor(sum) != got {
		t.Errorf("points and fixed floor to %d, and Score says %d", floor(sum), got)
	}
	if points[2] != 0 {
		t.Errorf("a zero-weight pair has %v points", points[2])
	}
	if points[0] == 0 || points[1] == 0 || points[3] == 0 || points[4] == 0 {
		t.Errorf("points %v leave a weighted pair empty", points)
	}
}

func TestPointsPenaliseAccessoriesOnly(t *testing.T) {
	stage := CustomStage([pairs]int{0, 100, 0, 0, 0}, nil)
	outfit := []Item{hair(100)}
	for range 4 {
		acc := hair(30)
		acc.Slot = Accessory
		outfit = append(outfit, acc)
	}
	if got, want := Points(outfit, stage)[1], 10*(100+0.95*4*30); math.Abs(got-want) > 1e-9 {
		t.Errorf("Lively points = %v, want %v", got, want)
	}
}

func TestPlacePicksByPoints(t *testing.T) {
	matches := [pairs]int8{Simple, Lively, Cute, Sexy, Cool}
	st := Stage{Attrs: matches, Weights: [pairs]float64{40, 10, 5, 5, 1}}
	dress := Item{Slot: Dress, Attrs: matches, Stats: [pairs]int{10, 300, 700, 100, 20}}
	if got := Place([]Item{dress}, st); got != (Placement{Cute, Lively}) {
		t.Errorf("Place = %+v, want Charming + Smile on Cute and Smile on Lively, the two largest point totals", got)
	}

	outfit := []Item{{Slot: Hair, Attrs: matches, Stats: [pairs]int{39, 0, 0, 0, 0}}}
	for range 4 {
		outfit = append(outfit, Item{Slot: Accessory, Attrs: matches, Stats: [pairs]int{0, 40, 0, 0, 0}})
	}
	if got := Place(outfit, st); got != (Placement{Simple, Lively}) {
		t.Errorf("Place = %+v, want Simple's 1560 points first: Lively's 1600 are 1520 after the accessory penalty", got)
	}
}

func TestPlaceBreaksTies(t *testing.T) {
	matches := [pairs]int8{Simple, Lively, Cute, Sexy, Cool}
	st := Stage{Attrs: matches, Weights: [pairs]float64{2, 4, 4, 8, 1}}
	even := Item{Slot: Top, Attrs: matches, Stats: [pairs]int{40, 20, 20, 10, 80}}
	if got := Place([]Item{even}, st); got != (Placement{Sexy, Lively}) {
		t.Errorf("Place = %+v, want equal points broken by weight (Sexy) and then by pair order (Lively before Cute)", got)
	}
	if got := Place(nil, st); got != (Placement{Sexy, Lively}) {
		t.Errorf("an empty outfit places %+v, want the heaviest weights in pair order", got)
	}
}

func TestPlaceSkipsUnweightedPairs(t *testing.T) {
	matches := [pairs]int8{Simple, Lively, Cute, Sexy, Cool}
	strong := Item{Slot: Top, Attrs: matches, Stats: [pairs]int{500, 400, 300, 200, 100}}

	if got := Place([]Item{strong}, Stage{Attrs: matches, Weights: [pairs]float64{0, 0, 0, 3, 0}}); got != (Placement{Sexy, -1}) {
		t.Errorf("one weighted pair places %+v, want Charming + Smile on Sexy and no Smile", got)
	}
	if got := Place([]Item{strong}, Stage{Attrs: matches}); got != (Placement{-1, -1}) {
		t.Errorf("a stage with no weights places %+v, want nothing", got)
	}
	if got := Place(nil, Stage{Attrs: matches, Weights: [pairs]float64{0, 0, 1, 0, 2}}); got != (Placement{Cool, Cute}) {
		t.Errorf("an empty outfit places %+v, want only the weighted pairs, heaviest first", got)
	}
}
