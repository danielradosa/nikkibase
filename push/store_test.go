package push

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func entry(endpoint string, topics ...string) Entry {
	return Entry{Endpoint: endpoint, P256dh: "p", Auth: "a", Topics: topics, Added: time.Unix(1_800_000_000, 0).UTC()}
}

func TestStoreKeepsSubscriptionsAcrossRestarts(t *testing.T) {
	dir := t.TempDir()
	s, err := OpenStore(dir, 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Put(entry("https://fcm.googleapis.com/a", "items"), ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(entry("https://fcm.googleapis.com/b", "fixes"), ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("https://fcm.googleapis.com/a", "https://fcm.googleapis.com/missing"); err != nil {
		t.Fatal(err)
	}
	again, err := OpenStore(dir, 10)
	if err != nil {
		t.Fatal(err)
	}
	all := again.All()
	if len(all) != 1 || all[0].Endpoint != "https://fcm.googleapis.com/b" || all[0].Topics[0] != "fixes" {
		t.Fatalf("after reopening: %+v", all)
	}
	info, err := os.Stat(filepath.Join(dir, "subscriptions.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("subscriptions.json mode %v, want 0600", info.Mode().Perm())
	}
	leftovers, _ := filepath.Glob(filepath.Join(dir, "subscriptions.json.*"))
	if len(leftovers) != 0 {
		t.Errorf("temporary files left behind: %v", leftovers)
	}
}

func TestStoreRefusesNewSubscriptionsWhenFullButUpdatesOldOnes(t *testing.T) {
	s, _ := OpenStore(t.TempDir(), 1)
	if err := s.Put(entry("https://fcm.googleapis.com/a", "items"), ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(entry("https://fcm.googleapis.com/b", "items"), ""); !errors.Is(err, ErrFull) {
		t.Fatalf("a second subscription in a store of one: %v", err)
	}
	if err := s.Put(entry("https://fcm.googleapis.com/a", "stages"), ""); err != nil {
		t.Fatalf("updating the one subscription: %v", err)
	}
	if err := s.Put(entry("https://fcm.googleapis.com/c", "fixes"), "https://fcm.googleapis.com/a"); err != nil {
		t.Fatalf("replacing the one subscription: %v", err)
	}
	all := s.All()
	if len(all) != 1 || all[0].Endpoint != "https://fcm.googleapis.com/c" {
		t.Fatalf("after replacing: %+v", all)
	}
}

func TestStoreRefusesAFileItCannotRead(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "subscriptions.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenStore(dir, 10); err == nil {
		t.Fatal("a broken subscriptions.json was accepted")
	}
}

func TestStoreKeepsItsListWhenSavingFails(t *testing.T) {
	dir := t.TempDir()
	s, _ := OpenStore(dir, 10)
	if err := s.Put(entry("https://fcm.googleapis.com/a", "items"), ""); err != nil {
		t.Fatal(err)
	}
	s.path = filepath.Join(dir, "missing", "subscriptions.json")
	if err := s.Put(entry("https://fcm.googleapis.com/b", "items"), ""); err == nil {
		t.Fatal("saving into a missing folder succeeded")
	}
	if err := s.Delete("https://fcm.googleapis.com/a"); err == nil {
		t.Fatal("deleting with a missing folder succeeded")
	}
	if s.Len() != 1 {
		t.Fatalf("store holds %d after failed saves, want 1", s.Len())
	}
}
