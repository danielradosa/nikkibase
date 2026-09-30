package push

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"
)

func mustDecode(t *testing.T, s string) []byte {
	t.Helper()
	b, err := decode(strings.ReplaceAll(s, " ", ""))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestEncryptMatchesRFC8291Example(t *testing.T) {
	asPrivate, err := ecdh.P256().NewPrivateKey(mustDecode(t, "yfWPiYE-n46HLnH0KqZOF1fJJU3MYrct3AELtAQ-oRw"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := encrypt(
		mustDecode(t, "V2hlbiBJIGdyb3cgdXAsIEkgd2FudCB0byBiZSBhIHdhdGVybWVsb24"),
		mustDecode(t, "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcx aOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4"),
		mustDecode(t, "BTBZMqHH6r4Tts7J_aSIgg"),
		asPrivate,
		mustDecode(t, "DGv6ra1nlYgDCS1FRnbzlw"),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "DGv6ra1nlYgDCS1FRnbzlwAAEABBBP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27ml" +
		"mlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A_yl95bQpu6cVPT" +
		"pK4Mqgkf1CXztLVBSt2Ks3oZwbuwXPXLWyouBWLVWGNWQexSgSxsj_Qulcy4a-fN"
	if b64.EncodeToString(got) != want {
		t.Fatalf("encrypted message\n got %s\nwant %s", b64.EncodeToString(got), want)
	}
}

func TestEncryptRejectsBadKeysAndLargePayloads(t *testing.T) {
	as, _ := ecdh.P256().NewPrivateKey(mustDecode(t, "yfWPiYE-n46HLnH0KqZOF1fJJU3MYrct3AELtAQ-oRw"))
	ua := mustDecode(t, "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4")
	auth := mustDecode(t, "BTBZMqHH6r4Tts7J_aSIgg")
	salt := mustDecode(t, "DGv6ra1nlYgDCS1FRnbzlw")
	if _, err := encrypt([]byte("x"), ua[:64], auth, as, salt); err == nil {
		t.Error("a short p256dh key was accepted")
	}
	if _, err := encrypt([]byte("x"), ua, auth[:15], as, salt); err == nil {
		t.Error("a short auth secret was accepted")
	}
	if got, err := encrypt(make([]byte, MaxPayload), ua, auth, as, salt); err != nil || len(got) != recordSize {
		t.Errorf("the largest payload RFC 8291 allows: %d bytes, %v", len(got), err)
	}
	if _, err := encrypt(make([]byte, MaxPayload+1), ua, auth, as, salt); err == nil {
		t.Error("a payload one byte over the RFC 8291 limit was accepted")
	}
	if _, err := Encrypt([]byte("x"), "not base64!", "BTBZMqHH6r4Tts7J_aSIgg"); err == nil {
		t.Error("a p256dh that is not base64url was accepted")
	}
}

func TestEncryptUsesAFreshKeyAndSaltEachTime(t *testing.T) {
	a, err := Encrypt([]byte("hello"), "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4", "BTBZMqHH6r4Tts7J_aSIgg")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Encrypt([]byte("hello"), "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4", "BTBZMqHH6r4Tts7J_aSIgg")
	if string(a[:16]) == string(b[:16]) || string(a[21:86]) == string(b[21:86]) {
		t.Error("two messages share a salt or a server key")
	}
	if len(a) != 86+len("hello")+1+16 {
		t.Errorf("length %d", len(a))
	}
}

func verifyJWT(token string, public []byte) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("jwt: want 3 parts")
	}
	sig, err := decode(parts[2])
	if err != nil || len(sig) != 64 {
		return nil, errors.New("jwt: bad signature encoding")
	}
	key, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), public)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.Verify(key, digest[:], new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])) {
		return nil, errors.New("jwt: signature does not verify")
	}
	body, err := decode(parts[1])
	if err != nil {
		return nil, err
	}
	var claims map[string]any
	return claims, json.Unmarshal(body, &claims)
}

func TestVAPIDAuthorizationIsASignedTokenForThePushOrigin(t *testing.T) {
	private, public, err := NewVAPIDKey()
	if err != nil {
		t.Fatal(err)
	}
	v, err := ParseVAPID(private, "mailto:nikkibaseproject@gmail.com")
	if err != nil {
		t.Fatal(err)
	}
	if v.Public != public {
		t.Fatalf("public key %s, want %s", v.Public, public)
	}
	now := time.Unix(1_800_000_000, 0)
	header, err := v.Authorization("https://fcm.googleapis.com/fcm/send/abc:def", now)
	if err != nil {
		t.Fatal(err)
	}
	token, key, ok := strings.Cut(strings.TrimPrefix(header, "vapid t="), ", k=")
	if !strings.HasPrefix(header, "vapid t=") || !ok || key != public {
		t.Fatalf("header %q", header)
	}
	claims, err := verifyJWT(token, mustDecode(t, public))
	if err != nil {
		t.Fatal(err)
	}
	if claims["aud"] != "https://fcm.googleapis.com" || claims["sub"] != "mailto:nikkibaseproject@gmail.com" {
		t.Errorf("claims %v", claims)
	}
	if exp := int64(claims["exp"].(float64)); exp <= now.Unix() || exp > now.Add(24*time.Hour).Unix() {
		t.Errorf("exp %d is not within 24 hours of %d", exp, now.Unix())
	}
	headJSON, _ := decode(strings.Split(token, ".")[0])
	if string(headJSON) != `{"alg":"ES256","typ":"JWT"}` {
		t.Errorf("jwt header %s", headJSON)
	}
}

func TestParseVAPIDRejectsBadInput(t *testing.T) {
	private, _, _ := NewVAPIDKey()
	if _, err := ParseVAPID(private, "nikkibaseproject@gmail.com"); err == nil {
		t.Error("a subject without mailto: was accepted")
	}
	if _, err := ParseVAPID("short", "mailto:a@b.c"); err == nil {
		t.Error("a short key was accepted")
	}
}
