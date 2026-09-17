package setup

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, root, relative, contents string) string {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func trackedRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	for name, contents := range files {
		writeFixture(t, repo, name, contents)
	}
	writeFixture(t, repo, ".stow-local-ignore", "^/\\.stow-local-ignore$\n")
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}} {
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "init.templateDir="}, args...)...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git: %s: %v", output, err)
		}
	}
	return repo
}

func realStow(t *testing.T) func(string, ...string) error {
	t.Helper()
	if _, err := exec.LookPath("stow"); err != nil {
		t.Skip("stow is required for linking integration tests")
	}
	return func(name string, args ...string) error {
		output, err := exec.Command(name, args...).CombinedOutput()
		if err != nil {
			return errors.New(string(output) + err.Error())
		}
		return nil
	}
}

func readFixture(t *testing.T, path, want string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != want {
		t.Fatalf("%s = %q, %v; want %q", path, contents, err, want)
	}
}

func TestLinkRefusesRepoHomeAliases(t *testing.T) {
	repo := trackedRepo(t, map[string]string{".zshrc": "managed"})
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(repo, alias); err != nil {
		t.Fatal(err)
	}
	for _, home := range []string{repo, alias} {
		if err := Link(repo, home, func(string, ...string) error { t.Fatal("runner called"); return nil }); err == nil {
			t.Fatal("accepted aliased home")
		}
	}
	readFixture(t, filepath.Join(repo, ".zshrc"), "managed")
}

func TestLinkUnfoldsDirectoriesWithoutMovingSourceOrLosingRuntime(t *testing.T) {
	for _, root := range []string{".config", ".claude", ".pi"} {
		for _, absolute := range []bool{false, true} {
			t.Run(root+map[bool]string{false: "relative", true: "absolute"}[absolute], func(t *testing.T) {
				leaf := map[string]string{".config": "app/config", ".claude": "settings.json", ".pi": "agent/AGENTS.md"}[root]
				repo := trackedRepo(t, map[string]string{filepath.Join(root, leaf): "managed"})
				writeFixture(t, repo, filepath.Join(root, "runtime/auth.json"), "private runtime")
				home := t.TempDir()
				source := filepath.Join(repo, root)
				target := filepath.Join(home, root)
				link := source
				if !absolute {
					var err error
					link, err = filepath.Rel(home, source)
					if err != nil {
						t.Fatal(err)
					}
				}
				if err := os.Symlink(link, target); err != nil {
					t.Fatal(err)
				}
				for i := 0; i < 2; i++ {
					if err := Link(repo, home, realStow(t)); err != nil {
						t.Fatal(err)
					}
				}
				readFixture(t, filepath.Join(repo, root, leaf), "managed")
				readFixture(t, filepath.Join(home, root, leaf), "managed")
				readFixture(t, filepath.Join(home, root, "runtime/auth.json"), "private runtime")
				info, err := os.Lstat(filepath.Join(home, root, "runtime/auth.json"))
				if err != nil || !info.Mode().IsRegular() {
					t.Fatal("runtime was linked instead of copied")
				}
				info, err = os.Lstat(target)
				if err != nil || !info.IsDir() {
					t.Fatal("root is not a real directory")
				}
			})
		}
	}
}

func TestLinkExcludesUntrackedAndPrivateFiles(t *testing.T) {
	repo := trackedRepo(t, map[string]string{".zshrc": "managed", ".config/gh/hosts.yml": "tracked private", ".pi/agent/extensions/node_modules/a/index.js": "dependency"})
	for _, path := range []string{".config/gh/hosts.yml", ".pi/agent/auth.json", ".secrets", ".pi/agent/extensions/summaries/config.private.json", ".config/new-runtime/state"} {
		writeFixture(t, repo, path, "stale")
	}
	home := t.TempDir()
	writeFixture(t, home, ".config/gh/hosts.yml", "active")
	if err := Link(repo, home, realStow(t)); err != nil {
		t.Fatal(err)
	}
	readFixture(t, filepath.Join(home, ".config/gh/hosts.yml"), "active")
	for _, path := range []string{".secrets", ".pi/agent/auth.json", ".pi/agent/extensions/node_modules/a/index.js", ".config/new-runtime/state"} {
		if _, err := os.Lstat(filepath.Join(home, path)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("private/untracked path linked: %s", path)
		}
	}
}

