package secrets

import (
	"github.com/emmsixx/dotfiles/internal/ui"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdatePreservesCustomLinesAndPermissions(t *testing.T) {
	home := t.TempDir()
	path := Path(home)
	initial := "# personal notes\nexport GH_WORK_USER=old\nexport CUSTOM_VALUE=keep\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Update(home, map[string]string{"GH_WORK_USER": `new\\value\"quoted`, "FIRECRAWL_API_KEY": "secret"}); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(contents)
	for _, want := range []string{"# personal notes", "export CUSTOM_VALUE=keep", `export GH_WORK_USER='new\\value\"quoted'`, `export FIRECRAWL_API_KEY='secret'`} {
		if !strings.Contains(text, want) {
			t.Errorf("updated secrets missing %q:\n%s", want, text)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("mode = %o, want 600", got)
	}
	values, err := Read(home)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := values["GH_WORK_USER"], "new\\\\value\\\"quoted"; got != want {
		t.Errorf("Read() = %q, want %q", got, want)
	}
}

func TestEnsureUsesTemplate(t *testing.T) {
	home := t.TempDir()
	template := filepath.Join(t.TempDir(), "template")
	if err := os.WriteFile(template, []byte("export VALUE=\"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	created, err := Ensure(home, template)
	if err != nil || !created {
		t.Fatalf("Ensure() = (%v, %v), want (true, nil)", created, err)
	}
	created, err = Ensure(home, template)
	if err != nil || created {
		t.Fatalf("second Ensure() = (%v, %v), want (false, nil)", created, err)
	}
}

func TestLiteralSecretsRoundTripThroughShells(t *testing.T) {
	cases := []string{"", "spaces and 'quotes'", `dollar$HOME-$(printf BAD)-` + "`printf BAD`", "line one\nline two", `back\slash\n`, "quote\"", "last\n"}
	for _, shell := range []string{"sh", "zsh"} {
		if _, err := exec.LookPath(shell); err != nil {
			continue
		}
		for _, value := range cases {
			home := t.TempDir()
			if err := Update(home, map[string]string{"VALUE": value}); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(shell, "-c", `. "$1"; printf '%s' "$VALUE"`, "test", Path(home))
			actual, err := cmd.Output()
			if err != nil || string(actual) != value {
				t.Fatalf("%s round trip: %q want %q (%v)", shell, actual, value, err)
			}
			values, err := Read(home)
			if err != nil || values["VALUE"] != value {
				t.Fatalf("Read round trip: %q want %q (%v)", values["VALUE"], value, err)
			}
			if err := Update(home, map[string]string{"VALUE": "replacement"}); err != nil {
				t.Fatal(err)
			}
			values, err = Read(home)
			if err != nil || values["VALUE"] != "replacement" {
				t.Fatal("multiline replacement failed")
			}
		}
	}
}

func TestPromptPreservesUnrelatedShellExports(t *testing.T) {
	home := t.TempDir()
	custom := "export CUSTOM='abc' # keep comment\nexport DYNAMIC=\"$HOME\"\nexport MULTILINE='one\ntwo'\n"
	if err := os.WriteFile(Path(home), []byte(custom+"export GH_WORK_USER='old' # comment\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := prompt(home, []Field{{Key: "GH_WORK_USER"}}, func(_ string, inputs []ui.Input) error {
		if *inputs[0].Value != "old" {
			t.Fatalf("inline comment parsed incorrectly: %q", *inputs[0].Value)
		}
		*inputs[0].Value = "new"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(Path(home))
	if !strings.HasPrefix(string(data), custom) {
		t.Fatalf("unrelated content changed: %s", data)
	}
}

func TestEnsureRepairsPermissionsAndRejectsSymlinks(t *testing.T) {
	home := t.TempDir()
	path := Path(home)
	if err := os.WriteFile(path, []byte("private"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Ensure(home, "unused"); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Fatal("permissions not repaired")
	}
	other := t.TempDir()
	if err := os.Symlink(path, Path(other)); err != nil {
		t.Fatal(err)
	}
	if _, err := Ensure(other, "unused"); err == nil {
		t.Fatal("accepted symlink")
	}
}
