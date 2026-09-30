package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielradosa/nikkibase/push"
)

func TestWritesAKeyPairOnlyTheOwnerCanRead(t *testing.T) {
	out := filepath.Join(t.TempDir(), "secrets", "vapid.env")
	var log bytes.Buffer
	if err := run(out, &log); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("file mode = %o, want 600", mode)
	}
	dir, err := os.Stat(filepath.Dir(out))
	if err != nil {
		t.Fatal(err)
	}
	if mode := dir.Mode().Perm(); mode != 0o700 {
		t.Errorf("directory mode = %o, want 700", mode)
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		k, v, _ := strings.Cut(line, "=")
		env[k] = v
	}
	vapid, err := push.ParseVAPID(env["VAPID_PRIVATE_KEY"], "mailto:test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if vapid.Public != env["VAPID_PUBLIC_KEY"] {
		t.Error("the public key in the file does not belong to the private key")
	}

	if strings.Contains(log.String(), env["VAPID_PRIVATE_KEY"]) {
		t.Error("the private key was printed")
	}
	if !strings.Contains(log.String(), env["VAPID_PUBLIC_KEY"]) {
		t.Error("the public key was not printed")
	}
}

func TestNeverReplacesAnExistingKey(t *testing.T) {
	out := filepath.Join(t.TempDir(), "vapid.env")
	if err := os.WriteFile(out, []byte("VAPID_PRIVATE_KEY=live\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var log bytes.Buffer
	if err := run(out, &log); err == nil {
		t.Fatal("want an error when the file already exists")
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "VAPID_PRIVATE_KEY=live\n" {
		t.Error("the existing key was changed")
	}
	if log.Len() != 0 {
		t.Errorf("printed %q", log.String())
	}
}
