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

BACKUP_DIR=""

ensure_backup_dir() {
    if [ -z "$BACKUP_DIR" ]; then
        BACKUP_DIR="$HOME/.dotfiles-backups/$(date +%Y%m%d-%H%M%S)"
        mkdir -p "$BACKUP_DIR"
    fi
}

backup_target() {
    local target="$1" rel dest
    if [ ! -e "$target" ] && [ ! -L "$target" ]; then
        return 0
    fi
    if [ -L "$target" ]; then
        return 0
    fi
    ensure_backup_dir
    rel="${target#$HOME/}"
    dest="$BACKUP_DIR/$rel"
    mkdir -p "$(dirname "$dest")"
    mv "$target" "$dest"
    warn "Moved existing $target to $dest"
}

backup_generated_children() {
    local source_dir="$1" target_dir="$2" child
    if [ ! -d "$source_dir" ] || [ ! -d "$target_dir" ]; then
        return 0
    fi
    for child in "$source_dir"/*; do
        if [ ! -e "$child" ] && [ ! -L "$child" ]; then
            continue
        fi
        backup_target "$target_dir/$(basename "$child")"
    done
}

link_generated_children() {
    local source_dir="$1" target_dir="$2" child name
    if [ ! -d "$source_dir" ]; then
        return 0
    fi
    mkdir -p "$target_dir"
    for child in "$source_dir"/*; do
        if [ ! -e "$child" ] && [ ! -L "$child" ]; then
            continue
        fi
        name="$(basename "$child")"
        rm -rf "$target_dir/$name"
        ln -s "$child" "$target_dir/$name"
    done
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
try_install jq
try_install wget
try_install curl
try_install unzip
try_install ffmpeg

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

# ── Pi ────────────────────────────────────────────────────────────────────────

if ! has pi; then
    info "Installing Pi..."
    if has npm; then
        npm install -g @mariozechner/pi-coding-agent && ok "pi" || warn "Failed to install pi via npm"
    else
        warn "Cannot install Pi — install manually: npm install -g @mariozechner/pi-coding-agent"
    fi
else
    skip "pi"
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
        svelte
    )
    for plugin in "${PLUGINS[@]}"; do
        claude plugin install "$plugin" && ok "$plugin" || warn "Failed to install $plugin"
    done
else
    warn "Claude not installed — skipping plugin install."
fi

# ── Codex ─────────────────────────────────────────────────────────────────────

if ! has codex; then
    info "Installing Codex..."
    if [ "$OS" = "Darwin" ]; then
        brew install --cask codex && ok "codex" || warn "Failed to install codex via brew"
    elif has npm; then
        npm install -g @openai/codex && ok "codex" || warn "Failed to install codex via npm"
    else
        warn "Cannot install Codex — install manually: npm install -g @openai/codex"
    fi
else
    skip "codex"
fi

# ── Firecrawl ─────────────────────────────────────────────────────────────────

if ! has firecrawl; then
    info "Installing Firecrawl CLI..."
    if has npm; then
        npm install -g firecrawl-cli && ok "firecrawl" || warn "Failed to install Firecrawl CLI via npm"
    else
        warn "Cannot install Firecrawl CLI — install manually: npm install -g firecrawl-cli"
    fi
else
    skip "firecrawl"
fi

# ── Agent Skills ──────────────────────────────────────────────────────────────
# Skills are declared in .skills — edit that file to add or remove skill repos.
# Generated skill artifacts stay local to the dotfiles repo and are gitignored.
# ─────────────────────────────────────────────────────────────────────────────

if has npx; then
    info "Installing agent skills..."
    (
        cd "$DOTFILES_DIR"
        while IFS= read -r line; do
            repo="${line%%#*}"
            repo="${repo#"${repo%%[![:space:]]*}"}"
            repo="${repo%"${repo##*[![:space:]]}"}"
            [ -z "$repo" ] && continue
            npx -y skills add "$repo" -y </dev/null && ok "$repo" || warn "Failed to install: $repo"
        done < "$DOTFILES_DIR/.skills"

        mkdir -p "$DOTFILES_DIR/.codex/instructions"
        if [ -d "$DOTFILES_DIR/.agents/skills/uncodixfy" ]; then
            ln -sfn ../../.agents/skills/uncodixfy "$DOTFILES_DIR/.codex/instructions/Uncodixfy"
            ok "Uncodixfy (Codex)"
        fi
    )
else
    warn "npx not found — skipping skills install."
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

# ── Prepare stow targets ─────────────────────────────────────────────────────

info "Preparing stow targets..."
backup_target "$HOME/.claude/CLAUDE.md"
backup_target "$HOME/.claude/settings.json"
backup_target "$HOME/.pi/agent/settings.json"
backup_target "$HOME/.codex/config.toml"
backup_target "$HOME/.codex/rules/default.rules"
backup_generated_children "$DOTFILES_DIR/.claude/skills" "$HOME/.claude/skills"
backup_generated_children "$DOTFILES_DIR/.pi/skills" "$HOME/.pi/skills"
backup_generated_children "$DOTFILES_DIR/.codex/instructions" "$HOME/.codex/instructions"
ok "stow targets"

# ── Stow dotfiles ────────────────────────────────────────────────────────────

info "Stowing dotfiles..."
stow -d "$DOTFILES_DIR" -t "$TARGET_DIR" .
ok "dotfiles"

# ── Link generated skills ────────────────────────────────────────────────────

info "Linking generated skills..."
link_generated_children "$DOTFILES_DIR/.claude/skills" "$HOME/.claude/skills"
link_generated_children "$DOTFILES_DIR/.pi/skills" "$HOME/.pi/skills"
link_generated_children "$DOTFILES_DIR/.codex/instructions" "$HOME/.codex/instructions"
ok "skills"

# ── Done ─────────────────────────────────────────────────────────────────────

echo ""
info "All done!"
echo "  · Fill in ~/.secrets with your credentials"
echo "  · Run 'pi' and '/login' if you want to use your existing Pi subscription"
echo "  · Install Claude Code plugins (see README)"
echo "  · Restart your shell or: source ~/.zshrc"
