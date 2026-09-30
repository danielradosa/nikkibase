package push

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func decryptForTest(t *testing.T, body []byte, ua *ecdh.PrivateKey, auth []byte) []byte {
	t.Helper()
	salt := body[:16]
	if binary.BigEndian.Uint32(body[16:20]) != recordSize {
		t.Fatalf("record size %d", binary.BigEndian.Uint32(body[16:20]))
	}
	n := int(body[20])
	keyID := body[21 : 21+n]
	as, err := ecdh.P256().NewPublicKey(keyID)
	if err != nil {
		t.Fatal(err)
	}
	shared, err := ua.ECDH(as)
	if err != nil {
		t.Fatal(err)
	}
	ikm, _ := hkdf.Key(sha256.New, shared, auth, "WebPush: info\x00"+string(ua.PublicKey().Bytes())+string(keyID), 32)
	prk, _ := hkdf.Extract(sha256.New, ikm, salt)
	cek, _ := hkdf.Expand(sha256.New, prk, "Content-Encoding: aes128gcm\x00", 16)
	nonce, _ := hkdf.Expand(sha256.New, prk, "Content-Encoding: nonce\x00", 12)
	block, _ := aes.NewCipher(cek)
	gcm, _ := cipher.NewGCM(block)
	plain, err := gcm.Open(nil, nonce, body[21+n:], nil)
	if err != nil {
		t.Fatal(err)
	}
	plain = []byte(strings.TrimRight(string(plain), "\x00"))
	if plain[len(plain)-1] != 2 {
		t.Fatalf("no last-record delimiter")
	}
	return plain[:len(plain)-1]
}

func TestTheRFCExampleDecryptsToItsPlaintext(t *testing.T) {
	ua, _ := ecdh.P256().NewPrivateKey(mustDecode(t, "q1dXpw3UpT5VOmu_cf_v6ih07Aems3njxI-JWgLcM94"))
	msg := mustDecode(t, "DGv6ra1nlYgDCS1FRnbzlwAAEABBBP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A_yl95bQpu6cVPTpK4Mqgkf1CXztLVBSt2Ks3oZwbuwXPXLWyouBWLVWGNWQexSgSxsj_Qulcy4a-fN")
	if got := string(decryptForTest(t, msg, ua, mustDecode(t, uaAuth))); got != "When I grow up, I want to be a watermelon" {
		t.Fatalf("%q", got)
	}
}

type fakePush struct {
	mu       sync.Mutex
	status   map[string]int
	requests []*http.Request
	bodies   map[string][]byte
}

func (f *fakePush) RoundTrip(req *http.Request) (*http.Response, error) {
	b, _ := io.ReadAll(req.Body)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, req)
	if f.bodies == nil {
		f.bodies = map[string][]byte{}
	}
	f.bodies[req.URL.String()] = b
	status := http.StatusCreated
	if s, ok := f.status[req.URL.String()]; ok {
		status = s
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}, Request: req}, nil
}

type subscriber struct {
	key  *ecdh.PrivateKey
	auth []byte
}

func newSubscriber(t *testing.T) (subscriber, string, string) {
	t.Helper()
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := mustDecode(t, uaAuth)
	return subscriber{key, auth}, b64.EncodeToString(key.PublicKey().Bytes()), uaAuth
}

func sender(t *testing.T, fake *fakePush) *Sender {
	t.Helper()
	private, _, err := NewVAPIDKey()
	if err != nil {
		t.Fatal(err)
	}
	v, err := ParseVAPID(private, "mailto:nikkibaseproject@gmail.com")
	if err != nil {
		t.Fatal(err)
	}
	return &Sender{
		VAPID:   v,
		Client:  &http.Client{Transport: fake},
		Allowed: PushHost,
		TTL:     72 * time.Hour,
		Workers: 3,
		Now:     func() time.Time { return time.Unix(1_800_000_000, 0) },
	}
}

func TestSendDeliversAnEncryptedSignedMessage(t *testing.T) {
	fake := &fakePush{status: map[string]int{}}
	s := sender(t, fake)
	sub, p256dh, auth := newSubscriber(t)
	res := s.Send(context.Background(), []Entry{{Endpoint: fcm, P256dh: p256dh, Auth: auth, Topics: []string{"items"}}},
		func(Entry) (Message, bool) {
			return Message{Title: "New in NikkiBase", Body: "2 new items.", Tag: "nikkibase-data", URL: "/"}, true
		})
	if res.Sent != 1 || res.Failed != 0 {
		t.Fatalf("%+v", res)
	}
	req := fake.requests[0]
	if req.Method != http.MethodPost || req.Header.Get("Content-Encoding") != "aes128gcm" || req.Header.Get("TTL") != "259200" ||
		req.Header.Get("Topic") != "nikkibase-data" || req.Header.Get("Urgency") != "normal" {
		t.Errorf("headers %v", req.Header)
	}
	token, key, ok := strings.Cut(strings.TrimPrefix(req.Header.Get("Authorization"), "vapid t="), ", k=")
	if !ok || key != s.VAPID.Public {
		t.Fatalf("authorization %q", req.Header.Get("Authorization"))
	}
	claims, err := verifyJWT(token, mustDecode(t, key))
	if err != nil || claims["aud"] != "https://fcm.googleapis.com" {
		t.Fatalf("token %v %v", claims, err)
	}
	var got Message
	if err := json.Unmarshal(decryptForTest(t, fake.bodies[fcm], sub.key, sub.auth), &got); err != nil {
		t.Fatal(err)
	}
	if got != (Message{Title: "New in NikkiBase", Body: "2 new items.", Tag: "nikkibase-data", URL: "/"}) {
		t.Errorf("decrypted %+v", got)
	}
}

