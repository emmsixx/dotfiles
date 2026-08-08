package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkRefusesWhenRepoAndHomeAreTheSameDirectory(t *testing.T) {
	same := t.TempDir()
	if err := os.WriteFile(filepath.Join(same, ".zshrc"), []byte("managed"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := Link(same, same, func(name string, args ...string) error {
		t.Fatalf("run should not be called, got %s %v", name, args)
		return nil
	})
	if err == nil {
		t.Fatal("expected Link to refuse when repo and home are the same directory")
	}
	if _, statErr := os.Stat(filepath.Join(same, ".zshrc")); statErr != nil {
		t.Fatalf("repo file should be untouched: %v", statErr)
	}
}

func TestUnfoldConfigPreservesUnmanagedRuntimeData(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".config", "managed"), 0o700); err != nil {
		t.Fatal(err)
	}
	oldConfig := filepath.Join(t.TempDir(), "config")
	if err := os.MkdirAll(filepath.Join(oldConfig, "managed"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(oldConfig, "astro"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldConfig, "astro", "config.json"), []byte("runtime"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(oldConfig, filepath.Join(home, ".config")); err != nil {
		t.Fatal(err)
	}
	if err := unfoldConfig(repo, home); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(filepath.Join(home, ".config"))
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("~/.config should be a real directory: info=%v err=%v", info, err)
	}
	contents, err := os.ReadFile(filepath.Join(home, ".config", "astro", "config.json"))
	if err != nil || string(contents) != "runtime" {
		t.Fatalf("runtime config was not preserved: %q, %v", contents, err)
	}
	if _, err := os.Stat(filepath.Join(oldConfig, "astro", "config.json")); err != nil {
		t.Fatalf("old checkout should remain untouched: %v", err)
	}
}

func TestBackUpStowConflictsMovesOnlyManagedFiles(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, ".zshrc"), []byte("managed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("custom"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".config", "runtime"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "runtime", "keep"), []byte("runtime"), 0o600); err != nil {
		t.Fatal(err)
	}
	restore, err := backUpStowConflicts(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(home, ".zshrc")); !os.IsNotExist(err) {
		t.Fatalf("managed conflict should be moved, err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "runtime", "keep")); err != nil {
		t.Fatalf("unmanaged runtime state should stay in place: %v", err)
	}
	restore()
	contents, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil || string(contents) != "custom" {
		t.Fatalf("restore failed: %q, %v", contents, err)
	}
}

func TestRemoveAbsoluteManagedLinksRemovesOnlyCurrentCheckoutLinks(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	source := filepath.Join(repo, ".config", "paneru")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(home, ".config", "paneru")
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, target); err != nil {
		t.Fatal(err)
	}
	unrelated := filepath.Join(home, ".config", "unrelated")
	if err := os.Symlink("/tmp/elsewhere", unrelated); err != nil {
		t.Fatal(err)
	}
	if err := removeAbsoluteManagedLinks(repo, home); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("managed absolute link should be removed, err = %v", err)
	}
	if info, err := os.Lstat(unrelated); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("unrelated link should be preserved: %v", err)
	}
}

func TestLinkGeneratedSkillsUsesPiGlobalDirectory(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	source := filepath.Join(repo, ".pi", "skills", "generated")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := linkGeneratedSkills(repo, home); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(home, ".pi", "agent", "skills", "generated")
	link, err := os.Readlink(target)
	if err != nil || link != source {
		t.Fatalf("generated Pi skill link = %q, %v; want %q", link, err, source)
	}
	legacyTarget := filepath.Join(home, ".pi", "skills", "generated")
	if _, err := os.Lstat(legacyTarget); !os.IsNotExist(err) {
		t.Fatalf("legacy Pi skill target should not be created, err = %v", err)
	}
}

func TestLinkGeneratedSkillsPreservesUnrelatedTargets(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	source := filepath.Join(repo, ".claude", "skills", "generated")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := linkGeneratedSkills(repo, home); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(home, ".claude", "skills", "generated")
	link, err := os.Readlink(target)
	if err != nil || link != source {
		t.Fatalf("generated skill link = %q, %v; want %q", link, err, source)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("custom"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := linkGeneratedSkills(repo, home); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(target); err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("custom generated-skill target should be preserved: %v", err)
	}
}
