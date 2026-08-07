package codexhome

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var privateEntries = map[string]bool{
	"auth.json":         true,
	"models_cache.json": true,
}

var shadowLocalEntries = map[string]bool{
	"log":      true,
	"memories": true,
	"tmp":      true,
}

// Account represents a T3-compatible Codex home. The main account uses
// ~/.codex; named accounts use ~/.codex-<slug>.
type Account struct {
	Name string
	Home string
	Main bool
}

func MainHome(home string) string { return filepath.Join(home, ".codex") }

func HomeFor(home, name string) (string, error) {
	if name == "" || name == "main" {
		return MainHome(home), nil
	}
	if !validName(name) {
		return "", fmt.Errorf("invalid account name %q; use lowercase letters, digits, and hyphens", name)
	}
	return filepath.Join(home, ".codex-"+name), nil
}

func Discover(home string) ([]Account, error) {
	entries, err := os.ReadDir(home)
	if err != nil {
		return nil, err
	}
	accounts := make([]Account, 0)
	for _, entry := range entries {
		name := entry.Name()
		main := name == ".codex"
		if !main && !strings.HasPrefix(name, ".codex-") {
			continue
		}
		path := filepath.Join(home, name)
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		if !hasPrivateAuth(path) {
			continue
		}
		accountName := "main"
		if !main {
			accountName = strings.TrimPrefix(name, ".codex-")
		}
		accounts = append(accounts, Account{Name: accountName, Home: path, Main: main})
	}
	sort.Slice(accounts, func(i, j int) bool {
		if accounts[i].Main != accounts[j].Main {
			return accounts[i].Main
		}
		return accounts[i].Name < accounts[j].Name
	})
	return accounts, nil
}

func Create(home, name string) (Account, error) {
	path, err := HomeFor(home, name)
	if err != nil {
		return Account{}, err
	}
	if name == "" || name == "main" {
		return Account{}, errors.New("the main Codex home already exists; log in with `dotfiles auth codex login main`")
	}
	if _, err := os.Lstat(path); err == nil {
		return Account{}, fmt.Errorf("account %q already exists", name)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Account{}, err
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		return Account{}, err
	}
	if err := SyncShared(home, path); err != nil {
		return Account{}, err
	}
	return Account{Name: name, Home: path}, nil
}

// SyncShared mirrors T3 Code's shadow-home policy: auth and model cache stay
// private, logs/memories/tmp stay local, and every other existing root entry is
// shared from the main home.
func SyncShared(home, accountHome string) error {
	main := MainHome(home)
	entries, err := os.ReadDir(main)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if privateEntries[name] || shadowLocalEntries[name] {
			continue
		}
		destination := filepath.Join(accountHome, name)
		if _, err := os.Lstat(destination); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.Symlink(filepath.Join(main, name), destination); err != nil {
			return fmt.Errorf("share %s: %w", name, err)
		}
	}
	return nil
}

func EnsurePrivateAuth(account Account) error {
	authPath := filepath.Join(account.Home, "auth.json")
	info, err := os.Lstat(authPath)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s was not created; rerun login and complete device authentication", authPath)
	}
	if err != nil {
		return err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%s must be a real private file, not a symlink", authPath)
	}
	if err := os.Chmod(authPath, 0o600); err != nil {
		return err
	}
	cachePath := filepath.Join(account.Home, "models_cache.json")
	if cache, err := os.Lstat(cachePath); err == nil && cache.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%s must be private, not a symlink", cachePath)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func hasPrivateAuth(home string) bool {
	info, err := os.Lstat(filepath.Join(home, "auth.json"))
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	cache, err := os.Lstat(filepath.Join(home, "models_cache.json"))
	return errors.Is(err, os.ErrNotExist) || (err == nil && cache.Mode()&fs.ModeSymlink == 0)
}

func validName(name string) bool {
	if name == "" || len(name) > 48 {
		return false
	}
	for index, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || (char == '-' && index > 0 && index < len(name)-1) {
			continue
		}
		return false
	}
	return true
}
