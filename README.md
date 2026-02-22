# Dotfiles

These are all of my dotfiles, for now it only contains cross-platform software to sync between my MacOS and Linux machines. The plan is to add everything eventually, but for now I'm just keeping it simple.


## Requirements
For the dotfiles to function properly, install the following.

- Basic utils: `git`, `make`, `unzip`, C Compiler (`gcc`)
- [ripgrep](https://github.com/BurntSushi/ripgrep#installation),
  [fd-find](https://github.com/sharkdp/fd#installation)
  [lsd](https://github.com/lsd-rs/lsd#installation)
- Clipboard tool (xclip/xsel/win32yank or other depending on the platform)
- [GoogleSansCode Nerd Font](https://github.com/E-Vertin/GoogleSansCode-NerdFont)
- [oh-my-tmux](https://github.com/gpakosz/.tmux)


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
| `kc` | kilo |


## Claude Code

`settings.json` enables plugins but does not install them. On a new machine, install them manually:

```sh
claude plugin install frontend-design@claude-plugins-official
claude plugin install claude-code-setup@claude-plugins-official
claude plugin install code-review@claude-plugins-official
claude plugin install feature-dev@claude-plugins-official
claude plugin install typescript-lsp@claude-plugins-official
claude plugin install explanatory-output-style@claude-plugins-official
claude plugin install claude-md-management@claude-plugins-official
```

Also add `GH_WORK_USER` and `GH_PERSONAL_USER` to `~/.secrets` — referenced in `CLAUDE.md` but not tracked.


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
