# Dotfiles

Personal dotfiles for macOS and Linux, including cross-platform shell config and Linux-specific application themes and settings.


## Setup

```sh
git clone https://github.com/emmsixx/dotfiles ~/dotfiles
cd ~/dotfiles
./setup.sh
```

This will install all dependencies, set up Oh My Zsh and plugins, install Starship, install Claude Code and its plugins, install agent skills into the dotfiles repo, initialize submodules, create `~/.secrets` from the template, and symlink the tracked configs plus generated agent skill links into `$HOME`.


## Requirements

The setup script handles most dependencies automatically. Install these manually:

- [GoogleSansCode Nerd Font](https://github.com/E-Vertin/GoogleSansCode-NerdFont)
- Clipboard tool (`wl-clipboard` on Wayland, `pbcopy` built-in on macOS)


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
| `kc` | kilo |
| `yeet` | git add, commit, push (interactive) |
| `y` | yazi |


## Claude Code

`CLAUDE.md` references `GH_WORK_USER` and `GH_PERSONAL_USER` from `~/.secrets`. The setup script copies `.secrets.example` to `~/.secrets` automatically — fill in your values before starting a new shell.


## Skills

Skills are declared in `.skills` and installed automatically by `setup.sh` via [skills.sh](https://skills.sh/).

- Skill contents are installed locally into the dotfiles repo and are **not committed**.
- Generated skill artifacts such as `.agents/skills`, agent-specific `*/skills` directories, and `skills-lock.json` are gitignored.
- `stow` then links the generated Claude/Pi skill entries into `$HOME`.
- Codex-specific `Uncodixfy` is generated from the locally installed skill and exposed via `~/.codex/instructions/Uncodixfy`.

To add a new skill repo, append it to `.skills` and rerun:

```sh
./setup.sh
```


## Structure

```
dotfiles/
├── .claude/       # Claude Code configuration (CLAUDE.md, settings.json)
├── .codex/        # Codex configuration
├── .config/       # Application configurations
├── .pi/           # Pi configuration
├── .scripts/      # Utility shell scripts
├── .zshrc         # Zsh configuration
├── .gitconfig     # Git configuration
└── .gitignore
```
