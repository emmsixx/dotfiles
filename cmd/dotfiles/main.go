package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/emmsixx/dotfiles/internal/app"
)

var version = "dev"

func main() {
	a := app.New(version, os.Stdout, os.Stderr)

	// Named account launchers are symlinks to this binary. Keeping the dispatch
	// here makes them work without generating a separate script per account.
	if name := strings.TrimPrefix(filepath.Base(os.Args[0]), "codex-"); name != filepath.Base(os.Args[0]) {
		if err := a.RunCodex(name, os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "dotfiles:", err)
			os.Exit(1)
		}
		return
	}

	if err := a.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "dotfiles:", err)
		os.Exit(1)
	}
}
