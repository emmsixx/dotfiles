package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCodexQuietChildSuppressesStderrOnSuccess(t *testing.T) {
	home := prepareCodexCommandTest(t)
	var out, errOut bytes.Buffer
	a := New("test", &out, &errOut)
	a.Home = home

	if err := a.Run([]string{"codex", "--account", "main", "--quiet-child", "--", "success"}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "child stdout\n" {
		t.Fatalf("stdout = %q, want child stdout", got)
	}
	if got := errOut.String(); got != "" {
		t.Fatalf("stderr = %q, want no output", got)
	}
}

func TestCodexQuietChildReplaysStderrOnFailure(t *testing.T) {
	home := prepareCodexCommandTest(t)
	var out, errOut bytes.Buffer
	a := New("test", &out, &errOut)
	a.Home = home

	if err := a.Run([]string{"codex", "--account", "main", "--quiet-child", "--", "fail"}); err == nil {
		t.Fatal("expected child failure")
	}
	if got := errOut.String(); got != "child stderr\n" {
		t.Fatalf("stderr = %q, want child failure output", got)
	}
}

func prepareCodexCommandTest(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	codexHome := filepath.Join(home, ".codex")
	if err := os.Mkdir(codexHome, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codexHome, "auth.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	bin := filepath.Join(t.TempDir(), "codex")
	script := `#!/bin/sh
printf 'child stdout\n'
printf 'child stderr\n' >&2
for arg do
    if [ "$arg" = "fail" ]; then
        exit 7
    fi
done
`
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	return home
}

func TestCodexRefusesSharedCredentialsBeforeLaunching(t *testing.T) {
	for _, bad := range []string{"auth.json", "models_cache.json", "home"} {
		t.Run(bad, func(t *testing.T) {
			home := prepareCodexCommandTest(t)
			main := filepath.Join(home, ".codex")
			work := filepath.Join(home, ".codex-work")
			if bad == "home" {
				if err := os.Symlink(main, work); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Mkdir(work, 0o700); err != nil {
					t.Fatal(err)
				}
				if bad != "auth.json" {
					if err := os.WriteFile(filepath.Join(work, "auth.json"), []byte("{}"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.Symlink(filepath.Join(main, "auth.json"), filepath.Join(work, bad)); err != nil {
					t.Fatal(err)
				}
			}
			for _, args := range [][]string{{"auth", "codex", "login", "work"}, {"codex", "--account", "work", "--", "success"}} {
				var out, errOut bytes.Buffer
				a := New("test", &out, &errOut)
				a.Home = home
				if err := a.Run(args); err == nil {
					t.Fatal("invalid account accepted")
				}
				if out.Len() != 0 {
					t.Fatalf("child launched before validation: %s", out.String())
				}
				contents, _ := os.ReadFile(filepath.Join(main, "auth.json"))
				if string(contents) != "{}" {
					t.Fatal("main auth changed")
				}
			}
		})
	}
}
