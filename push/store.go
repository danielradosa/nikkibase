package push

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type Entry struct {
	Endpoint string    `json:"endpoint"`
	P256dh   string    `json:"p256dh"`
	Auth     string    `json:"auth"`
	Topics   []string  `json:"topics"`
	Added    time.Time `json:"added"`
}

var ErrFull = errors.New("push: no room for more subscriptions")

type Store struct {
	mu      sync.Mutex
	path    string
	limit   int
	entries map[string]Entry
}

func OpenStore(dir string, limit int) (*Store, error) {
	s := &Store{path: filepath.Join(dir, "subscriptions.json"), limit: limit, entries: map[string]Entry{}}
	b, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	var list []Entry
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, fmt.Errorf("%s: %w", s.path, err)
	}
	for _, e := range list {
		s.entries[e.Endpoint] = e
	}
	return s, nil
}

func (s *Store) Get(endpoint string) (Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[endpoint]
	return e, ok
}

func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

func (s *Store) All() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.list()
}

func (s *Store) Put(e Entry, drop string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	before := s.copy()
	if drop != "" && drop != e.Endpoint {
		delete(s.entries, drop)
	}
	if _, ok := s.entries[e.Endpoint]; !ok && len(s.entries) >= s.limit {
		s.entries = before
		return ErrFull
	}
	s.entries[e.Endpoint] = e
	if err := s.save(); err != nil {
		s.entries = before
		return err
	}
	return nil
}

func (s *Store) Delete(endpoints ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	before := s.copy()
	changed := false
	for _, endpoint := range endpoints {
		if _, ok := s.entries[endpoint]; ok {
			delete(s.entries, endpoint)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	if err := s.save(); err != nil {
		s.entries = before
		return err
	}
	return nil
}

func (s *Store) copy() map[string]Entry {
	out := make(map[string]Entry, len(s.entries))
	for k, v := range s.entries {
		out[k] = v
	}
	return out
}

func (s *Store) list() []Entry {
	out := make([]Entry, 0, len(s.entries))
	for _, e := range s.entries {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Endpoint < out[j].Endpoint })
	return out
}

func (s *Store) save() error {
	b, err := json.Marshal(s.list())
	if err != nil {
		return err
	}
	return writeFile(s.path, b)
}

func writeFile(path string, b []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
