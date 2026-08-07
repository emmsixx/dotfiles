package setup

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/emmsixx/dotfiles/internal/secrets"
	"github.com/emmsixx/dotfiles/internal/ui"
)

type Options struct {
	Repo           string
	Home           string
	Profile        string
	Defaults       bool
	NonInteractive bool
	Components     []string
	T3Channel      string
}

type Runner struct {
	Run func(name string, args ...string) error
}

type Component struct {
	ID          string
	Label       string
	Description string
	DesktopOnly bool
}

var Components = []Component{
	{"core", "Essentials", "Core terminal tools and package-manager setup.", false},
	{"shell", "Shell", "Oh My Zsh, plugins, Starship, and shell dependencies.", false},
	{"runtimes", "Runtimes", "Bun, pnpm, FNM, and the pinned Node.js version.", false},
	{"agents", "Agent CLIs", "Codex, Claude Code, Pi, Firecrawl, and agent skills.", false},
	{"agent-customization", "Agent customization", "Claude plugins and Pi extension dependencies.", false},
	{"desktop", "Desktop tools", "Terminal and Linux Wayland conveniences. Not for servers.", true},
	{"t3-desktop", "T3 Code desktop app", "Optional stable desktop app; supported automatically on macOS and Arch Linux.", true},
	{"config", "Link dotfiles", "Safely stow tracked configuration into your home directory.", false},
	{"secrets-auth", "Secrets and sign-in", "Create ~/.secrets and offer provider login flows.", false},
	{"t3-service", "T3 Code background service", "Linux systemd server feature; usually not recommended on a desktop.", true},
}

func (o *Options) Normalize() error {
	if o.Home == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		o.Home = home
	}
	if o.Repo == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		o.Repo = cwd
	}
	o.Repo = filepath.Clean(o.Repo)
	if o.Profile == "" {
		o.Profile = "desktop"
	}
	if o.Profile != "desktop" && o.Profile != "server" {
		return fmt.Errorf("unknown profile %q", o.Profile)
	}
	if o.T3Channel != "" && o.T3Channel != "nightly" && o.T3Channel != "latest" {
		return fmt.Errorf("unknown T3 channel %q", o.T3Channel)
	}
	return nil
}

func (o Options) DefaultComponents() []string {
	result := []string{"core", "shell", "runtimes", "agents", "agent-customization", "config", "secrets-auth"}
	if o.Profile == "desktop" {
		result = append(result, "desktop")
	}
	return result
}

func (o *Options) Choose() error {
	if o.NonInteractive && !o.Defaults && len(o.Components) == 0 {
		return errors.New("non-interactive setup requires --defaults or --components")
	}
	if o.Defaults || len(o.Components) > 0 {
		if len(o.Components) == 0 {
			o.Components = o.DefaultComponents()
		}
		if o.T3Channel != "" && !contains(o.Components, "t3-service") {
			o.Components = append(o.Components, "t3-service")
		}
		return nil
	}
	if o.NonInteractive || !ui.Interactive() {
		return errors.New("setup needs a terminal; use --defaults or --components")
	}
	profile := o.Profile
	if err := ui.Select("Where is this machine used?", "This sets sensible defaults; you can still select individual groups.", []ui.Choice[string]{
		{Label: "Desktop — full local workstation", Value: "desktop"},
		{Label: "Server — CLI and agent tooling only", Value: "server"},
	}, &profile); err != nil {
		return err
	}
	o.Profile = profile
	choices := make([]ui.Choice[string], 0, len(Components))
	defaults := map[string]bool{}
	for _, id := range o.DefaultComponents() {
		defaults[id] = true
	}
	for _, component := range Components {
		if component.ID == "t3-service" && (o.Profile != "server" || runtime.GOOS != "linux") {
			continue
		}
		if component.DesktopOnly && component.ID != "t3-service" && o.Profile == "server" {
			continue
		}
		choices = append(choices, ui.Choice[string]{Label: component.Label, Value: component.ID})
	}
	selected := make([]string, 0, len(defaults))
	for _, choice := range choices {
		if defaults[choice.Value] {
			selected = append(selected, choice.Value)
		}
	}
	if err := ui.MultiSelect("What should dotfiles set up?", "Space toggles items. The T3 service is intentionally opt-in.", choices, &selected); err != nil {
		return err
	}
	o.Components = selected
	if o.T3Channel != "" && !contains(o.Components, "t3-service") {
		o.Components = append(o.Components, "t3-service")
	}
	return nil
}

