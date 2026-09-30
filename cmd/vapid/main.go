package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/danielradosa/nikkibase/push"
)

func main() {
	out := flag.String("out", ".ai/secrets/vapid.env", "the file to write the new key pair into; an existing file is never replaced")
	flag.Parse()

	if err := run(*out, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "vapid:", err)
		os.Exit(1)
	}
}

func run(out string, log io.Writer) error {
	private, public, err := push.NewVAPIDKey()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("%s already holds a key; changing the key breaks every subscription, so move it away first if you really mean to", out)
	}
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(f, "VAPID_PRIVATE_KEY=%s\nVAPID_PUBLIC_KEY=%s\n", private, public); err != nil {
		f.Close()
		os.Remove(out)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(out)
		return err
	}
	fmt.Fprintf(log, "vapid: wrote %s (0600)\nVAPID_PUBLIC_KEY=%s\n", out, public)
	return nil
}