func TestSendSortsOutEndedFailedAndUninterestedSubscriptions(t *testing.T) {
	_, p256dh, auth := newSubscriber(t)
	gone := "https://fcm.googleapis.com/fcm/send/gone"
	broken := "https://updates.push.services.mozilla.com/wpush/v2/broken"
	fake := &fakePush{status: map[string]int{gone: http.StatusGone, broken: http.StatusInternalServerError}}
	s := sender(t, fake)
	entries := []Entry{
		{Endpoint: fcm, P256dh: p256dh, Auth: auth, Topics: []string{"items"}},
		{Endpoint: gone, P256dh: p256dh, Auth: auth, Topics: []string{"items"}},
		{Endpoint: broken, P256dh: p256dh, Auth: auth, Topics: []string{"items"}},
		{Endpoint: "https://web.push.apple.com/quiet", P256dh: p256dh, Auth: auth, Topics: []string{"fixes"}},
		{Endpoint: "https://evil.example/x", P256dh: p256dh, Auth: auth, Topics: []string{"items"}},
	}
	res := s.Send(context.Background(), entries, func(e Entry) (Message, bool) {
		return (Changes{NewItems: []string{"a", "b"}}).Message(e.Topics)
	})
	if res.Sent != 1 || res.Skipped != 1 || res.Failed != 2 || len(res.Gone) != 1 || res.Gone[0] != gone {
		t.Fatalf("%+v", res)
	}
	for _, req := range fake.requests {
		if req.URL.Host == "evil.example" {
			t.Error("a request went to a host that is not a push service")
		}
	}
}

func TestAnnounceRecordsTheFirstReleaseThenSendsOnlyForChanges(t *testing.T) {
	dir := t.TempDir()
	store, _ := OpenStore(dir, 10)
	_, p256dh, auth := newSubscriber(t)
	gone := "https://fcm.googleapis.com/fcm/send/gone"
	for _, e := range []Entry{
		{Endpoint: fcm, P256dh: p256dh, Auth: auth, Topics: []string{"items"}},
		{Endpoint: gone, P256dh: p256dh, Auth: auth, Topics: []string{"items", "fixes"}},
		{Endpoint: "https://web.push.apple.com/stages", P256dh: p256dh, Auth: auth, Topics: []string{"stages"}},
	} {
		if err := store.Put(e, ""); err != nil {
			t.Fatal(err)
		}
	}
	fake := &fakePush{status: map[string]int{gone: http.StatusGone}}
	s := sender(t, fake)
	var logs []string
	logf := func(format string, args ...any) { logs = append(logs, format) }

	if err := Announce(context.Background(), writeRoot(t, base), dir, store, s, logf); err != nil {
		t.Fatal(err)
	}
	if len(fake.requests) != 0 {
		t.Fatalf("the first release sent %d messages", len(fake.requests))
	}
	if _, err := os.Stat(filepath.Join(dir, "release.json")); err != nil {
		t.Fatal("the first release was not recorded")
	}

	renamed := base
	renamed.version = "pigeon"
	if err := Announce(context.Background(), writeRoot(t, renamed), dir, store, s, logf); err != nil {
		t.Fatal(err)
	}
	if len(fake.requests) != 0 {
		t.Fatalf("an unchanged release sent %d messages", len(fake.requests))
	}

	next := renamed
	next.items = strings.Replace(base.items, `]
	]}`, `],
		[10003,"Rose Gown",1,1,3,5,7,8,"S","A","A","A","A",4,"Metallic Crisis"]
	]}`, 1)
	if err := Announce(context.Background(), writeRoot(t, next), dir, store, s, logf); err != nil {
		t.Fatal(err)
	}
	if len(fake.requests) != 2 {
		t.Fatalf("new items sent %d messages, want 2 (the stages-only subscriber is not interested)", len(fake.requests))
	}
	if _, ok := store.Get(gone); ok {
		t.Error("the ended subscription was kept")
	}
	if store.Len() != 2 {
		t.Errorf("store holds %d", store.Len())
	}
	saved, err := LoadRelease(filepath.Join(dir, "release.json"))
	if err != nil || saved.Version != "pigeon" || len(saved.Items) != 3 {
		t.Fatalf("recorded release %+v %v", saved, err)
	}

	if err := Announce(context.Background(), writeRoot(t, next), dir, store, s, logf); err != nil {
		t.Fatal(err)
	}
	if len(fake.requests) != 2 {
		t.Errorf("a restart with the same data sent again: %d", len(fake.requests))
	}
}

func TestConfirmSendsAShortLivedHelloAndReportsRefusals(t *testing.T) {
	_, p256dh, auth := newSubscriber(t)
	dead := "https://fcm.googleapis.com/fcm/send/dead"
	fake := &fakePush{status: map[string]int{dead: http.StatusNotFound}}
	s := sender(t, fake)
	if err := s.Confirm(context.Background(), Entry{Endpoint: fcm, P256dh: p256dh, Auth: auth}); err != nil {
		t.Fatal(err)
	}
	req := fake.requests[0]
	if req.Header.Get("TTL") != "60" || req.Header.Get("Topic") != "nikkibase-hello" {
		t.Errorf("headers %v", req.Header)
	}
	if err := s.Confirm(context.Background(), Entry{Endpoint: dead, P256dh: p256dh, Auth: auth}); err == nil {
		t.Error("a 404 from the push service counted as confirmed")
	}
	if err := s.Confirm(context.Background(), Entry{Endpoint: "https://x1.notify.windows.com.evil.example/w", P256dh: p256dh, Auth: auth}); err == nil {
		t.Error("a host that is not a push service was confirmed")
	}
	if len(fake.requests) != 2 {
		t.Errorf("%d requests went out, want 2", len(fake.requests))
	}
}