func (r Runner) Apply(o Options) error {
	if err := o.Normalize(); err != nil {
		return err
	}
	if err := o.Choose(); err != nil {
		return err
	}
	if err := validateRepo(o.Repo); err != nil {
		return err
	}
	ui.Heading("Setting up dotfiles")
	for _, component := range o.Components {
		var err error
		switch component {
		case "core":
			err = r.installCore(o)
		case "shell":
			err = r.installShell(o)
		case "runtimes":
			err = r.installRuntimes(o)
		case "agents":
			err = r.installAgents(o)
		case "agent-customization":
			err = r.installAgentCustomization(o)
		case "desktop":
			err = r.installDesktop(o)
		case "t3-desktop":
			err = r.installT3Desktop(o)
		case "config":
			err = Link(o.Repo, o.Home, r.Run)
		case "secrets-auth":
			err = r.configureSecretsAndAuth(o)
		case "t3-service":
			err = r.installT3Service(o)
		default:
			err = fmt.Errorf("unknown component %q", component)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", component, err)
		}
	}
	return writeProfile(o.Home, o.Profile)
}

func (r Runner) installCore(o Options) error {
	return r.installPackages([]string{"curl", "git", "stow", "zsh", "jq", "wget", "unzip", "gh", "neovim", "bat", "lsd", "fzf", "zoxide", "sesh", "lazygit", "btop", "git-delta", "fastfetch", "yazi", "carapace", "superfile", "git-lfs", "ripgrep", "fd-find", "starship", "ffmpeg"})
}

func (r Runner) installShell(o Options) error {
	if _, err := exec.LookPath("zsh"); err != nil {
		ui.Warn("zsh is unavailable; install the Shell component's package prerequisites first")
		return nil
	}
	if _, err := os.Stat(filepath.Join(o.Home, ".oh-my-zsh")); errors.Is(err, os.ErrNotExist) {
		return r.shell("RUNZSH=no CHSH=no sh -c \"$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)\"")
	}
	ui.OK("Oh My Zsh already installed")
	return nil
}

func (r Runner) installRuntimes(o Options) error {
	if _, err := exec.LookPath("fnm"); err != nil {
		if err := r.shell("curl -fsSL https://fnm.vercel.app/install | bash -s -- --skip-shell"); err != nil {
			return err
		}
	}
	version, err := os.ReadFile(filepath.Join(o.Repo, ".node-version"))
	if err != nil {
		return err
	}
	if err := r.shell("export PATH=\"$HOME/.local/share/fnm:$PATH\"; eval \"$(fnm env --shell sh)\"; fnm install " + strings.TrimSpace(string(version)) + " --use; fnm default " + strings.TrimSpace(string(version))); err != nil {
		return err
	}
	return r.shell("command -v bun >/dev/null 2>&1 || curl -fsSL https://bun.sh/install | bash; command -v pnpm >/dev/null 2>&1 || curl -fsSL https://get.pnpm.io/install.sh | sh -")
}

func (r Runner) installAgents(o Options) error {
	for _, command := range []string{
		"curl -fsSL https://claude.ai/install.sh | bash",
		"curl -fsSL https://pi.dev/install.sh | sh",
		"curl -fsSL https://chatgpt.com/codex/install.sh | sh",
	} {
		if err := r.shell(command); err != nil {
			return err
		}
	}
	if err := r.nodeShell("npm install -g firecrawl-cli"); err != nil {
		ui.Warn("could not install Firecrawl; run the Runtimes component first")
	}
	return r.installSkills(o)
}

