package app

import (
	"bytes"
	"strings"
	"testing"
)

func TestZshCompletionIncludesTopLevelAndCodexAccountCommands(t *testing.T) {
	var out bytes.Buffer
	a := New("test", &out, &bytes.Buffer{})
	if err := a.Run([]string{"completion", "zsh"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"#compdef dotfiles", "setup[run the setup wizard]", "--account=[Codex account]", "auth codex list"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("completion is missing %q", want)
		}
	}
}
