package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/emmsixx/dotfiles/internal/codexhome"
	"github.com/emmsixx/dotfiles/internal/secrets"
	"github.com/emmsixx/dotfiles/internal/setup"
	"github.com/emmsixx/dotfiles/internal/ui"
)

type App struct {
	Version string
	Out     io.Writer
	Err     io.Writer
	Home    string
}

func New(version string, out, errOut io.Writer) *App {
	home, _ := os.UserHomeDir()
	return &App{Version: version, Out: out, Err: errOut, Home: home}
}

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		args = []string{"setup"}
	}
	switch args[0] {
	case "-h", "--help", "help":
		a.usage()
		return nil
	case "-v", "--version", "version":
		fmt.Fprintln(a.Out, a.Version)
		return nil
	case "setup":
		return a.runSetup(args[1:])
	case "secrets":
		return a.runSecrets(args[1:])
	case "auth":
		return a.runAuth(args[1:])
	case "codex":
		return a.runCodexCommand(args[1:])
	case "doctor":
		return a.runDoctor()
	default:
		return fmt.Errorf("unknown command %q; run `dotfiles help`", args[0])
	}
}

func (a *App) usage() {
	fmt.Fprint(a.Out, `dotfiles — interactive development-environment setup

Usage:
  dotfiles setup [--repo PATH] [--profile desktop|server] [--defaults] [--non-interactive]
                 [--components core,shell,...] [--t3-service nightly|latest]
  dotfiles secrets [status|edit]
  dotfiles auth [status|login]
  dotfiles auth codex [list|add NAME|login NAME|sync]
  dotfiles codex [--account NAME] -- [codex arguments]
  dotfiles doctor

The default command is setup. --defaults bypasses the component picker but
still creates secrets and offers authentication when running in a terminal.
codex-NAME launchers are generated for named accounts; cx opens an account
picker through dotfiles codex.
`)
}

func (a *App) runSetup(args []string) error {
	flags := flag.NewFlagSet("setup", flag.ContinueOnError)
	flags.SetOutput(a.Err)
	var o setup.Options
	var componentCSV string
	var server bool
	flags.StringVar(&o.Repo, "repo", os.Getenv("DOTFILES_DIR"), "dotfiles repository")
	flags.StringVar(&o.Profile, "profile", os.Getenv("DOTFILES_PROFILE"), "desktop or server")
	flags.BoolVar(&o.Defaults, "defaults", false, "use profile defaults without a picker")
	flags.BoolVar(&o.NonInteractive, "non-interactive", false, "never show secret/auth prompts")
	flags.BoolVar(&server, "server", false, "compatibility shorthand for --profile server")
	flags.StringVar(&componentCSV, "components", "", "comma-separated component IDs")
	flags.StringVar(&o.T3Channel, "t3-service", "", "install the T3 service from nightly or latest")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if componentCSV != "" {
		o.Components = splitCSV(componentCSV)
	}
	if server {
		o.Profile = "server"
	}
	o.Home = a.Home
	return (setup.Runner{Run: a.run}).Apply(o)
}

func (a *App) runSecrets(args []string) error {
	command := "status"
	if len(args) > 0 {
		command = args[0]
	}
	switch command {
	case "status":
		status, err := secrets.Status(a.Home, secrets.DefaultFields)
		if err != nil {
			return err
		}
		for _, line := range status {
			fmt.Fprintln(a.Out, line)
		}
		return nil
	case "edit":
		if !ui.Interactive() {
			return errors.New("secrets edit requires an interactive terminal")
		}
		return secrets.Prompt(a.Home, secrets.DefaultFields)
	default:
		return fmt.Errorf("unknown secrets command %q", command)
	}
}

func (a *App) runAuth(args []string) error {
	if len(args) > 0 && args[0] == "codex" {
		return a.runCodexAuth(args[1:])
	}
	command := "status"
	if len(args) > 0 {
		command = args[0]
	}
	switch command {
	case "status":
		return a.authStatus()
	case "login":
		return a.authLogin()
	default:
		return fmt.Errorf("unknown auth command %q", command)
	}
}

func (a *App) authStatus() error {
	for _, provider := range []struct {
		name, command string
		args          []string
	}{
		{"Codex", "codex", []string{"login", "status"}},
		{"Claude", "claude", []string{"auth", "status", "--json"}},
		{"GitHub", "gh", []string{"auth", "status"}},
		{"Firecrawl", "firecrawl", []string{"auth", "status"}},
	} {
		if _, err := exec.LookPath(provider.command); err != nil {
			fmt.Fprintf(a.Out, "%s: not installed\n", provider.name)
			continue
		}
		cmd := exec.Command(provider.command, provider.args...)
		cmd.Stdout, cmd.Stderr = a.Out, a.Err
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(a.Out, "%s: not logged in\n", provider.name)
		}
	}
	return nil
}

func (a *App) authLogin() error {
	if !ui.Interactive() {
		return errors.New("auth login requires an interactive terminal")
	}
	provider := "codex"
	if err := ui.Select("Which provider should sign in?", "Provider credentials stay managed by that provider.", []ui.Choice[string]{
		{Label: "Codex (device authentication)", Value: "codex"},
		{Label: "Claude Code", Value: "claude"},
		{Label: "GitHub CLI", Value: "github"},
		{Label: "Firecrawl", Value: "firecrawl"},
		{Label: "Pi — launches Pi; use /login", Value: "pi"},
	}, &provider); err != nil {
		return err
	}
	switch provider {
	case "codex":
		return a.loginCodex("main")
	case "claude":
		return a.run("claude", "auth", "login")
	case "github":
		return a.run("gh", "auth", "login")
	case "firecrawl":
		return a.run("firecrawl", "auth", "login")
	case "pi":
		return a.run("pi")
	}
	return nil
}

