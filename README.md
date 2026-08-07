# Dotfiles

An interactive development-environment setup tool for this dotfiles repository.
It uses Stow for configuration management; it does not try to replace a general
purpose dotfiles manager such as chezmoi.

## Setup

```sh
git clone https://github.com/emmsixx/dotfiles ~/dotfiles
cd ~/dotfiles
./setup.sh
```

The tiny shell bootstrapper downloads a verified release of the `dotfiles` Go
binary, then launches an interactive Charm/Huh wizard. Choose a desktop or
server profile and the groups you want: essentials, shell, runtimes, agent
CLIs, agent customization, desktop tools, the optional T3 Code desktop app,
linking, and secrets/authentication. The desktop-app installer only acts on
officially supported paths: Homebrew Cask on macOS and `yay`/`paru` on Arch.
The Shell group runs commands through Zsh once available and offers to make it
your login shell.

For the prior "install the normal profile" behavior, use:

```sh
./setup.sh --defaults
```

For a headless server:

```sh
./setup.sh --profile server --defaults
```

`setup-server.sh` remains as a compatibility shortcut for the latter command.
Use `--components core,shell,config --non-interactive` for automation. The only
service offered by setup is the opt-in Linux server T3 Code background service:

```sh
./setup.sh --profile server --defaults --t3-service nightly
# or: --t3-service latest
```

Nightly is its default channel. This service is intentionally not selected by
default and is usually unnecessary on a desktop.

On a Homebrew host, the picker also offers **Update Homebrew packages**. It runs
`brew update` followed by `brew upgrade`; it is intentionally opt-in because it
can update tools unrelated to this repository.

## CLI

```text
dotfiles setup [--defaults] [--components ...]
dotfiles secrets status|edit
dotfiles auth status|login
dotfiles auth codex list|add NAME|login [NAME]|sync
dotfiles codex [--account NAME] -- [codex arguments]
dotfiles doctor
```

The initial auth wizard offers Codex, Claude Code, GitHub CLI, Firecrawl, and
Pi. Secrets remain in `~/.secrets` with mode `0600`; existing comments and
unrelated exports are preserved.

Codex accounts use T3 Code-compatible homes: `~/.codex` for main and
`~/.codex-NAME` for named accounts. Named homes share non-auth Codex state but
keep `auth.json` private. `cx` opens an account picker, and authenticated named
accounts get `codex-NAME` launchers in `~/.local/bin`. Claude multi-account is
deliberately deferred because its credential-store isolation is not yet reliable
across platforms.

## Requirements

Released binaries work on Linux and macOS without a local Go toolchain. The
bootstrapper handles most dependencies selected in the wizard. Install these
manually when relevant:

- [GoogleSansCode Nerd Font](https://github.com/E-Vertin/GoogleSansCode-NerdFont)
- Clipboard tool (`pbcopy` is built in on macOS; select Desktop tools to attempt
  `wl-clipboard` on Linux Wayland)

## Code Style

- Go code is formatted with `gofmt`.
- Shell bootstrap scripts are POSIX `sh`.
- Use four spaces in shell scripts and avoid unnecessary inline comments.

## Key Aliases

| Alias | Command |
|-------|---------|
| `n` | nvim |
| `b` | bat |
| `ls` | lsd |
| `cc` | claude |
| `ccsp` | claude --dangerously-skip-permissions |
| `cx` | Codex account picker |
| `yeet` | git add, commit, push (interactive) |
| `y` | yazi |

## Claude Code

`CLAUDE.md` references `GH_WORK_USER` and `GH_PERSONAL_USER` from
`~/.secrets`. Run `dotfiles secrets edit` to fill in values.

## Pi

The Agent CLIs group installs Pi from npm through the Node version managed by
FNM. It deliberately does not run Pi's shell installer, so setup never adds a
version-specific Pi path to `.zshrc`.
After installation, run `pi` and use `/login` if you want to authenticate with
your existing subscription instead of an API key.

## Skills

Skills are declared in `.skills` and are installed by the Agent CLIs group via
[skills.sh](https://skills.sh/).

- Skill contents are installed locally into `.agents/skills` and are not committed.
- Generated Claude/Pi skill links are ignored and linked into `$HOME` by setup.
- `skills-lock.json` is tracked with installed skill source and hash metadata.
- Use `owner/repo|skill1,skill2` in `.skills` to install only selected skills.

To add a skill repo, append it to `.skills` and rerun:

```sh
dotfiles setup --components agents
```

## Node.js

The Runtimes group installs FNM and uses the exact version in `.node-version`
for Pi's installer and other npm-based tools. Upgrades are intentional rather
than automatic.

```sh
fnm install --lts --use
node --version > .node-version
dotfiles setup --components runtimes
```

## Structure

```text
dotfiles/
├── cmd/dotfiles/   # Go CLI entry point
├── internal/       # setup, secrets, Codex-home, and UI logic
├── .claude/        # Claude Code configuration
├── .config/        # Application configurations
├── .pi/            # Pi configuration
├── .scripts/       # Utility shell scripts
├── .goreleaser.yaml
└── setup.sh         # release-binary bootstrapper
```
