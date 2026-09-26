package scoring

import (
	"cmp"
	"math"
	"slices"
)

const (
	Gorgeous = iota
	Simple
	Elegant
	Lively
	Mature
	Cute
	Sexy
	Pure
	Warm
	Cool
)

const pairs = 5

type Slot uint8

const (
	Hair Slot = iota
	Dress
	Coat
	Top
	Bottom
	Hosiery
	Shoes
	Makeup
	Accessory
	Spirit
)

var tagFactor = [...]float64{
	Hair: 0.5, Dress: 2, Coat: 0.2, Top: 1, Bottom: 1,
	Hosiery: 0.3, Shoes: 0.4, Makeup: 0.1, Accessory: 0.2, Spirit: 0.2,
}

type Item struct {
	ID        int
	Slot      Slot
	Attrs     [pairs]int8
	Stats     [pairs]int
	Tags      []int
	FlatBonus int
}

type Stage struct {
	Attrs   [pairs]int8
	Weights [pairs]float64
	Tags    map[int]int
}

const (
	CharmingSmile = 1.778
	SmileOnly     = 1.27
)

type Skills map[int]float64

func (s Skills) mult(code int8) float64 {
	if m, ok := s[int(code)]; ok {
		return m
	}
	return 1
}

type Placement struct{ CharmSmile, Smile int }

type Levels struct{ Charming, Smile int }

var MaxLevels = Levels{Charming: 9, Smile: 9}

var (
	charmingPercent = [...]int{0, 24, 26, 28, 30, 32, 34, 36, 38, 40}
	smilePercent    = [...]int{0, 15, 17, 19, 20, 22, 24, 25, 26, 27}
)

func (l Levels) Valid() bool {
	return l.Charming >= 0 && l.Charming < len(charmingPercent) && l.Smile >= 0 && l.Smile < len(smilePercent)
}

func (l Levels) None() bool {
	return l.Charming == 0 && l.Smile == 0
}

func (p Placement) Skills() Skills {
	return p.SkillsAt(MaxLevels)
}

func (p Placement) SkillsAt(l Levels) Skills {
	c, s := charmingPercent[l.Charming], smilePercent[l.Smile]
	sk := Skills{}
	if p.CharmSmile >= 0 && c+s > 0 {
		sk[p.CharmSmile] = float64((100+c)*(100+s)) / 10000
	}
	if p.Smile >= 0 && p.Smile != p.CharmSmile && s > 0 {
		sk[p.Smile] = float64(100+s) / 100
	}
	return sk
}

func Points(outfit []Item, st Stage) [pairs]float64 {
	var main, accessory [pairs]float64
	accessories := 0
	for _, it := range outfit {
		sum := &main
		if it.Slot == Accessory {
			sum = &accessory
			accessories++
		}
		for p := range pairs {
			if it.Attrs[p] != st.Attrs[p] {
				continue
			}
			sum[p] += st.Weights[p] * float64(it.Stats[p])
		}
	}
	ratio := AccessoryPenalty(accessories)
	var points [pairs]float64
	for p := range pairs {
		points[p] = main[p] + ratio*accessory[p]
	}
	return points
}

func Place(outfit []Item, st Stage) Placement {
	return PlaceFrom(Points(outfit, st), st)
}

func PlaceFrom(points [pairs]float64, st Stage) Placement {
	var weighted [pairs]int
	order := weighted[:0]
	for p := range pairs {
		if st.Weights[p] > 0 {
			order = append(order, p)
		}
	}
	slices.SortStableFunc(order, func(a, b int) int {
		if c := cmp.Compare(points[b], points[a]); c != 0 {
			return c
		}
		return cmp.Compare(st.Weights[b], st.Weights[a])
	})
	placed := Placement{CharmSmile: -1, Smile: -1}
	if len(order) > 0 {
		placed.CharmSmile = int(st.Attrs[order[0]])
	}
	if len(order) > 1 {
		placed.Smile = int(st.Attrs[order[1]])
	}
	return placed
}

func Score(outfit []Item, st Stage, sk Skills) int {
	return floor(unfloored(outfit, st, sk))
}

func unfloored(outfit []Item, st Stage, sk Skills) float64 {
	var total, accessoryAttrs float64
	accessories := 0

	for _, it := range outfit {
		scaled, fixed := Contribution(it, st, sk)
		total += fixed
		if it.Slot == Accessory {
			accessoryAttrs += scaled
			accessories++
			continue
		}
		total += scaled
	}

	return total + AccessoryPenalty(accessories)*accessoryAttrs
}

func Floor(x float64) int {
	return floor(x)
}

func floor(x float64) int {
	a := math.Abs(x)
	ulp := math.Nextafter(a, math.Inf(1)) - a
	return int(math.Floor(x + 64*ulp))
}

func Contribution(it Item, st Stage, sk Skills) (scaled, fixed float64) {
	for p := range pairs {
		if it.Attrs[p] != st.Attrs[p] {
			continue
		}
		scaled += st.Weights[p] * sk.mult(st.Attrs[p]) * float64(it.Stats[p])
	}
	for _, t := range it.Tags {
		if award, ok := st.Tags[t]; ok {
			fixed += float64(award) * tagFactor[it.Slot]
		}
	}
	if it.Slot == Spirit {
		fixed += float64(it.FlatBonus)
	}
	return scaled, fixed
}

func AccessoryPenalty(worn int) float64 {
	ratios := [...]float64{4: 0.95, 5: 0.90, 6: 0.825, 7: 0.75, 8: 0.70, 9: 0.65,
		10: 0.60, 11: 0.55, 12: 0.51, 13: 0.475, 14: 0.45, 15: 0.425}
	switch {
	case worn <= 3:
		return 1
	case worn >= 16:
		return 0.40
	default:
		return ratios[worn]
	}
}

func CustomStage(sliders [pairs]int, tags map[int]int) Stage {
	st := Stage{Tags: tags}
	var attrv [pairs]float64
	var sum float64
	for p, v := range sliders {
		attrv[p] = math.Abs(float64(v)) / 2
		sum += attrv[p]
		st.Attrs[p] = int8(p * 2)
		if v > 0 {
			st.Attrs[p]++
		}
	}
	if sum == 0 {
		return st
	}
	for p, a := range attrv {
		st.Weights[p] = a / sum * 10
	}
	return st
}

func SlotLimit(s Slot) int {
	switch s {
	case Accessory:
		return 24
	case Hosiery:
		return 2
	default:
		return 1
	}
}

func SlotSize(s Slot) float64 { return tagFactor[s] }
