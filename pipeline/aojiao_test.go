package pipeline

import (
	"testing"

	"github.com/danielradosa/nikkibase/core/scoring"
)

const packedSrc = `
var category = ['发型','连衣裙','外套','上装','下装','袜子-袜套','袜子-袜子','鞋子','饰品-头饰·发饰','饰品-耳饰','饰品-颈饰·围巾','饰品-手饰·右','饰品-手饰·左','饰品-手饰·双','饰品-手持·右','饰品-手持·左','饰品-手持·双','饰品-腰饰','饰品-特殊·面饰','饰品-特殊·胸饰','饰品-特殊·纹身','饰品-特殊·翅膀','饰品-特殊·尾巴','饰品-特殊·前景','饰品-特殊·后景','饰品-特殊·顶饰','饰品-特殊·地面','饰品-皮肤','饰品-头饰·头纱','饰品-头饰·发卡','饰品-头饰·耳朵','饰品-特殊·备用','妆容','萤光之灵'];
var code2tag = ['中性风','大小姐','欧式古典','中式古典','舞者','波西米亚','乐队风','和风','医务使者','轻熟风'];
var codewardrobe = [
'烈火女神|1J|4aS7|2||0|1/2/3|1',
'剑魄|X1|14t-|华丽+200||S|~绫罗1-1|21',
];
`

func TestParsePacked(t *testing.T) {
	entries, stats, err := ParsePacked([]byte(packedSrc), nil)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Rows != 2 || stats.Parsed != 2 {
		t.Fatalf("rows=%d parsed=%d, want 2 and 2", stats.Rows, stats.Parsed)
	}
	byID := map[int]Entry{}
	for _, e := range entries {
		byID[e.Item.ID] = e
	}

	dress, ok := byID[20019]
	if !ok {
		t.Fatalf("dress row did not decode to 20019; got %v", entries[0].Item.ID)
	}
	if dress.Item.Slot != scoring.Dress {
		t.Errorf("slot = %v, want Dress", dress.Item.Slot)
	}
	wantAttrs := [5]int8{scoring.Gorgeous, scoring.Elegant, scoring.Mature, scoring.Pure, scoring.Warm}
	if dress.Item.Attrs != wantAttrs {
		t.Errorf("attrs = %v, want %v", dress.Item.Attrs, wantAttrs)
	}
	if want := [5]string{"S", "S", "A", "S", "S"}; dress.Grades != want {
		t.Errorf("grades = %v, want %v", dress.Grades, want)
	}
	if dress.Item.FlatBonus != 0 {
		t.Errorf("a dress got a flat bonus of %d", dress.Item.FlatBonus)
	}

	spirit, ok := byID[880001]
	if !ok {
		t.Fatal("spirit row did not decode to 880001")
	}
	if spirit.Item.Slot != scoring.Spirit {
		t.Errorf("slot = %v, want Spirit", spirit.Item.Slot)
	}
	if spirit.Item.FlatBonus != 200 {
		t.Errorf("flat bonus = %d, want 200", spirit.Item.FlatBonus)
	}
	if len(spirit.Item.Tags) != 0 {
		t.Errorf("the bonus field was read as tags: %v", spirit.Item.Tags)
	}
	if stats.Bonuses != 1 {
		t.Errorf("Bonuses = %d, want 1", stats.Bonuses)
	}
}

func TestParsePackedSkipsItemsTheGlobalGameLacks(t *testing.T) {
	entries, stats, err := ParsePacked([]byte(packedSrc), map[int]bool{20019: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Item.ID != 20019 {
		t.Fatalf("got %d entries, want only 20019", len(entries))
	}
	if stats.NotGlobal != 1 {
		t.Errorf("NotGlobal = %d, want 1", stats.NotGlobal)
	}
}

func TestPackedCodes(t *testing.T) {
	for _, c := range []struct {
		in   string
		want int
	}{
		{"0", 0}, {"9", 9}, {"A", 10}, {"Z", 35}, {"a", 36}, {"z", 61},
		{"-", 62}, {"_", 63}, {"10", 64}, {"1J", 83},
	} {
		if got := code2num(c.in); got != c.want {
			t.Errorf("code2num(%q) = %d, want %d", c.in, got, c.want)
		}
	}
	if first, second := packedCodes(4); first != scoring.Cool || second != scoring.Warm {
		t.Errorf("pair 4 = (%d,%d), want (Cool,Warm)", first, second)
	}
	if first, second := packedCodes(0); first != scoring.Gorgeous || second != scoring.Simple {
		t.Errorf("pair 0 = (%d,%d), want (Gorgeous,Simple)", first, second)
	}
}
