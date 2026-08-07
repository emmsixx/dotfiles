package codexhome

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateSharesOnlyT3SharedState(t *testing.T) {
	home := t.TempDir()
	main := MainHome(home)
	if err := os.Mkdir(main, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sessions", "skills", "auth.json", "models_cache.json", "log", "memories", "tmp"} {
		if err := os.MkdirAll(filepath.Join(main, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	account, err := Create(home, "work")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sessions", "skills"} {
		info, err := os.Lstat(filepath.Join(account.Home, name))
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s should be a shared symlink: %v", name, err)
		}
	}
	for _, name := range []string{"auth.json", "models_cache.json", "log", "memories", "tmp"} {
		if _, err := os.Lstat(filepath.Join(account.Home, name)); !os.IsNotExist(err) {
			t.Errorf("%s should remain private/local, err = %v", name, err)
		}
	}
}

func TestDiscoverRequiresRealAuthFile(t *testing.T) {
	home := t.TempDir()
	main := MainHome(home)
	if err := os.Mkdir(main, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(main, "auth.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(home, ".codex-bad")
	if err := os.Mkdir(bad, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(main, "auth.json"), filepath.Join(bad, "auth.json")); err != nil {
		t.Fatal(err)
	}
	accounts, err := Discover(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 1 || accounts[0].Name != "main" {
		t.Fatalf("Discover() = %#v, want main only", accounts)
	}
}

func TestDiscoverRejectsSharedModelCache(t *testing.T) {
	home := t.TempDir()
	account := filepath.Join(home, ".codex-work")
	if err := os.Mkdir(account, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(account, "auth.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/tmp/not-a-private-cache", filepath.Join(account, "models_cache.json")); err != nil {
		t.Fatal(err)
	}
	accounts, err := Discover(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 0 {
		t.Fatalf("Discover() = %#v, want no compatible account", accounts)
	}
}