func (r Runner) installAgentCustomization(o Options) error {
	if _, err := exec.LookPath("claude"); err == nil {
		plugins := []string{"frontend-design@claude-plugins-official", "claude-code-setup@claude-plugins-official", "code-review@claude-plugins-official", "feature-dev@claude-plugins-official", "typescript-lsp@claude-plugins-official", "explanatory-output-style@claude-plugins-official", "claude-md-management@claude-plugins-official", "svelte"}
		for _, plugin := range plugins {
			if err := r.Run("claude", "plugin", "install", plugin); err != nil {
				ui.Warn("could not install Claude plugin " + plugin)
			}
		}
	}
	extensions := filepath.Join(o.Repo, ".pi", "agent", "extensions", "package-lock.json")
	if _, err := os.Stat(extensions); err == nil {
		return r.nodeShell("npm --prefix " + shellPath(filepath.Dir(extensions)) + " ci --omit=dev")
	}
	return nil
}

func (r Runner) installDesktop(o Options) error {
	if o.Profile == "server" {
		return errors.New("desktop tools cannot be selected for the server profile")
	}
	if runtime.GOOS == "darwin" {
		return r.shell("command -v brew >/dev/null && brew install --cask ghostty || true")
	}
	if runtime.GOOS == "linux" {
		return r.installPackages([]string{"tmux", "grim", "slurp", "satty", "solaar", "wl-clipboard", "libnotify-bin", "file"})
	}
	return nil
}

func (r Runner) installT3Desktop(o Options) error {
	if o.Profile != "desktop" {
		return errors.New("the T3 desktop app is not available for the server profile")
	}
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("brew"); err != nil {
			ui.Warn("install T3 Code manually: Homebrew is unavailable")
			return nil
		}
		return r.Run("brew", "install", "--cask", "t3-code")
	case "linux":
		if _, err := exec.LookPath("yay"); err == nil {
			return r.Run("yay", "-S", "--noconfirm", "t3code-bin")
		}
		if _, err := exec.LookPath("paru"); err == nil {
			return r.Run("paru", "-S", "--noconfirm", "t3code-bin")
		}
		ui.Warn("install T3 Code manually from GitHub Releases; automatic desktop installation is only supported on macOS and Arch Linux")
		return nil
	default:
		ui.Warn("install T3 Code manually; this platform has no supported automatic installer")
		return nil
	}
}

func (r Runner) configureSecretsAndAuth(o Options) error {
	created, err := secrets.Ensure(o.Home, filepath.Join(o.Repo, ".secrets.example"))
	if err != nil {
		return err
	}
	if created {
		ui.OK("created ~/.secrets with mode 0600")
	}
	if o.NonInteractive || !ui.Interactive() {
		ui.Warn("run `dotfiles secrets edit` and `dotfiles auth login` from a terminal to finish onboarding")
		return nil
	}
	edit := created
	if !created {
		if err := ui.Confirm("Update secrets now?", "Existing values are shown only by name; API keys stay masked.", &edit); err != nil {
			return err
		}
	}
	if edit {
		if err := secrets.Prompt(o.Home, secrets.DefaultFields); err != nil {
			return err
		}
	}
	return nil
}

func (r Runner) installT3Service(o Options) error {
	if runtime.GOOS != "linux" || o.Profile != "server" {
		return errors.New("the T3 background service is available only for the Linux server profile")
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return errors.New("systemd is required for the T3 background service")
	}
	channel := o.T3Channel
	if channel == "" && ui.Interactive() {
		channel = "nightly"
		if err := ui.Select("T3 Code release channel", "Nightly is the default. Choose latest if you prefer stable releases.", []ui.Choice[string]{
			{Label: "Nightly (default)", Value: "nightly"},
			{Label: "Latest stable", Value: "latest"},
		}, &channel); err != nil {
			return err
		}
	}
	if channel == "" {
		return errors.New("pass --t3-service nightly or --t3-service latest in non-interactive mode")
	}
	return r.nodeShell("npx -y t3@" + channel + " service install")
}

