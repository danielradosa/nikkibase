package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func root(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for path, body := range map[string]string{
		"index.html":                   "<!doctype html><title>shell</title>",
		"assets/index-abc123.js":       "console.log(1)",
		"assets/index-abc123.js.br":    "brotli js",
		"assets/index-abc123.js.gz":    "gzip js",
		"data/index.json":              `{"version":"2026-09-22"}`,
		"data/2026-09-22/it.json":      `{"items":[]}`,
		"data/2026-09-22/it.json.gz":   "gzip json",
		"assets/nikkibase-abc.wasm":    "\x00asm",
		"assets/nikkibase-abc.wasm.br": "brotli wasm",
		"keystream.bin":                "key",
	} {
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func get(t *testing.T, h http.Handler, path string) *http.Response {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w.Result()
}

func fetch(t *testing.T, h http.Handler, method, path, accept string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if accept != "" {
		req.Header.Set("Accept-Encoding", accept)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w.Result()
}

func TestPrecompressedSiblings(t *testing.T) {
	h := handler(root(t))
	for _, c := range []struct{ path, accept, encoding, body, ctype string }{
		{"/assets/index-abc123.js", "gzip, deflate, br, zstd", "br", "brotli js", "text/javascript"},
		{"/assets/index-abc123.js", "gzip", "gzip", "gzip js", "text/javascript"},
		{"/assets/index-abc123.js", "br;q=0, gzip;q=0.5", "gzip", "gzip js", "text/javascript"},
		{"/assets/index-abc123.js", "*", "br", "brotli js", "text/javascript"},
		{"/assets/index-abc123.js", "*, br;q=0", "gzip", "gzip js", "text/javascript"},
		{"/assets/index-abc123.js", "BR", "br", "brotli js", "text/javascript"},
		{"/assets/index-abc123.js", "", "", "console.log(1)", "text/javascript"},
		{"/assets/index-abc123.js", "identity", "", "console.log(1)", "text/javascript"},
		{"/assets/index-abc123.js", "*;q=0", "", "console.log(1)", "text/javascript"},
		{"/data/2026-09-22/it.json", "br, gzip", "gzip", "gzip json", "application/json"},
		{"/assets/nikkibase-abc.wasm", "br", "br", "brotli wasm", "application/wasm"},
		{"/data/index.json", "br, gzip", "", `{"version":"2026-09-22"}`, "application/json"},
	} {
		res := fetch(t, h, http.MethodGet, c.path, c.accept)
		body, _ := io.ReadAll(res.Body)
		name := c.path + " [" + c.accept + "]"
		if res.StatusCode != http.StatusOK {
			t.Errorf("%s: status %d", name, res.StatusCode)
		}
		if got := res.Header.Get("Content-Encoding"); got != c.encoding {
			t.Errorf("%s: Content-Encoding = %q, want %q", name, got, c.encoding)
		}
		if string(body) != c.body {
			t.Errorf("%s: body = %q, want %q", name, body, c.body)
		}
		if got := res.Header.Get("Content-Type"); !strings.HasPrefix(got, c.ctype) {
			t.Errorf("%s: Content-Type = %q, want %s", name, got, c.ctype)
		}
		if got := res.Header.Get("Content-Length"); c.encoding == "" && got != strconv.Itoa(len(c.body)) {
			t.Errorf("%s: Content-Length = %q, want %d", name, got, len(c.body))
		}
		if got := res.Header.Get("Vary"); got != "Accept-Encoding" {
			t.Errorf("%s: Vary = %q, want Accept-Encoding", name, got)
		}
	}
}

func TestPrecompressedKeepsHeadersAndHead(t *testing.T) {
	h := handler(root(t))
	res := fetch(t, h, http.MethodHead, "/assets/index-abc123.js", "br")
	body, _ := io.ReadAll(res.Body)
	if res.Header.Get("Content-Encoding") != "br" || len(body) != 0 {
		t.Errorf("HEAD: Content-Encoding %q with %d body bytes, want br and none", res.Header.Get("Content-Encoding"), len(body))
	}
	if !strings.Contains(res.Header.Get("Cache-Control"), "immutable") {
		t.Errorf("HEAD: Cache-Control = %q, want immutable", res.Header.Get("Cache-Control"))
	}
	if !strings.Contains(res.Header.Get("Content-Security-Policy"), "default-src 'self'") {
		t.Error("HEAD: the compressed response lost its CSP")
	}
	if res := fetch(t, h, http.MethodGet, "/assets/missing-deadbeef.js", "br, gzip"); res.StatusCode != http.StatusNotFound {
		t.Errorf("a missing asset returned %d with compression accepted, want 404", res.StatusCode)
	}
	if res := fetch(t, h, http.MethodGet, "/stages", "br"); res.StatusCode != http.StatusOK || res.Header.Get("Content-Encoding") != "" {
		t.Errorf("a route returned %d, Content-Encoding %q; want the plain shell", res.StatusCode, res.Header.Get("Content-Encoding"))
	}
}

func TestCacheLifetimes(t *testing.T) {
	h := handler(root(t))
	for _, c := range []struct{ path, want string }{
		{"/data/index.json", "no-cache"},
		{"/data/2026-09-22/it.json", "immutable"},
		{"/assets/index-abc123.js", "immutable"},
		{"/assets/nikkibase-abc.wasm", "immutable"},
		{"/", "no-cache"},
		{"/index.html", "no-cache"},
		{"/stages", "no-cache"},
		{"/keystream.bin?v=0123abcd", "immutable"},
		{"/keystream.bin", "max-age=300"},
	} {
		got := get(t, h, c.path).Header.Get("Cache-Control")
		if !strings.Contains(got, c.want) {
			t.Errorf("%s: Cache-Control = %q, want it to contain %q", c.path, got, c.want)
		}
	}
}

func TestWasmContentType(t *testing.T) {
	res := get(t, handler(root(t)), "/assets/nikkibase-abc.wasm")
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/wasm") {
		t.Errorf("Content-Type = %q, want application/wasm", ct)
	}
}

func TestRoutesFallBackButAssetsDoNot(t *testing.T) {
	h := handler(root(t))
	if res := get(t, h, "/stages"); res.StatusCode != http.StatusOK {
		t.Errorf("a route returned %d, want it to serve the shell", res.StatusCode)
	}
	if res := get(t, h, "/assets/missing-deadbeef.js"); res.StatusCode != http.StatusNotFound {
		t.Errorf("a missing asset returned %d, want 404", res.StatusCode)
	}
}

func TestSecurityHeaders(t *testing.T) {
	res := get(t, handler(root(t)), "/index.html")
	csp := res.Header.Get("Content-Security-Policy")
	for _, want := range []string{"connect-src 'self'", "frame-ancestors 'none'", "'wasm-unsafe-eval'", "font-src 'self'"} {
		if !strings.Contains(csp, want) {
			t.Errorf("CSP %q is missing %q", csp, want)
		}
	}
	for _, banned := range []string{"http:", "https:", "googleapis", "gstatic"} {
		if strings.Contains(csp, banned) {
			t.Errorf("CSP %q allows another origin (%q); everything must be served from 'self'", csp, banned)
		}
	}
	if res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing nosniff")
	}
}

func TestHiddenCopyAsksNotToBeIndexed(t *testing.T) {
	if got := get(t, handler(root(t)), "/").Header.Get("X-Robots-Tag"); got != "" {
		t.Errorf("the public site sends X-Robots-Tag %q", got)
	}
	hidden = true
	t.Cleanup(func() { hidden = false })
	h := handler(root(t))
	for _, path := range []string{"/", "/some/route", "/assets/index-abc123.js", "/data/index.json", "/missing.js"} {
		if got := get(t, h, path).Header.Get("X-Robots-Tag"); got != "noindex, nofollow" {
			t.Errorf("%s: X-Robots-Tag %q, want noindex, nofollow", path, got)
		}
	}
}

func TestOnlyReads(t *testing.T) {
	w := httptest.NewRecorder()
	handler(root(t)).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST returned %d, want 405", w.Code)
	}
}
