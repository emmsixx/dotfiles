package gitidentity

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCreatePreservesIdentityAndMissingProfilesFailClosed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, ".gitconfig"))
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	for _, key := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(key, "")
	}
	repo := filepath.Join(home, "repos", "Work", "demo")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "-c", "init.templateDir=", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %s %v", output, err)
	}
	for _, name := range []string{".gitconfig", ".config/git/work.gitconfig"} {
		data, err := os.ReadFile(filepath.Join("..", "..", name))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(home, name)
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	output, err := exec.Command("git", "-C", repo, "config", "--get", "user.email").Output()
	if err != nil || string(output) != "\n" {
		t.Fatalf("missing Work identity inherited fallback: %q %v", output, err)
	}
	profile := Profiles[1]
	if Configured(home, profile) {
		t.Fatal("missing identity configured")
	}
	if err := Create(home, profile, `Name "Quoted"`, "test@example.invalid"); err != nil {
		t.Fatal(err)
	}
	if !Configured(home, profile) {
		t.Fatal("created identity not configured")
	}
	output, err = exec.Command("git", "-C", repo, "config", "--get", "user.email").Output()
	if err != nil || string(output) != "test@example.invalid\n" {
		t.Fatalf("local identity not loaded: %q %v", output, err)
	}
	if err := Create(home, profile, "Other", "other@example.invalid"); err == nil {
		t.Fatal("existing identity overwritten")
	}
}
