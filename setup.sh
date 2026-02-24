#!/usr/bin/env bash

set -e

DOTFILES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_DIR="$HOME"

has()  { command -v "$1" &>/dev/null; }
info() { printf '\033[1;34m==> %s\033[0m\n' "$*"; }
ok()   { printf '\033[1;32m ✓ %s\033[0m\n' "$*"; }
skip() { printf '\033[2m · %s already installed\033[0m\n' "$*"; }
warn() { printf '\033[1;33m ! %s\033[0m\n' "$*"; }

# ── OS & Package Manager ──────────────────────────────────────────────────────

OS="$(uname -s)"

if [ "$OS" = "Darwin" ]; then
    if ! has brew; then
        info "Installing Homebrew..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    fi
    pkg_install() { brew install "$@"; }
elif has pacman; then
    pkg_install() { sudo pacman -S --noconfirm "$@"; }
elif has apt-get; then
    pkg_install() { sudo apt-get install -y "$@"; }
else
    warn "Unknown package manager — install packages manually."
    pkg_install() { warn "Cannot auto-install: $*"; }
fi

try_install() {
    local binary="$1" package="${2:-$1}"
    if has "$binary"; then skip "$binary"; else info "Installing $binary..."; pkg_install "$package"; ok "$binary"; fi
}

# ── Core Tools ────────────────────────────────────────────────────────────────

info "Checking core tools..."
try_install stow
try_install git
try_install gh
try_install nvim neovim
try_install bat
try_install lsd
try_install fzf
try_install zoxide
try_install tmux
try_install sesh
try_install lazygit
try_install btop
try_install delta git-delta
try_install fastfetch
try_install yazi
try_install carapace
try_install spf superfile
try_install git-lfs
try_install rg ripgrep
try_install fd
try_install starship

if [ "$OS" = "Darwin" ]; then
    try_install ghostty
fi

# ── pokemon-colorscripts ──────────────────────────────────────────────────────

if ! has pokemon-colorscripts; then
    info "Installing pokemon-colorscripts..."
    if [ "$OS" = "Darwin" ]; then
        brew install --no-quarantine phisch/tap/pokemon-colorscripts
    elif has yay; then
        yay -S --noconfirm pokemon-colorscripts-git
    else
        warn "Install pokemon-colorscripts manually: https://gitlab.com/phoneybadger/pokemon-colorscripts"
    fi
else
    skip "pokemon-colorscripts"
fi

# ── Oh My Zsh ────────────────────────────────────────────────────────────────

if [ ! -d "$HOME/.oh-my-zsh" ]; then
    info "Installing Oh My Zsh..."
    RUNZSH=no CHSH=no sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"
    ok "oh-my-zsh"
else
    skip "oh-my-zsh"
fi

ZSH_CUSTOM="${ZSH_CUSTOM:-$HOME/.oh-my-zsh/custom}"

if [ ! -d "$ZSH_CUSTOM/plugins/zsh-autosuggestions" ]; then
    info "Installing zsh-autosuggestions..."
    git clone https://github.com/zsh-users/zsh-autosuggestions "$ZSH_CUSTOM/plugins/zsh-autosuggestions"
    ok "zsh-autosuggestions"
else
    skip "zsh-autosuggestions"
fi

if [ ! -d "$ZSH_CUSTOM/plugins/zsh-syntax-highlighting" ]; then
    info "Installing zsh-syntax-highlighting..."
    git clone https://github.com/zsh-users/zsh-syntax-highlighting "$ZSH_CUSTOM/plugins/zsh-syntax-highlighting"
    ok "zsh-syntax-highlighting"
else
    skip "zsh-syntax-highlighting"
fi

# ── Bun ──────────────────────────────────────────────────────────────────────

if ! has bun; then
    info "Installing bun..."
    curl -fsSL https://bun.sh/install | bash
    ok "bun"
else
    skip "bun"
fi

# ── pnpm ─────────────────────────────────────────────────────────────────────

if ! has pnpm; then
    info "Installing pnpm..."
    curl -fsSL https://get.pnpm.io/install.sh | sh -
    ok "pnpm"
else
    skip "pnpm"
fi

# ── Claude Code ───────────────────────────────────────────────────────────────

if ! has claude; then
    info "Installing Claude Code..."
    curl -fsSL https://claude.ai/install.sh | bash
    ok "claude"
else
    skip "claude"
fi

# ── Claude Code Plugins ───────────────────────────────────────────────────────

if has claude; then
    info "Installing Claude Code plugins..."
    PLUGINS=(
        frontend-design@claude-plugins-official
        claude-code-setup@claude-plugins-official
        code-review@claude-plugins-official
        feature-dev@claude-plugins-official
        typescript-lsp@claude-plugins-official
        explanatory-output-style@claude-plugins-official
        claude-md-management@claude-plugins-official
    )
    for plugin in "${PLUGINS[@]}"; do
        claude plugin install "$plugin" && ok "$plugin" || warn "Failed to install $plugin"
    done
else
    warn "Claude not installed — skipping plugin install."
fi

# ── Git Submodules ────────────────────────────────────────────────────────────

info "Initializing git submodules..."
git -C "$DOTFILES_DIR" submodule update --init --recursive
ok "submodules"

# ── Secrets ──────────────────────────────────────────────────────────────────

if [ ! -f "$TARGET_DIR/.secrets" ]; then
    info "Creating ~/.secrets from template..."
    cp "$DOTFILES_DIR/.secrets.example" "$TARGET_DIR/.secrets"
    warn "Fill in your values in ~/.secrets before starting a new shell."
else
    skip "~/.secrets"
fi

# ── Stow dotfiles ────────────────────────────────────────────────────────────

info "Stowing dotfiles..."
stow -d "$DOTFILES_DIR" -t "$TARGET_DIR" .
ok "dotfiles"

# ── Done ─────────────────────────────────────────────────────────────────────

echo ""
info "All done!"
echo "  · Fill in ~/.secrets with your credentials"
echo "  · Install Claude Code plugins (see README)"
echo "  · Restart your shell or: source ~/.zshrc"
