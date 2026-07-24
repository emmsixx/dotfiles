# Dotfiles

Personal dotfiles for macOS and Linux, including cross-platform shell config and Linux-specific application themes and settings.


## Setup

```sh
git clone https://github.com/emmsixx/dotfiles ~/dotfiles
cd ~/dotfiles
./setup.sh
```

For a headless Debian server or homelab VM, use the server profile instead:

```sh
./setup-server.sh
```

The desktop profile installs the full workstation setup. The server profile installs the shared CLI and agent tooling while skipping desktop-specific packages such as Ghostty, tmux (including shell auto-attach), Pokémon-colorscripts, and Linux Wayland utilities.

This will install the selected dependencies, set up Oh My Zsh and plugins, install Starship, Claude Code and its plugins, Pi, Codex, and Firecrawl, install agent skills into the dotfiles repo, initialize submodules, create `~/.secrets` from the template, and symlink the tracked configs plus generated agent skill links into `$HOME`.


## Requirements

The setup script handles most dependencies automatically. Install these manually:

- [GoogleSansCode Nerd Font](https://github.com/E-Vertin/GoogleSansCode-NerdFont)
- Clipboard tool (`pbcopy` is built in on macOS; setup attempts `wl-clipboard` on Linux Wayland and falls back to a manual install if unavailable)


## Code Style

- Shell scripts should be POSIX-compliant where possible
- Use 4-space indentation for shell scripts
- Avoid inline comments unless necessary


## Key Aliases

| Alias | Command |
|-------|---------|
| `n` | nvim |
| `b` | bat |
| `ls` | lsd |
| `cc` | claude |
| `ccsp` | claude --dangerously-skip-permissions |
| `cx` | codex |
| `yeet` | git add, commit, push (interactive) |
| `y` | yazi |


## Claude Code

`CLAUDE.md` references `GH_WORK_USER` and `GH_PERSONAL_USER` from `~/.secrets`. The setup script copies `.secrets.example` to `~/.secrets` automatically — fill in your values before starting a new shell.

## Pi

`setup.sh` installs Pi with `npm install -g @earendil-works/pi-coding-agent`. After installation, run `pi` and use `/login` if you want to authenticate with your existing subscription instead of an API key.


## Skills

Skills are declared in `.skills` and installed automatically by `setup.sh` via [skills.sh](https://skills.sh/).

- Skill contents are installed locally into `.agents/skills` and are not committed.
- Generated Claude/Pi skill links are also ignored and linked into `$HOME` by setup.
- `skills-lock.json` is tracked with installed skill source and hash metadata; setup uses it to prune stale generated skills.

To add a new skill repo, append it to `.skills` and rerun:

```sh
./setup.sh
```


## Node.js

Setup installs FNM and uses the exact version in `.node-version` for npm-based tools. It reports when a newer major Node.js LTS release is available; upgrades are intentional rather than automatic.

To upgrade deliberately:

```sh
fnm install --lts --use
node --version > .node-version
./setup.sh
```

## Future Plans / Ideas

- Switch to `varlock` / `infiscal` for environment variable and secret handling
- Add an interactive CLI for setup and maintenance tasks
- Potentially prompt for environment variables during setup, though this may be more annoying than helpful

## Structure

```
dotfiles/
├── .claude/       # Claude Code configuration (CLAUDE.md, settings.json)
├── .config/       # Application configurations
├── .pi/           # Pi configuration
├── .node-version  # Pinned Node.js version for FNM
├── .scripts/      # Utility shell scripts
├── .zshrc         # Zsh configuration
├── .gitconfig     # Git configuration
└── .gitignore
```
