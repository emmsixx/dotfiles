# Dotfiles

These are all of my dotfiles, for now it only contains cross-platform software to sync between my MacOS and Linux machines. The plan is to add everything eventually, but for now I'm just keeping it simple.


## Setup

```sh
git clone https://github.com/emmsixx/dotfiles ~/dotfiles
cd ~/dotfiles
./setup.sh
```

This will install all dependencies, set up Oh My Zsh and plugins, install Claude Code and its plugins, initialize submodules, create `~/.secrets` from the template, and symlink everything to `$HOME`.


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
| `kc` | kilo |
| `yeet` | git add, commit, push (interactive) |
| `y` | yazi |


## Claude Code

`CLAUDE.md` references `GH_WORK_USER` and `GH_PERSONAL_USER` from `~/.secrets`. The setup script copies `.secrets.example` to `~/.secrets` automatically — fill in your values before starting a new shell.


## Structure

```
dotfiles/
├── .claude/       # Claude Code configuration (CLAUDE.md, settings.json)
├── .config/       # Application configurations
├── .scripts/      # Utility shell scripts
├── .zshrc         # Zsh configuration
├── .gitconfig     # Git configuration
└── .gitignore
```
