package push

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	uaPublic = "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4"
	uaAuth   = "BTBZMqHH6r4Tts7J_aSIgg"
	fcm      = "https://fcm.googleapis.com/fcm/send/abc:def"
)

func service(t *testing.T) *Service {
	t.Helper()
	store, err := OpenStore(t.TempDir(), 3)
	if err != nil {
		t.Fatal(err)
	}
	return &Service{
		Store:   store,
		Public:  "PUBLICKEY",
		Allowed: PushHost,
		Now:     func() time.Time { return time.Unix(1_800_000_000, 0) },
		Limit:   &Limiter{Max: 5, Window: time.Hour},
	}
}

func body(endpoint, p256dh, auth string, topics []string, replaces string) string {
	req := map[string]any{
		"subscription": map[string]any{"endpoint": endpoint, "expirationTime": nil, "keys": map[string]string{"p256dh": p256dh, "auth": auth}},
		"topics":       topics,
	}
	if replaces != "" {
		req["replaces"] = replaces
	}
	b, _ := json.Marshal(req)
	return string(b)
}

func call(h http.Handler, method, path, payload string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "https://nikkibase.example"+path, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Origin", "https://nikkibase.example")
	for k, v := range headers {
		if v == "" {
			req.Header.Del(k)
		} else {
			req.Header.Set(k, v)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestKeyGivesThePublicKeyAndTopics(t *testing.T) {
	s := service(t)
	rec := call(s, http.MethodGet, "/push/key", "", nil)
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("%d %v", rec.Code, rec.Header())
	}
	var got struct {
		Key    string   `json:"key"`
		Topics []string `json:"topics"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Key != "PUBLICKEY" || strings.Join(got.Topics, ",") != "items,stages,fixes" {
		t.Fatalf("%s %v", rec.Body, err)
	}
	if rec := call(s, http.MethodPost, "/push/key", "", nil); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /push/key: %d", rec.Code)
	}
	if rec := call(s, http.MethodGet, "/push/other", "", nil); rec.Code != http.StatusNotFound {
		t.Errorf("GET /push/other: %d", rec.Code)
	}
}

func TestSubscribeStoresTheSubscriptionWithTheChosenTopics(t *testing.T) {
	s := service(t)
	rec := call(s, http.MethodPost, "/push/subscribe", body(fcm, uaPublic, uaAuth, []string{"fixes", "items", "items"}, ""), nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	e, ok := s.Store.Get(fcm)
	if !ok || strings.Join(e.Topics, ",") != "items,fixes" || e.P256dh != uaPublic || e.Auth != uaAuth || !e.Added.Equal(time.Unix(1_800_000_000, 0)) {
		t.Fatalf("%+v %v", e, ok)
	}
	later := service(t)
	later.Store = s.Store
	later.Now = func() time.Time { return time.Unix(1_900_000_000, 0) }
	if rec := call(later, http.MethodPost, "/push/subscribe", body(fcm, uaPublic, uaAuth, []string{"stages"}, ""), nil); rec.Code != http.StatusNoContent {
		t.Fatalf("changing topics: %d", rec.Code)
	}
	e, _ = s.Store.Get(fcm)
	if strings.Join(e.Topics, ",") != "stages" || !e.Added.Equal(time.Unix(1_800_000_000, 0)) {
		t.Errorf("after changing topics: %+v", e)
	}
}

func TestSubscribeRefusesAnythingButAPushServiceSubscription(t *testing.T) {
	cases := map[string]string{
		"not a push service":      body("https://evil.example/push", uaPublic, uaAuth, []string{"items"}, ""),
		"internal address":        body("https://127.0.0.1/push", uaPublic, uaAuth, []string{"items"}, ""),
		"plain http":              body("http://fcm.googleapis.com/fcm/send/x", uaPublic, uaAuth, []string{"items"}, ""),
		"another port":            body("https://fcm.googleapis.com:8443/fcm/send/x", uaPublic, uaAuth, []string{"items"}, ""),
		"user info":               body("https://a@fcm.googleapis.com/fcm/send/x", uaPublic, uaAuth, []string{"items"}, ""),
		"look-alike host":         body("https://fcm.googleapis.com.evil.example/x", uaPublic, uaAuth, []string{"items"}, ""),
		"long endpoint":           body(fcm+strings.Repeat("a", 1100), uaPublic, uaAuth, []string{"items"}, ""),
		"bad p256dh":              body(fcm, uaPublic[:40], uaAuth, []string{"items"}, ""),
		"point not on the curve":  body(fcm, "B"+strings.Repeat("A", 86), uaAuth, []string{"items"}, ""),
		"bad auth":                body(fcm, uaPublic, "c2hvcnQ", []string{"items"}, ""),
		"unknown topic":           body(fcm, uaPublic, uaAuth, []string{"items", "ads"}, ""),
		"no topics":               body(fcm, uaPublic, uaAuth, nil, ""),
		"not json":                "{",
		"replaces nothing, empty": body(fcm, uaPublic, uaAuth, nil, "https://fcm.googleapis.com/fcm/send/gone"),
	}
	for name, payload := range cases {
		s := service(t)
		s.Limit = nil
		if rec := call(s, http.MethodPost, "/push/subscribe", payload, nil); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body)
		}
		if s.Store.Len() != 0 {
			t.Errorf("%s: stored something", name)
		}
	}
}

func TestSubscribeAcceptsTheKnownPushServices(t *testing.T) {
	for _, endpoint := range []string{
		"https://fcm.googleapis.com/fcm/send/x",
		"https://updates.push.services.mozilla.com/wpush/v2/x",
		"https://web.push.apple.com/QGx",
		"https://wns2-par02p.notify.windows.com/w/?token=x",
	} {
		s := service(t)
		if rec := call(s, http.MethodPost, "/push/subscribe", body(endpoint, uaPublic, uaAuth, []string{"items"}, ""), nil); rec.Code != http.StatusNoContent {
			t.Errorf("%s: %d %s", endpoint, rec.Code, rec.Body)
		}
	}
}

func TestSubscribeOnlyTakesSameSiteJSONPosts(t *testing.T) {
	ok := body(fcm, uaPublic, uaAuth, []string{"items"}, "")
	cases := []struct {
		name    string
		method  string
		payload string
		headers map[string]string
		want    int
	}{
		{"GET", http.MethodGet, "", nil, http.StatusMethodNotAllowed},
		{"cross-site fetch", http.MethodPost, ok, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"other origin", http.MethodPost, ok, map[string]string{"Origin": "https://evil.example"}, http.StatusForbidden},
		{"form post", http.MethodPost, ok, map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, http.StatusUnsupportedMediaType},
		{"no content type", http.MethodPost, ok, map[string]string{"Content-Type": ""}, http.StatusUnsupportedMediaType},
		{"too large", http.MethodPost, `{"topics":["` + strings.Repeat("a", maxBody) + `"]}`, nil, http.StatusRequestEntityTooLarge},
		{"no fetch metadata, same origin", http.MethodPost, ok, map[string]string{"Sec-Fetch-Site": ""}, http.StatusNoContent},
		{"charset", http.MethodPost, ok, map[string]string{"Content-Type": "application/json; charset=utf-8"}, http.StatusNoContent},
	}
	for _, tc := range cases {
		s := service(t)
		if rec := call(s, tc.method, "/push/subscribe", tc.payload, tc.headers); rec.Code != tc.want {
			t.Errorf("%s: %d, want %d (%s)", tc.name, rec.Code, tc.want, rec.Body)
		}
	}
}

func TestAChangedSubscriptionKeepsItsTopics(t *testing.T) {
	s := service(t)
	call(s, http.MethodPost, "/push/subscribe", body(fcm, uaPublic, uaAuth, []string{"stages", "fixes"}, ""), nil)
	next := "https://fcm.googleapis.com/fcm/send/new"
	if rec := call(s, http.MethodPost, "/push/subscribe", body(next, uaPublic, uaAuth, nil, fcm), nil); rec.Code != http.StatusNoContent {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if _, ok := s.Store.Get(fcm); ok {
		t.Error("the old subscription is still there")
	}
	if e, ok := s.Store.Get(next); !ok || strings.Join(e.Topics, ",") != "stages,fixes" {
		t.Errorf("the new subscription: %+v %v", e, ok)
	}
}

func TestUnsubscribeRemovesTheSubscription(t *testing.T) {
	s := service(t)
	call(s, http.MethodPost, "/push/subscribe", body(fcm, uaPublic, uaAuth, []string{"items"}, ""), nil)
	if rec := call(s, http.MethodPost, "/push/unsubscribe", `{"endpoint":"`+fcm+`"}`, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if s.Store.Len() != 0 {
		t.Error("still subscribed")
	}
	if rec := call(s, http.MethodPost, "/push/unsubscribe", `{"endpoint":"`+fcm+`"}`, nil); rec.Code != http.StatusNoContent {
		t.Errorf("unsubscribing twice: %d", rec.Code)
	}
	if rec := call(s, http.MethodPost, "/push/unsubscribe", `{}`, nil); rec.Code != http.StatusBadRequest {
		t.Errorf("no endpoint: %d", rec.Code)
	}
}

func TestTooManyChangesFromOneAddressAreRefused(t *testing.T) {
	s := service(t)
	for i := range 5 {
		if rec := call(s, http.MethodPost, "/push/unsubscribe", `{"endpoint":"x"}`, map[string]string{"X-Real-IP": "203.0.113.9"}); rec.Code != http.StatusNoContent {
			t.Fatalf("request %d: %d", i+1, rec.Code)
		}
	}
	if rec := call(s, http.MethodPost, "/push/unsubscribe", `{"endpoint":"x"}`, map[string]string{"X-Real-IP": "203.0.113.9"}); rec.Code != http.StatusTooManyRequests {
		t.Errorf("sixth request: %d", rec.Code)
	}
	if rec := call(s, http.MethodPost, "/push/unsubscribe", `{"endpoint":"x"}`, map[string]string{"X-Real-IP": "203.0.113.10"}); rec.Code != http.StatusNoContent {
		t.Errorf("another address: %d", rec.Code)
	}
	s.Now = func() time.Time { return time.Unix(1_800_000_000, 0).Add(time.Hour) }
	if rec := call(s, http.MethodPost, "/push/unsubscribe", `{"endpoint":"x"}`, map[string]string{"X-Real-IP": "203.0.113.9"}); rec.Code != http.StatusNoContent {
		t.Errorf("an hour later: %d", rec.Code)
	}
}

func TestAFullStoreSaysSo(t *testing.T) {
	s := service(t)
	s.Limit = nil
	for i, host := range []string{"fcm.googleapis.com/a", "fcm.googleapis.com/b", "fcm.googleapis.com/c"} {
		if rec := call(s, http.MethodPost, "/push/subscribe", body("https://"+host, uaPublic, uaAuth, []string{"items"}, ""), nil); rec.Code != http.StatusNoContent {
			t.Fatalf("subscription %d: %d", i+1, rec.Code)
		}
	}
	if rec := call(s, http.MethodPost, "/push/subscribe", body("https://fcm.googleapis.com/d", uaPublic, uaAuth, []string{"items"}, ""), nil); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("fourth subscription in a store of three: %d", rec.Code)
	}
}

func TestLimiterForgetsOldAddressesWhenItFillsUp(t *testing.T) {
	l := &Limiter{Max: 1, Window: time.Minute}
	start := time.Unix(1_800_000_000, 0)
	for i := range limiterKeys {
		l.Allow(string(rune(i)), start)
	}
	if l.Allow("new", start) {
		t.Error("a full limiter let a new address in within the window")
	}
	if !l.Allow("new", start.Add(time.Minute)) {
		t.Error("a full limiter did not make room once the window passed")
	}
}

func TestClientIPPrefersTheProxyHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	if got := clientIP(req); got != "10.0.0.1" {
		t.Errorf("remote addr: %s", got)
	}
	req.Header.Set("X-Forwarded-For", "1.1.1.1, 203.0.113.5")
	if got := clientIP(req); got != "203.0.113.5" {
		t.Errorf("forwarded: %s", got)
	}
	req.Header.Set("X-Real-IP", "198.51.100.7")
	if got := clientIP(req); got != "198.51.100.7" {
		t.Errorf("real ip: %s", got)
	}
}
