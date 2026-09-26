package pipeline

import "testing"

func TestNormalizeName(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Doll Dress-Blue", "Doll Dress - Blue"},
		{"Athena's Armor-Purple", "Athena's Armor - Purple"},
		{"Song of Clouds·Mist", "Song of Clouds · Mist"},
		{"Sweet&Smooth-Joy", "Sweet & Smooth - Joy"},
		{"Soundless  Yearning", "Soundless Yearning"},
		{"Dreams of  Past - Shadow", "Dreams of Past - Shadow"},

		{"High-top Sneakers", "High-top Sneakers"},
		{"Long-Haired Doll", "Long-Haired Doll"},
		{"Mid-Autumn Night", "Mid-Autumn Night"},
		{"Cross-country Boots", "Cross-country Boots"},
		{"Multi-functional Bag", "Multi-functional Bag"},
		{"Anti-destiny Blade", "Anti-destiny Blade"},
		{"Two-piece Suit", "Two-piece Suit"},

		{"Trimmed T-shirt", "Trimmed T-shirt"},
		{"V-neck Sweater", "V-neck Sweater"},
		{"Bi-thread Chain", "Bi-thread Chain"},
		{"B&W Wristband", "B&W Wristband"},

		{"1-1", "1-1"},
		{"2-Side 1", "2-Side 1"},
		{"Lunar 1-6", "Lunar 1-6"},
		{"Ace-Assassins Union", "Ace - Assassins Union"},
		{"Fu Su-Simple&Cool Cloud", "Fu Su - Simple & Cool Cloud"},

		{"Doll Dress - Blue", "Doll Dress - Blue"},
		{"Nikki's Pinky", "Nikki's Pinky"},
	} {
		if got := NormalizeName(c.in); got != c.want {
			t.Errorf("NormalizeName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeNameIsIdempotent(t *testing.T) {
	for _, in := range []string{"Doll Dress-Blue", "Song of Clouds·Mist", "Sweet&Smooth", "High-top"} {
		once := NormalizeName(in)
		if twice := NormalizeName(once); twice != once {
			t.Errorf("NormalizeName(%q) = %q, then %q", in, once, twice)
		}
	}
}

func TestNormalizeNameLeavesHyphenatedPhrases(t *testing.T) {
	for _, in := range []string{
		"Heart-to-Heart", "Merry-go-round", "Go-As-You-Please", "Three-in-one",
		"End-of-time Frenzy", "Around-the-World", "Passers-By-Red", "9-6-1",
	} {
		if got := NormalizeName(in); got != in {
			t.Errorf("NormalizeName(%q) = %q, want it unchanged", in, got)
		}
	}
}

func TestNormalizeNameKeepsLowercaseCompoundsJoined(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Tea-picking Girl", "Tea-picking Girl"},
		{"Silver-wings Hero", "Silver-wings Hero"},
		{"Snow Hare Lop-ear·Rare", "Snow Hare Lop-ear · Rare"},
		{"Honey-gold Sunlight-Blue", "Honey-gold Sunlight - Blue"},
	} {
		if got := NormalizeName(c.in); got != c.want {
			t.Errorf("NormalizeName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeNameFollowsTheOtherSourcesSpelling(t *testing.T) {
	for _, c := range []struct {
		in        string
		spellings []string
		want      string
	}{
		{"Honey-Soaked Song", []string{"Honey-Soaked Song"}, "Honey-Soaked Song"},
		{"Honey-Soaked Song", nil, "Honey - Soaked Song"},
		{"Honey-Soaked Song", []string{"", "Honey Soaked Song"}, "Honey - Soaked Song"},
		{"Honey-Soaked Song", []string{"Honey - Soaked Song", "Honey-Soaked Song"}, "Honey-Soaked Song"},
		{"Doll Dress-Blue", []string{"Doll Dress·Blue", "Doll Dress - Blue"}, "Doll Dress - Blue"},
		{"Fair Lady-Gorgeous", []string{"Fair Lady-Gorgeous"}, "Fair Lady-Gorgeous"},
		{"Falling Snow-White Sakura", []string{"", "Falling Snow-White Sakura"}, "Falling Snow-White Sakura"},
		{"Day-Night Concerto", []string{"day-night concerto"}, "Day-Night Concerto"},
		{"Day-Night Rose-Red", []string{"Day-Night Rose·Red"}, "Day-Night Rose - Red"},
		{"Fair Lady-Gorgeous", []string{"Fair Lady-Gorgeousness"}, "Fair Lady - Gorgeous"},
		{"Fair Lady-Gorgeous", []string{"Fair Milady-Gorgeous"}, "Fair Lady - Gorgeous"},
		{"Rose-Red", []string{"Rose-Redwood Rose-Red"}, "Rose-Red"},
	} {
		got := NormalizeName(c.in, c.spellings...)
		if got != c.want {
			t.Errorf("NormalizeName(%q, %q) = %q, want %q", c.in, c.spellings, got, c.want)
		}
		if again := NormalizeName(got, c.spellings...); again != got {
			t.Errorf("NormalizeName(%q, %q) = %q, then %q", c.in, c.spellings, got, again)
		}
	}
}

func TestNormalizeNameSpacesHyphensSpacedOnOneSide(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Moonlight -White", "Moonlight - White"},
		{"Palace- Moonlight", "Palace - Moonlight"},
		{"Whispers of the Night- Foamy", "Whispers of the Night - Foamy"},
		{"Snow Scarf- Epic", "Snow Scarf - Epic"},
		{"Lantern Elf -Illusion Blue", "Lantern Elf - Illusion Blue"},
		{"Sun -Moon- Star", "Sun - Moon - Star"},
		{"-Lead Word", "-Lead Word"},
		{"Trailing Word-", "Trailing Word-"},
		{"Endless -- Dash", "Endless -- Dash"},
		{"Moonlight - White", "Moonlight - White"},
	} {
		got := NormalizeName(c.in)
		if got != c.want {
			t.Errorf("NormalizeName(%q) = %q, want %q", c.in, got, c.want)
		}
		if again := NormalizeName(got); again != got {
			t.Errorf("NormalizeName(%q) = %q, then %q", c.in, got, again)
		}
	}
}

func TestNormalizeNameSpacesWhereTheOtherSourcesSeparate(t *testing.T) {
	for _, c := range []struct {
		in        string
		spellings []string
		want      string
	}{
		{"Fox Talk-Me", []string{"Fox Talk·Me"}, "Fox Talk - Me"},
		{"Fox Talk-Me", []string{"", "fox talk - me"}, "Fox Talk - Me"},
		{"Fox Talk-Me", []string{"Fox Talk -Me"}, "Fox Talk - Me"},
		{"Stripe Sun-top-White", []string{"Stripe Sun-top·White"}, "Stripe Sun-top - White"},
		{"Passers-By-Red", []string{"Passers-By · Red"}, "Passers-By - Red"},
		{"Trimmed T-shirt-Blue", []string{"Trimmed T-shirt - Blue"}, "Trimmed T-shirt - Blue"},
		{"Waist-Length-White", []string{"Waist-Length·White", "Waist-Length·White"}, "Waist-Length - White"},
		{"Fox Talk-Me", []string{"Fox Talk- Me"}, "Fox Talk - Me"},
		{"Red-Blue-Green", []string{"Red · Blue · Green"}, "Red - Blue - Green"},
		{"Red-Blue-Green", []string{"red - blue - green"}, "Red - Blue - Green"},
		{"Red-Blue-Green", []string{"Red·Blue-Green"}, "Red - Blue-Green"},
		{"Nine-Tail-Fox", []string{"Nine-Tail · Fox", "Nine · Tail-Fox"}, "Nine - Tail - Fox"},
		{"Nine-Tail-Fox Coat", []string{"Nine·Tail·Fox Coat"}, "Nine - Tail - Fox Coat"},
		{"Nine-Tail-Fox", []string{"Nine·Tail·Foxes"}, "Nine-Tail-Fox"},

		{"Fox Talk-Me", nil, "Fox Talk-Me"},
		{"Fox Talk-Me", []string{"Fox Talk-Me"}, "Fox Talk-Me"},
		{"Fox Talk-Me", []string{"Fox Talk·Me", "Fox Talk-Me"}, "Fox Talk-Me"},
		{"Fox Talk-Me", []string{"Fox Stalk·Me"}, "Fox Talk-Me"},
		{"Fox Talk-Me", []string{"Fox Talk·Messy"}, "Fox Talk-Me"},
		{"Passers-By-Red", []string{"Passers·By-Red"}, "Passers - By-Red"},
		{"Heart-to-Heart", []string{"Heart-to-Heart·Rare"}, "Heart-to-Heart"},
		{"Tail--Fox", []string{"Tail · -Fox"}, "Tail--Fox"},
	} {
		got := NormalizeName(c.in, c.spellings...)
		if got != c.want {
			t.Errorf("NormalizeName(%q, %q) = %q, want %q", c.in, c.spellings, got, c.want)
		}
		if again := NormalizeName(got, c.spellings...); again != got {
			t.Errorf("NormalizeName(%q, %q) = %q, then %q", c.in, c.spellings, got, again)
		}
	}
}