func (a *App) runCodexAuth(args []string) error {
	command := "list"
	if len(args) > 0 {
		command = args[0]
	}
	switch command {
	case "list":
		accounts, err := codexhome.Discover(a.Home)
		if err != nil {
			return err
		}
		if len(accounts) == 0 {
			fmt.Fprintln(a.Out, "No private Codex accounts found. Run `dotfiles auth codex login main` or `dotfiles auth codex add NAME`.")
		}
		for _, account := range accounts {
			fmt.Fprintf(a.Out, "%s\t%s\n", account.Name, account.Home)
		}
		return nil
	case "add":
		if len(args) != 2 {
			return errors.New("usage: dotfiles auth codex add NAME")
		}
		account, err := codexhome.Create(a.Home, args[1])
		if err != nil {
			return err
		}
		if err := a.syncLaunchers(); err != nil {
			return err
		}
		fmt.Fprintf(a.Out, "Created %s. Next: dotfiles auth codex login %s\n", account.Home, account.Name)
		return nil
	case "login":
		name := "main"
		if len(args) == 2 {
			name = args[1]
		} else if len(args) > 2 {
			return errors.New("usage: dotfiles auth codex login [NAME]")
		}
		return a.loginCodex(name)
	case "sync":
		return a.syncLaunchers()
	default:
		return fmt.Errorf("unknown Codex auth command %q", command)
	}
}

func (a *App) loginCodex(name string) error {
	path, err := codexhome.HomeFor(a.Home, name)
	if err != nil {
		return err
	}
	if name != "main" {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			if _, err := codexhome.Create(a.Home, name); err != nil {
				return err
			}
		}
		if err := codexhome.SyncShared(a.Home, path); err != nil {
			return err
		}
	}
	if err := a.runWithEnv([]string{"CODEX_HOME=" + path}, "codex", "-c", "cli_auth_credentials_store=\"file\"", "login", "--device-auth"); err != nil {
		return err
	}
	account := codexhome.Account{Name: name, Home: path, Main: name == "main"}
	if err := codexhome.EnsurePrivateAuth(account); err != nil {
		return err
	}
	return a.syncLaunchers()
}

func (a *App) syncLaunchers() error {
	accounts, err := codexhome.Discover(a.Home)
	if err != nil {
		return err
	}
	bin := filepath.Join(a.Home, ".local", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	for _, account := range accounts {
		if account.Main {
			continue
		}
		launcher := filepath.Join(bin, "codex-"+account.Name)
		if info, err := os.Lstat(launcher); err == nil {
			if info.Mode()&os.ModeSymlink == 0 {
				ui.Warn("preserving non-symlink launcher " + launcher)
				continue
			}
			if err := os.Remove(launcher); err != nil {
				return err
			}
		}
		if err := os.Symlink(self, launcher); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) runCodexCommand(args []string) error {
	flags := flag.NewFlagSet("codex", flag.ContinueOnError)
	flags.SetOutput(a.Err)
	account := flags.String("account", os.Getenv("DOTFILES_CODEX_ACCOUNT"), "Codex account")
	if err := flags.Parse(args); err != nil {
		return err
	}
	return a.RunCodex(*account, flags.Args())
}

func (a *App) RunCodex(name string, args []string) error {
	accounts, err := codexhome.Discover(a.Home)
	if err != nil {
		return err
	}
	if name == "" {
		if len(accounts) == 1 {
			name = accounts[0].Name
		} else if ui.Interactive() && len(accounts) > 0 {
			choices := make([]ui.Choice[string], 0, len(accounts))
			for _, account := range accounts {
				choices = append(choices, ui.Choice[string]{Label: account.Name, Value: account.Name})
			}
			if err := ui.Select("Choose a Codex account", "Each account keeps a private auth.json while sharing non-auth state.", choices, &name); err != nil {
				return err
			}
		} else {
			return errors.New("choose an account with --account NAME or DOTFILES_CODEX_ACCOUNT")
		}
	}
	path, err := codexhome.HomeFor(a.Home, name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("Codex account %q does not exist; run `dotfiles auth codex add %s`", name, name)
	}
	if err := codexhome.SyncShared(a.Home, path); err != nil {
		return err
	}
	codexArgs := append([]string{"-c", "cli_auth_credentials_store=\"file\""}, args...)
	return a.runWithEnv([]string{"CODEX_HOME=" + path}, "codex", codexArgs...)
}

func (a *App) runDoctor() error {
	fmt.Fprintln(a.Out, "Dotfiles doctor")
	for _, command := range []string{"stow", "zsh", "codex", "claude", "gh", "pi", "firecrawl"} {
		if path, err := exec.LookPath(command); err == nil {
			fmt.Fprintf(a.Out, "✓ %s: %s\n", command, path)
		} else {
			fmt.Fprintf(a.Out, "! %s: missing\n", command)
		}
	}
	if info, err := os.Lstat(filepath.Join(a.Home, ".config")); err == nil && info.Mode()&os.ModeSymlink != 0 {
		fmt.Fprintln(a.Out, "! ~/.config is folded into a repository; run `dotfiles setup --components config`")
	} else {
		fmt.Fprintln(a.Out, "✓ ~/.config is a normal directory")
	}
	return a.runCodexAuth([]string{"list"})
}

func (a *App) run(name string, args ...string) error {
	return a.runWithEnv(nil, name, args...)
}

func (a *App) runWithEnv(extra []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), extra...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, a.Out, a.Err
	return cmd.Run()
}

func splitCSV(value string) []string {
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}

func SourceHome() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex")
}
