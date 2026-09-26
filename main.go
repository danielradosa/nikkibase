package main

import (
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var hidden = os.Getenv("NOINDEX") != ""

func main() {
	root := envOr("WEB_ROOT", "web/dist")
	port := envOr("PORT", "8080")

	if _, err := os.Stat(filepath.Join(root, "index.html")); err != nil {
		log.Fatalf("serve: no index.html under %s -- run the web build first: %v", root, err)
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler(root),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("serve: %s on :%s", root, port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func handler(root string) http.Handler {
	if err := mime.AddExtensionType(".wasm", "application/wasm"); err != nil {
		log.Printf("serve: registering the wasm media type: %v", err)
	}
	files := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		clean := filepath.Clean("/" + r.URL.Path)

		security(w)
		cache(w, clean, r.URL.Query().Has("v"))
		w.Header().Add("Vary", "Accept-Encoding")

		if filepath.Ext(clean) != "" && serveEncoded(w, r, root, clean) {
			return
		}
		if filepath.Ext(clean) == "" && clean != "/" {
			if _, err := os.Stat(filepath.Join(root, clean)); err != nil {
				r = r.Clone(r.Context())
				r.URL.Path = "/"
			}
		}
		files.ServeHTTP(w, r)
	})
}

var encodings = []struct{ token, suffix string }{
	{"br", ".br"},
	{"gzip", ".gz"},
}

func serveEncoded(w http.ResponseWriter, r *http.Request, root, clean string) bool {
	accept := strings.Join(r.Header.Values("Accept-Encoding"), ",")
	for _, enc := range encodings {
		if !accepts(accept, enc.token) {
			continue
		}
		f, err := os.Open(filepath.Join(root, clean+enc.suffix))
		if err != nil {
			continue
		}
		info, err := f.Stat()
		if err != nil || info.IsDir() {
			f.Close()
			continue
		}
		ctype := mime.TypeByExtension(filepath.Ext(clean))
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		w.Header().Set("Content-Type", ctype)
		w.Header().Set("Content-Encoding", enc.token)
		http.ServeContent(w, r, clean, info.ModTime(), f)
		f.Close()
		return true
	}
	return false
}

func accepts(header, coding string) bool {
	wildcard := false
	for _, part := range strings.Split(header, ",") {
		name, params, _ := strings.Cut(part, ";")
		name = strings.TrimSpace(name)
		switch {
		case strings.EqualFold(name, coding):
			return positive(params)
		case name == "*":
			wildcard = positive(params)
		}
	}
	return wildcard
}

func positive(params string) bool {
	for _, param := range strings.Split(params, ";") {
		key, value, ok := strings.Cut(param, "=")
		if ok && strings.EqualFold(strings.TrimSpace(key), "q") {
			q, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			return err == nil && q > 0
		}
	}
	return true
}

func cache(w http.ResponseWriter, path string, versioned bool) {
	switch {
	case path == "/data/index.json", path == "/index.html", filepath.Ext(path) == "":
		w.Header().Set("Cache-Control", "no-cache")
	case strings.HasPrefix(path, "/data/"), strings.HasPrefix(path, "/assets/"), versioned:
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	default:
		w.Header().Set("Cache-Control", "public, max-age=300")
	}
}

func security(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Security-Policy", strings.Join([]string{
		"default-src 'self'",
		"script-src 'self' 'wasm-unsafe-eval'",
		"style-src 'self' 'unsafe-inline'",
		"font-src 'self'",
		"img-src 'self' data:",
		"connect-src 'self'",
		"worker-src 'self' blob:",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'none'",
	}, "; "))
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Cross-Origin-Opener-Policy", "same-origin")
	h.Set("Permissions-Policy", "geolocation=(), camera=(), microphone=(), interest-cohort=()")
	if hidden {
		h.Set("X-Robots-Tag", "noindex, nofollow")
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
