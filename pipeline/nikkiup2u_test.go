package pipeline

import (
	"fmt"
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

func TestParseWardrobe(t *testing.T) {
	const src = `var wardrobe = [
  ['Nikki\'s Pinky(默认粉毛)','发型','001','2','','S','','A','','A','','A','','A','','赠送','','V1.0.0','登录'],
  ['Sunset Gown(晚霞长裙)','连衣裙','372','5','SS','','','S','A','','','A','A','','Gorgeous','店','','V2.0.0','金'],
  ['Pearl Earring(珍珠耳环)','饰品-耳饰','1067','4','','A','','B','','A','','A','','B','','店','','V1.2.0','金'],
  ['nonsense','不存在的类别','1','1','','A','','A','','A','','A','','A','','','','',''],
]`

	entries, skipped, err := ParseWardrobe([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 || skipped != 1 {
		t.Fatalf("got %d entries and %d skipped, want 3 and 1", len(entries), skipped)
	}

	hair := entries[0]
	if hair.Name != "Nikki's Pinky" {
		t.Errorf("name = %q, want the English part unescaped", hair.Name)
	}
	if hair.Item.ID != 10001 {
		t.Errorf("ID = %d, want 10001", hair.Item.ID)
	}
	if hair.Item.Slot != scoring.Hair {
		t.Errorf("slot = %v, want Hair", hair.Item.Slot)
	}
	if got := hair.Item.Attrs[0]; got != scoring.Simple {
		t.Errorf("first pair = %d, want Simple", got)
	}
	if got := hair.Item.Stats[0]; got != 70 {
		t.Errorf("Simple stat = %d, want 70", got)
	}
	if got := hair.Item.Attrs[4]; got != scoring.Warm {
		t.Errorf("last pair = %d, want Warm", got)
	}

	dress := entries[1]
	if dress.Item.ID != 20372 || dress.Item.Slot != scoring.Dress {
		t.Errorf("dress = %d/%v, want 20372/Dress", dress.Item.ID, dress.Item.Slot)
	}
	if got := dress.Item.Attrs[0]; got != scoring.Gorgeous {
		t.Errorf("dress first pair = %d, want Gorgeous", got)
	}
	if got := dress.Item.Stats[0]; got != 348 {
		t.Errorf("dress Gorgeous stat = %d, want 348", got)
	}
	if len(dress.Item.Tags) != 1 {
		t.Errorf("dress tags = %v, want one", dress.Item.Tags)
	}

	earring := entries[2]
	if earring.Item.ID != 81067 {
		t.Errorf("accessory ID = %d, want 81067", earring.Item.ID)
	}
	if earring.Position != "饰品-耳饰" {
		t.Errorf("position = %q, want the accessory's own category", earring.Position)
	}
}

func TestParseStages(t *testing.T) {
	const src = `var levelsRaw = {
  '1-1': [1, 2, 3, 2, 1],
  '1-2': [3, 1.5, -3, 3, -1],
}`
	stages, _, err := ParseStages([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(stages) != 2 {
		t.Fatalf("got %d stages, want 2", len(stages))
	}

	first := stages[0]
	if first.Name != "1-1" {
		t.Errorf("name = %q, want 1-1", first.Name)
	}
	if want := [5]float64{15, 45, 30, 30, 15}; first.Stage.Weights != want {
		t.Errorf("weights = %v, want %v", first.Stage.Weights, want)
	}
	if want := [5]int8{scoring.Simple, scoring.Lively, scoring.Cute, scoring.Pure, scoring.Cool}; first.Stage.Attrs != want {
		t.Errorf("attrs = %v, want %v", first.Stage.Attrs, want)
	}

	if got := stages[1].Stage.Attrs[1]; got != scoring.Elegant {
		t.Errorf("negative weight gave %d, want Elegant", got)
	}
}

func TestStatCalibration(t *testing.T) {
	for _, tc := range []struct {
		grade string
		slot  scoring.Slot
		want  int
	}{
		{"SS", scoring.Dress, 348},
		{"SS", scoring.Hair, 87},
		{"S", scoring.Accessory, 28},
		{"unknown", scoring.Dress, 0},
	} {
		if got := Stat(tc.grade, tc.slot); got != tc.want {
			t.Errorf("Stat(%q, %v) = %d, want %d", tc.grade, tc.slot, got, tc.want)
		}
	}
}

func TestPositionsKeepsOnlyOwned(t *testing.T) {
	entries := []Entry{
		{Item: scoring.Item{ID: 1, Slot: scoring.Hair}, Position: "hair"},
		{Item: scoring.Item{ID: 2, Slot: scoring.Hair}, Position: "hair"},
		{Item: scoring.Item{ID: 3, Slot: scoring.Shoes}, Position: "shoes"},
	}
	positions, unplaced := Positions(entries, []int{1, 3, 999})
	if unplaced != 0 {
		t.Errorf("unplaced = %d, want 0", unplaced)
	}

	if len(positions) != 2 {
		t.Fatalf("got %d positions, want 2", len(positions))
	}
	if len(positions[0].Items) != 1 || positions[0].Items[0].ID != 1 {
		t.Errorf("hair position = %v, want only the owned item 1", positions[0].Items)
	}
}

func TestEveryNikkiup2uCategoryIsPlaced(t *testing.T) {
	categories := []string{"发型", "连衣裙", "上装", "下装", "外套", "袜子-袜套", "袜子-袜子", "鞋子", "妆容", "萤光之灵",
		"饰品-头饰·发饰", "饰品-头饰·头纱", "饰品-头饰·发卡", "饰品-头饰·耳朵", "饰品-耳饰", "饰品-颈饰·围巾",
		"饰品-颈饰·项链", "饰品-手饰·右", "饰品-手饰·左", "饰品-手饰·双", "饰品-手持·右", "饰品-手持·左",
		"饰品-手持·双", "饰品-腰饰", "饰品-特殊·面饰", "饰品-特殊·胸饰", "饰品-特殊·纹身", "饰品-特殊·翅膀",
		"饰品-特殊·尾巴", "饰品-特殊·前景", "饰品-特殊·后景", "饰品-特殊·顶饰", "饰品-特殊·地面", "饰品-特殊·皮肤"}
	var src strings.Builder
	src.WriteString("var wardrobe = [\n")
	for i, c := range categories {
		fmt.Fprintf(&src, "  ['Item %d(物品)','%s','%d','3','','A','','A','','A','','A','','A','','','','',''],\n", i, c, i+1)
	}
	src.WriteString("]")
	entries, skipped, err := ParseWardrobe([]byte(src.String()))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 1 || len(entries) != len(categories)-1 {
		t.Fatalf("parsed %d of %d categories, %d skipped; want every one but the spirit", len(entries), len(categories), skipped)
	}
	owned := make([]int, 0, len(entries))
	for _, e := range entries {
		owned = append(owned, e.Item.ID)
	}
	positions, unplaced := Positions(entries, owned)
	if unplaced != 0 {
		for _, e := range entries {
			if _, err := ResolvePosition(e.Position, e.Item.Slot); err != nil {
				t.Errorf("%s: %v", e.Position, err)
			}
		}
	}
	placed := 0
	for _, p := range positions {
		placed += len(p.Items)
	}
	if placed != len(entries) {
		t.Errorf("placed %d of %d items", placed, len(entries))
	}
}

func TestPositionsPlacesEachIDOnce(t *testing.T) {
	scarf := Entry{Item: scoring.Item{ID: 82786, Slot: scoring.Accessory}, Name: "Soul Collectio Bead", Position: "accessory_scarf"}
	skin := Entry{Item: scoring.Item{ID: 82786, Slot: scoring.Accessory}, Name: "Warm Ray", Position: "accessory_skin"}
	positions, left := Positions([]Entry{scarf, skin, scarf}, []int{82786})
	if len(positions) != 1 || len(positions[0].Items) != 1 || positions[0].Items[0].ID != 82786 {
		t.Fatalf("positions = %+v, want the scarf alone", positions)
	}
	if left != 2 {
		t.Errorf("left out %d rows, want 2", left)
	}
}
