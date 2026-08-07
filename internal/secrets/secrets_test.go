package secrets

import (
	"os"
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
	for _, want := range []string{"# personal notes", "export CUSTOM_VALUE=keep", `export GH_WORK_USER="new\\\\value\\\"quoted"`, `export FIRECRAWL_API_KEY="secret"`} {
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
