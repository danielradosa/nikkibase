package wardrobe

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

func loadKeystream(t *testing.T) *Keystream {
	t.Helper()
	b, err := os.ReadFile("testdata/keystream.bin")
	if err != nil {
		t.Skipf("keystream fixture missing: %v", err)
	}
	ks, err := ParseKeystream(b)
	if err != nil {
		t.Fatal(err)
	}
	return ks
}

func TestDecodeSynthetic(t *testing.T) {
	ks := loadKeystream(t)
	src, err := os.ReadFile("testdata/synthetic_clothes_date")
	if err != nil {
		t.Skipf("wardrobe fixture missing: %v", err)
	}
	var want []int
	b, err := os.ReadFile("testdata/synthetic_ids.json")
	if err != nil {
		t.Skipf("expected-ids fixture missing: %v", err)
	}
	if err := json.Unmarshal(b, &want); err != nil {
		t.Fatal(err)
	}

	got, err := Decode(src, ks)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Unresolved != 0 {
		t.Errorf("Unresolved = %d, want 0", got.Unresolved)
	}
	if !slices.Equal(got.Items, want) {
		t.Errorf("decoded %d items, want %d; first mismatch at %d",
			len(got.Items), len(want), firstDiff(got.Items, want))
	}
}

func firstDiff(a, b []int) int {
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return i
		}
	}
	return min(len(a), len(b))
}

func TestDecodeRejectsGarbage(t *testing.T) {
	ks := loadKeystream(t)
	for _, tc := range []struct{ name, src string }{
		{"not base64", "this is not base64!!"},
		{"too short", "aGk="},
		{"not a table", "Zm9vYmFyYmF6cXV1eA=="},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Decode([]byte(tc.src), ks); err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}

func TestParseKeystreamRejectsBadLength(t *testing.T) {
	if _, err := ParseKeystream([]byte{4, 0, 0, 0, 1, 2}); err == nil {
		t.Error("want error, got nil")
	}
}
