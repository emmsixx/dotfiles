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
	Run   func(name string, args ...string) error
	Login func() error
}

type Component struct {
	ID          string
	Label       string
	Description string
	DesktopOnly bool
}

var Components = []Component{
	{"core", "Essentials", "Core terminal tools using Homebrew, apt, or pacman.", false},
	{"shell", "Shell", "Oh My Zsh, plugins, Starship, and shell dependencies.", false},
	{"runtimes", "Runtimes", "Bun, pnpm, FNM, and the pinned Node.js version.", false},
	{"agents", "Agent CLIs", "Codex, Claude Code, Pi, Firecrawl, and agent skills.", false},
	{"agent-customization", "Agent customization", "Claude plugins and Pi extension dependencies.", false},
	{"desktop", "Desktop tools", "Terminal and Linux Wayland conveniences. Not for servers.", true},
	{"t3-desktop", "T3 Code desktop app", "Optional stable desktop app; supported automatically on macOS and Arch Linux.", true},
	{"homebrew-upgrade", "Update Homebrew packages", "Runs brew update then brew upgrade. Optional: this may update unrelated tools.", false},
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
	var err error
	o.Repo, err = filepath.Abs(o.Repo)
	if err != nil {
		return err
	}
	o.Home, err = filepath.Abs(o.Home)
	if err != nil {
		return err
	}
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

func (o Options) canPrompt() bool { return !o.NonInteractive && ui.Interactive() }

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
		if component.ID == "homebrew-upgrade" && packageManager() != "brew" {
			continue
		}
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
	for _, id := range o.Components {
		known := false
		for _, component := range Components {
			if component.ID == id {
				known = true
			}
		}
		if !known {
			return fmt.Errorf("unknown component %q", id)
		}
		if o.Profile == "server" && (id == "desktop" || id == "t3-desktop") {
			return fmt.Errorf("%s is unavailable for servers", id)
		}
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
		case "homebrew-upgrade":
			err = r.updateHomebrew()
		case "config":
			err = r.initializeSubmodules(o.Repo)
			if err == nil {
				err = Link(o.Repo, o.Home, r.Run)
			}
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
	packages := []string{"curl", "git", "stow", "zsh", "jq", "wget", "unzip", "gh", "neovim", "bat", "lsd", "fzf", "zoxide", "sesh", "lazygit", "btop", "git-delta", "fastfetch", "yazi", "carapace", "superfile", "git-lfs", "ripgrep", "fd", "starship", "ffmpeg"}
	if packageManager() == "apt-get" {
		packages = replacePackage(packages, "fd", "fd-find")
	}
	if err := r.installPackages(packages); err != nil {
		return err
	}
	if packageManager() == "apt-get" {
		return ensureCommandAlias(o.Home, "fd", "fdfind")
	}
	return nil
}

func (r Runner) installShell(o Options) error {
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		return errors.New("zsh is unavailable; run the core component first")
	}
	// macOS already ships /bin/zsh as a supported login shell. Homebrew can put
	// another zsh earlier on PATH, but chsh rejects that non-system path.
	if runtime.GOOS != "darwin" && o.canPrompt() && needsZshLoginShell(os.Getenv("SHELL"), zsh) {
		makeDefault := true
		if err := ui.Confirm("Make Zsh your login shell?", "This runs chsh and may ask for your password.", &makeDefault); err != nil {
			return err
		}
		if makeDefault {
			if err := r.Run("chsh", "-s", zsh); err != nil {
				return err
			}
		}
	}
	if _, err := os.Stat(filepath.Join(o.Home, ".oh-my-zsh")); errors.Is(err, os.ErrNotExist) {
		if err := r.shell("export RUNZSH=no CHSH=no KEEP_ZSHRC=yes; " + installerCommand("https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh", "sh")); err != nil {
			return err
		}
	}
	for _, plugin := range []string{"zsh-autosuggestions", "zsh-syntax-highlighting"} {
		custom := os.Getenv("ZSH_CUSTOM")
		if custom == "" {
			custom = filepath.Join(o.Home, ".oh-my-zsh", "custom")
		}
		destination := filepath.Join(custom, "plugins", plugin)
		if info, err := os.Stat(destination); err == nil {
			if !info.IsDir() {
				return fmt.Errorf("plugin path %s is not a directory", destination)
			}
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := r.Run("git", "clone", "--depth", "1", "https://github.com/zsh-users/"+plugin, destination); err != nil {
			return err
		}
	}
	ui.OK("Oh My Zsh and shell plugins are ready")
	return nil
}

func needsZshLoginShell(currentShell, zshPath string) bool {
	return filepath.Base(strings.TrimSpace(currentShell)) != filepath.Base(zshPath)
}

func (r Runner) installRuntimes(o Options) error {
	if _, err := exec.LookPath("fnm"); err != nil {
		if err := r.shell(installerCommand("https://fnm.vercel.app/install", "bash", "--skip-shell")); err != nil {
			return err
		}
	}
	version, err := os.ReadFile(filepath.Join(o.Repo, ".node-version"))
	if err != nil {
		return err
	}
	if err := r.shell(fnmEnvironment() + "fnm install " + shellPath(strings.TrimSpace(string(version))) + " --use && fnm default " + shellPath(strings.TrimSpace(string(version)))); err != nil {
		return err
	}
	for _, tool := range []struct{ name, url, shell string }{{"bun", "https://bun.sh/install", "bash"}, {"pnpm", "https://get.pnpm.io/install.sh", "sh"}} {
		if err := r.shell("if ! command -v " + tool.name + " >/dev/null 2>&1; then " + installerCommand(tool.url, tool.shell) + "; fi"); err != nil {
			return err
		}
	}
	return nil
}

func (r Runner) installAgents(o Options) error {
	if _, err := exec.LookPath("claude"); err != nil {
		if err := r.shell(installerCommand("https://claude.ai/install.sh", "bash")); err != nil {
			return err
		}
	} else {
		ui.OK("Claude Code already installed")
	}
	// Pi's upstream installer may append an FNM-version-specific PATH entry to
	// .zshrc. Installing the published package through our selected FNM Node
	// keeps the shell configuration owned by this repository instead.
	if err := r.nodeShell("command -v pi >/dev/null 2>&1 || npm install -g --ignore-scripts --no-fund --no-audit @earendil-works/pi-coding-agent"); err != nil {
		return fmt.Errorf("install Pi (run runtimes first): %w", err)
	} else {
		ui.OK("Pi is ready")
	}
	if _, err := exec.LookPath("codex"); err != nil {
		if err := r.shell(installerCommand("https://chatgpt.com/codex/install.sh", "sh")); err != nil {
			return err
		}
	} else {
		ui.OK("Codex already installed")
	}
	if err := r.nodeShell("npm install -g firecrawl-cli"); err != nil {
		return fmt.Errorf("install Firecrawl (run runtimes first): %w", err)
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
		return r.Run("brew", "install", "--cask", "ghostty")
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

func (r Runner) updateHomebrew() error {
	if packageManager() != "brew" {
		return errors.New("Homebrew is required for the homebrew-upgrade component")
	}
	ui.Warn("updating all outdated Homebrew packages")
	if err := r.Run("brew", "update"); err != nil {
		return err
	}
	return r.Run("brew", "upgrade")
}

func (r Runner) configureSecretsAndAuth(o Options) error {
	created, err := secrets.Ensure(o.Home, filepath.Join(o.Repo, ".secrets.example"))
	if err != nil {
		return err
	}
	if created {
		ui.OK("created ~/.secrets with mode 0600")
	}
	if !o.canPrompt() {
		ui.Warn("run `dotfiles secrets edit` and `dotfiles auth login` from a terminal to finish onboarding")
		return r.configureGitIdentities(o)
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
	if err := r.configureGitIdentities(o); err != nil {
		return err
	}
	if r.Login != nil {
		login := false
		if err := ui.Confirm("Sign in to a provider now?", "You can also run dotfiles auth login later.", &login); err != nil {
			return err
		}
		if login {
			return r.Login()
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
	if channel == "" && o.canPrompt() {
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
		return errors.New("no supported package manager found; install Homebrew on macOS, or use an apt/pacman Linux host (otherwise install the requested tools manually)")
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
	args := packageInstallArgs(manager, available)
	if (manager == "apt-get" || manager == "pacman") && os.Geteuid() != 0 {
		return r.Run("sudo", append([]string{manager}, args...)...)
	}
	return r.Run(manager, args...)
}

func (r Runner) shell(command string) error {
	if zsh, err := exec.LookPath("zsh"); err == nil {
		return r.Run(zsh, "-e", "-c", command)
	}
	return r.Run("sh", "-e", "-c", command)
}

func (r Runner) nodeShell(command string) error {
	return r.shell(fnmEnvironment() + command)
}

func fnmEnvironment() string {
	shell := "bash"
	if _, err := exec.LookPath("zsh"); err == nil {
		shell = "zsh"
	}
	return "export PATH=\"$HOME/.local/share/fnm:$HOME/Library/Application Support/fnm:$HOME/.local/bin:$HOME/.bun/bin:$HOME/.local/share/pnpm:$PATH\"; fnm_env=$(fnm env --shell " + shell + "); eval \"$fnm_env\"; "
}

// Download and execute in separate checked steps. The surrounding shell runs
// with errexit, so neither a failed HTTP request nor a failed installer can
// be hidden by a successful command later in the sequence.
func installerCommand(url, interpreter string, args ...string) string {
	command := "install_tmp=$(mktemp -d); trap 'rm -rf \"$install_tmp\"' EXIT; curl -fsSL " + shellPath(url) + " -o \"$install_tmp/install\"; test -s \"$install_tmp/install\"; " + shellPath(interpreter) + " \"$install_tmp/install\""
	for _, arg := range args {
		command += " " + shellPath(arg)
	}
	return command
}

// Initialize only missing submodules; preserve already initialized checkouts,
// including a user's local changes or deliberately checked-out revision.
func (r Runner) initializeSubmodules(repo string) error {
	if _, err := os.Stat(filepath.Join(repo, ".gitmodules")); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	output, err := exec.Command("git", "-C", repo, "submodule", "status", "--recursive").Output()
	if err != nil {
		return fmt.Errorf("inspect submodules: %w", err)
	}
	for _, line := range strings.Split(string(output), "\n") {
		if !strings.HasPrefix(line, "-") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			return fmt.Errorf("invalid submodule status %q", line)
		}
		if err := r.Run("git", "-C", repo, "submodule", "update", "--init", "--recursive", "--", parts[1]); err != nil {
			return err
		}
	}
	return nil
}

func (r Runner) installSkills(o Options) error {
	contents, err := os.ReadFile(filepath.Join(o.Repo, ".skills"))
	if err != nil {
		return err
	}
	ui.Heading("Installing agent skills")
	installed := 0
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
			if err := r.runSkillInstall(o.Repo, repo, ""); err != nil {
				ui.Warn("could not install skill source " + repo)
			} else {
				installed++
			}
			continue
		}
		for _, skill := range strings.Split(parts[1], ",") {
			skill = strings.TrimSpace(skill)
			if skill == "" {
				continue
			}
			if err := r.runSkillInstall(o.Repo, repo, skill); err != nil {
				ui.Warn("could not install skill " + repo + "@" + skill)
			} else {
				installed++
			}
		}
	}
	ui.OK(fmt.Sprintf("%d agent skill installs complete", installed))
	return nil
}

// runSkillInstall keeps successful skills.sh output out of setup's terminal.
// On failure it prints the captured diagnostic, then returns an error so the
// caller can say exactly which source needs attention.
func (r Runner) runSkillInstall(repoDir, source, skill string) error {
	command := "cd " + shellPath(repoDir) + " && npx -y skills add " + shellPath(source)
	if skill != "" {
		command += " --skill " + shellPath(skill)
	}
	command += " -y </dev/null"
	return r.nodeShell(quietCommand(command))
}

func quietCommand(command string) string {
	return "log=$(mktemp); trap 'rm -f \"$log\"' EXIT; if ! (" + command + ") >\"$log\" 2>&1; then cat \"$log\" >&2; exit 1; fi"
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

func packageInstallArgs(manager string, available []string) []string {
	args := append([]string(nil), available...)
	switch manager {
	case "brew":
		return append([]string{"install"}, args...)
	case "apt-get":
		return append([]string{"install", "-y"}, args...)
	case "pacman":
		return append([]string{"-S", "--needed", "--noconfirm"}, args...)
	default:
		return args
	}
}

func replacePackage(packages []string, old, replacement string) []string {
	result := append([]string(nil), packages...)
	for index, packageName := range result {
		if packageName == old {
			result[index] = replacement
		}
	}
	return result
}

func ensureCommandAlias(home, expected, actual string) error {
	if _, err := exec.LookPath(expected); err == nil {
		return nil
	}
	path, err := exec.LookPath(actual)
	if err != nil {
		return nil
	}
	bin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		return err
	}
	link := filepath.Join(bin, expected)
	if _, err := os.Lstat(link); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Symlink(path, link)
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
