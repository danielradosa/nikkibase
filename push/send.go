package push

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type Sender struct {
	VAPID   *VAPID
	Client  *http.Client
	Allowed func(string) bool
	TTL     time.Duration
	Workers int
	Now     func() time.Time
}

func NewClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

type Result struct {
	Sent    int
	Skipped int
	Gone    []string
	Failed  int
}

func (s *Sender) Send(ctx context.Context, entries []Entry, message func(Entry) (Message, bool)) Result {
	var (
		mu  sync.Mutex
		res Result
		wg  sync.WaitGroup
	)
	jobs := make(chan Entry)
	workers := max(s.Workers, 1)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for e := range jobs {
				msg, ok := message(e)
				if !ok {
					mu.Lock()
					res.Skipped++
					mu.Unlock()
					continue
				}
				status, err := s.one(ctx, e, msg)
				mu.Lock()
				switch {
				case err == nil && status >= 200 && status < 300:
					res.Sent++
				case err == nil && (status == http.StatusNotFound || status == http.StatusGone):
					res.Gone = append(res.Gone, e.Endpoint)
				default:
					res.Failed++
				}
				mu.Unlock()
			}
		}()
	}
	for _, e := range entries {
		select {
		case jobs <- e:
		case <-ctx.Done():
		}
	}
	close(jobs)
	wg.Wait()
	return res
}

func (s *Sender) one(ctx context.Context, e Entry, msg Message) (int, error) {
	if err := CheckEndpoint(e.Endpoint, s.Allowed); err != nil {
		return 0, err
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}
	body, err := Encrypt(payload, e.P256dh, e.Auth)
	if err != nil {
		return 0, err
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	auth, err := s.VAPID.Authorization(e.Endpoint, now())
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.Endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("TTL", strconv.Itoa(int(s.TTL/time.Second)))
	req.Header.Set("Urgency", "normal")
	req.Header.Set("Topic", "nikkibase-data")
	resp, err := s.Client.Do(req)
	if err != nil {
		return 0, err
	}
	io.CopyN(io.Discard, resp.Body, 4096)
	resp.Body.Close()
	return resp.StatusCode, nil
}

func Announce(ctx context.Context, webRoot, dir string, store *Store, sender *Sender, logf func(string, ...any)) error {
	cur, err := ReadRelease(webRoot)
	if err != nil {
		return fmt.Errorf("reading the data: %w", err)
	}
	path := filepath.Join(dir, "release.json")
	prev, err := LoadRelease(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := cur.Save(path); err != nil {
			return err
		}
		logf("push: data %s recorded, nothing to announce yet", cur.Version)
		return nil
	}
	if err != nil {
		return err
	}
	changes := Compare(prev, cur)
	if changes.Empty() {
		if prev.Version != cur.Version {
			return cur.Save(path)
		}
		return nil
	}
	if err := cur.Save(path); err != nil {
		return err
	}
	entries := store.All()
	res := sender.Send(ctx, entries, func(e Entry) (Message, bool) { return changes.Message(e.Topics) })
	if err := store.Delete(res.Gone...); err != nil {
		logf("push: removing %d ended subscriptions: %v", len(res.Gone), err)
	}
	logf("push: data %s announced: %d sent, %d not interested, %d ended, %d failed (%d new items, %d new stages, %d items and %d stages fixed)",
		cur.Version, res.Sent, res.Skipped, len(res.Gone), res.Failed, len(changes.NewItems), len(changes.NewStages), changes.FixedItems, changes.FixedStages)
	return nil
}