func TestLinkPreservesLocalGeneratedPalettes(t *testing.T) {
	palettes := []string{
		".config/nvim/colors/dms.lua",
		".config/nvim/lua/plugins/dankcolors.lua",
		".config/ghostty/themes/dankcolors",
		".config/niri/colors.kdl",
	}
	files := map[string]string{".zshrc": "managed"}
	for _, path := range palettes {
		files[path] = "another machine's palette"
	}
	repo := trackedRepo(t, files)
	for _, existing := range []bool{false, true} {
		home := t.TempDir()
		if existing {
			for _, path := range palettes {
				writeFixture(t, home, path, "local palette")
			}
		}
		if err := Link(repo, home, realStow(t)); err != nil {
			t.Fatal(err)
		}
		for _, path := range palettes {
			target := filepath.Join(home, path)
			if existing {
				readFixture(t, target, "local palette")
				info, err := os.Lstat(target)
				if err != nil || !info.Mode().IsRegular() {
					t.Fatalf("local palette was replaced: %s", target)
				}
			} else if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("generated palette was installed: %s", target)
			}
		}
	}
}

func TestLinkRefusesUnrelatedAncestorBeforeMutating(t *testing.T) {
	repo := trackedRepo(t, map[string]string{".zshrc": "managed", ".claude/settings.json": "managed"})
	home := t.TempDir()
	elsewhere := t.TempDir()
	writeFixture(t, home, ".zshrc", "custom")
	writeFixture(t, elsewhere, "settings.json", "external")
	if err := os.Symlink(elsewhere, filepath.Join(home, ".claude")); err != nil {
		t.Fatal(err)
	}
	if err := Link(repo, home, func(string, ...string) error { t.Fatal("stow called"); return nil }); err == nil {
		t.Fatal("unrelated directory accepted")
	}
	readFixture(t, filepath.Join(home, ".zshrc"), "custom")
	readFixture(t, filepath.Join(elsewhere, "settings.json"), "external")
}

func TestLinkRollsBackFoldedDirectoriesFilesAndPartialStow(t *testing.T) {
	repo := trackedRepo(t, map[string]string{".config/app/config": "managed", ".zshrc": "managed", "new-file": "new"})
	home := t.TempDir()
	writeFixture(t, home, ".zshrc", "custom")
	oldLink := filepath.Join(repo, ".config")
	if err := os.Symlink(oldLink, filepath.Join(home, ".config")); err != nil {
		t.Fatal(err)
	}
	err := Link(repo, home, func(string, ...string) error {
		if err := os.Symlink(filepath.Join(repo, "new-file"), filepath.Join(home, "new-file")); err != nil {
			t.Fatal(err)
		}
		return errors.New("injected stow failure")
	})
	if err == nil || !strings.Contains(err.Error(), "injected") {
		t.Fatal(err)
	}
	readFixture(t, filepath.Join(home, ".zshrc"), "custom")
	readFixture(t, filepath.Join(home, ".config/app/config"), "managed")
	link, err := os.Readlink(filepath.Join(home, ".config"))
	if err != nil || link != oldLink {
		t.Fatalf("original link not restored: %s %v", link, err)
	}
	if _, err := os.Lstat(filepath.Join(home, "new-file")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("partial stow link survived")
	}
}

func TestLinkGeneratedSkillsWithRelativeRepoAndExistingTargets(t *testing.T) {
	repo := trackedRepo(t, map[string]string{".zshrc": "managed"})
	writeFixture(t, repo, ".pi/skills/generated/SKILL.md", "skill")
	writeFixture(t, repo, ".claude/skills/custom/SKILL.md", "generated")
	home := t.TempDir()
	writeFixture(t, home, ".claude/skills/custom/SKILL.md", "custom")
	t.Chdir(filepath.Dir(repo))
	if err := Link(filepath.Base(repo), home, realStow(t)); err != nil {
		t.Fatal(err)
	}
	readFixture(t, filepath.Join(home, ".pi/agent/skills/generated/SKILL.md"), "skill")
	readFixture(t, filepath.Join(home, ".claude/skills/custom/SKILL.md"), "custom")
}
