package push

import (
	"context"
	"crypto/ecdh"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var Topics = []string{"items", "stages", "fixes"}

const (
	maxBody     = 4096
	maxEndpoint = 1024
)

func PushHost(host string) bool {
	host = strings.ToLower(host)
	return host == "fcm.googleapis.com" ||
		host == "updates.push.services.mozilla.com" ||
		host == "web.push.apple.com" ||
		strings.HasSuffix(host, ".notify.windows.com")
}

func CheckEndpoint(endpoint string, allowed func(string) bool) error {
	if len(endpoint) > maxEndpoint {
		return errors.New("endpoint too long")
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	if u.Scheme != "https" || u.User != nil || u.Opaque != "" || (u.Port() != "" && u.Port() != "443") {
		return errors.New("endpoint must be a plain https URL")
	}
	if !allowed(u.Hostname()) {
		return errors.New("endpoint is not a known push service")
	}
	return nil
}

func checkKeys(p256dh, auth string) error {
	pub, err := decode(p256dh)
	if err != nil {
		return err
	}
	if _, err := ecdh.P256().NewPublicKey(pub); err != nil {
		return err
	}
	secret, err := decode(auth)
	if err != nil {
		return err
	}
	if len(secret) != 16 {
		return errors.New("auth must be 16 bytes")
	}
	return nil
}

func cleanTopics(in []string) ([]string, error) {
	var out []string
	for _, topic := range Topics {
		if has(in, topic) {
			out = append(out, topic)
		}
	}
	for _, t := range in {
		if !has(Topics, t) {
			return nil, errors.New("unknown topic " + t)
		}
	}
	return out, nil
}

type Service struct {
	Store   *Store
	Public  string
	Allowed func(string) bool
	Now     func() time.Time
	Limit   *Limiter
	Confirm func(context.Context, Entry) error
}

type subscribeRequest struct {
	Subscription struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
	} `json:"subscription"`
	Topics   []string `json:"topics"`
	Replaces string   `json:"replaces"`
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	switch r.URL.Path {
	case "/push/key":
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"key": s.Public, "topics": Topics})
	case "/push/subscribe":
		s.change(w, r, true, s.subscribe)
	case "/push/unsubscribe":
		s.change(w, r, false, s.unsubscribe)
	default:
		http.NotFound(w, r)
	}
}

func (s *Service) change(w http.ResponseWriter, r *http.Request, limited bool, do func(ctx context.Context, body []byte) (int, string)) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !sameOrigin(r) {
		http.Error(w, "cross-site request", http.StatusForbidden)
		return
	}
	if ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); ct != "application/json" {
		http.Error(w, "want application/json", http.StatusUnsupportedMediaType)
		return
	}
	if limited && s.Limit != nil && !s.Limit.Allow(clientIP(r), s.now()) {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
	if err != nil {
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return
	}
	status, msg := do(r.Context(), body)
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	http.Error(w, msg, status)
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) subscribe(ctx context.Context, body []byte) (int, string) {
	var req subscribeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return http.StatusBadRequest, "bad json"
	}
	sub := req.Subscription
	if err := CheckEndpoint(sub.Endpoint, s.Allowed); err != nil {
		return http.StatusBadRequest, err.Error()
	}
	if err := checkKeys(sub.Keys.P256dh, sub.Keys.Auth); err != nil {
		return http.StatusBadRequest, "bad keys"
	}
	topics, err := cleanTopics(req.Topics)
	if err != nil {
		return http.StatusBadRequest, err.Error()
	}
	if len(topics) == 0 && req.Replaces != "" {
		if old, ok := s.Store.Get(req.Replaces); ok {
			topics = old.Topics
		}
	}
	if len(topics) == 0 {
		return http.StatusBadRequest, "no topics"
	}
	entry := Entry{Endpoint: sub.Endpoint, P256dh: sub.Keys.P256dh, Auth: sub.Keys.Auth, Topics: topics, Added: s.now().UTC().Truncate(time.Second)}
	old, known := s.Store.Get(sub.Endpoint)
	if known && old.P256dh == entry.P256dh && old.Auth == entry.Auth {
		entry.Added = old.Added
	} else {
		if s.Store.Len() >= s.Store.limit && req.Replaces == "" {
			return http.StatusServiceUnavailable, "no room for more subscriptions"
		}
		if s.Confirm != nil {
			if err := s.Confirm(ctx, entry); err != nil {
				log.Printf("push: a new subscription was refused by its push service: %v", err)
				return http.StatusBadRequest, "the push service refused this subscription"
			}
		}
	}
	err = s.Store.Put(entry, req.Replaces)
	if errors.Is(err, ErrFull) {
		return http.StatusServiceUnavailable, "no room for more subscriptions"
	}
	if err != nil {
		log.Printf("push: saving a subscription: %v", err)
		return http.StatusServiceUnavailable, "could not save"
	}
	return http.StatusNoContent, ""
}

func (s *Service) unsubscribe(_ context.Context, body []byte) (int, string) {
	var req struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.Unmarshal(body, &req); err != nil || req.Endpoint == "" {
		return http.StatusBadRequest, "bad json"
	}
	if err := s.Store.Delete(req.Endpoint); err != nil {
		log.Printf("push: removing a subscription: %v", err)
		return http.StatusServiceUnavailable, "could not save"
	}
	return http.StatusNoContent, ""
}

func sameOrigin(r *http.Request) bool {
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && u.Host == r.Host
}

func clientIP(r *http.Request) string {
	raw := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if raw == "" {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			raw = strings.TrimSpace(parts[len(parts)-1])
		}
	}
	if raw == "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		raw = host
	}
	ip := net.ParseIP(raw)
	if ip == nil {
		return raw
	}
	if ip.To4() != nil {
		return ip.To4().String()
	}
	return ip.Mask(net.CIDRMask(64, 128)).String() + "/64"
}

type Limiter struct {
	mu     sync.Mutex
	Max    int
	Window time.Duration
	seen   map[string]*window
}

type window struct {
	start time.Time
	n     int
}

const limiterKeys = 10000

func (l *Limiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.seen == nil {
		l.seen = map[string]*window{}
	}
	w := l.seen[key]
	if w == nil && len(l.seen) >= limiterKeys {
		oldest, first := "", time.Time{}
		for k, v := range l.seen {
			if now.Sub(v.start) >= l.Window {
				delete(l.seen, k)
			} else if oldest == "" || v.start.Before(first) {
				oldest, first = k, v.start
			}
		}
		if len(l.seen) >= limiterKeys {
			delete(l.seen, oldest)
		}
	}
	if w == nil || now.Sub(w.start) >= l.Window {
		l.seen[key] = &window{start: now, n: 1}
		return true
	}
	if w.n >= l.Max {
		return false
	}
	w.n++
	return true
}