func (r Runner) installPackages(packages []string) error {
	manager := packageManager()
	if manager == "" {
		ui.Warn("no supported package manager found; install manually: " + strings.Join(packages, ", "))
		return nil
	}
	available := make([]string, 0, len(packages))
	for _, packageName := range packages {
		if packageAvailable(manager, packageName) {
			available = append(available, packageName)
		} else {
			ui.Warn(packageName + " is not available through " + manager + "; skipping")
		}
	}
	if len(available) == 0 {
		return nil
	}
	args := available
	switch manager {
	case "brew":
		args = append([]string{"install"}, packages...)
	case "apt-get":
		args = append([]string{"install", "-y"}, packages...)
	case "pacman":
		args = append([]string{"-S", "--needed", "--noconfirm"}, packages...)
	}
	if (manager == "apt-get" || manager == "pacman") && os.Geteuid() != 0 {
		return r.Run("sudo", append([]string{manager}, args...)...)
	}
	return r.Run(manager, args...)
}

func (r Runner) shell(command string) error { return r.Run("sh", "-c", command) }

func (r Runner) nodeShell(command string) error {
	return r.shell("export PATH=\"$HOME/.local/share/fnm:$PATH\"; eval \"$(fnm env --shell sh)\"; " + command)
}

func (r Runner) installSkills(o Options) error {
	contents, err := os.ReadFile(filepath.Join(o.Repo, ".skills"))
	if err != nil {
		return err
	}
	for _, rawLine := range strings.Split(string(contents), "\n") {
		entry := strings.TrimSpace(strings.SplitN(rawLine, "#", 2)[0])
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "|", 2)
		repo := strings.TrimSpace(parts[0])
		if repo == "" {
			continue
		}
		if len(parts) == 1 {
			if err := r.nodeShell("cd " + shellPath(o.Repo) + " && npx -y skills add " + shellPath(repo) + " -y </dev/null"); err != nil {
				ui.Warn("could not install skill source " + repo)
			}
			continue
		}
		for _, skill := range strings.Split(parts[1], ",") {
			skill = strings.TrimSpace(skill)
			if skill == "" {
				continue
			}
			command := "cd " + shellPath(o.Repo) + " && npx -y skills add " + shellPath(repo) + " --skill " + shellPath(skill) + " -y </dev/null"
			if err := r.nodeShell(command); err != nil {
				ui.Warn("could not install skill " + repo + "@" + skill)
			}
		}
	}
	return nil
}

func shellPath(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func packageManager() string {
	for _, candidate := range []string{"brew", "pacman", "apt-get"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func packageAvailable(manager, packageName string) bool {
	var command *exec.Cmd
	switch manager {
	case "brew":
		command = exec.Command("brew", "info", "--formula", packageName)
	case "apt-get":
		command = exec.Command("apt-cache", "show", packageName)
	case "pacman":
		command = exec.Command("pacman", "-Si", packageName)
	default:
		return false
	}
	return command.Run() == nil
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func validateRepo(repo string) error {
	for _, required := range []string{".zshrc", ".secrets.example", ".stow-local-ignore"} {
		if _, err := os.Stat(filepath.Join(repo, required)); err != nil {
			return fmt.Errorf("%s is not a dotfiles repository (%s missing)", repo, required)
		}
	}
	return nil
}

func writeProfile(home, profile string) error {
	path := filepath.Join(home, ".local", "state", "dotfiles", "profile")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(profile+"\n"), 0o600)
}
