#!/usr/bin/env bash

set -e

DOTFILES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_DIR="$HOME"
PROFILE="${DOTFILES_PROFILE:-desktop}"

usage() {
    cat <<EOF
Usage: $(basename "$0") [--profile desktop|server]

Profiles:
  desktop  Install the full local workstation setup (default)
  server   Install the cross-platform CLI and agent setup without desktop tools
EOF
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --profile)
            [ "$#" -ge 2 ] || { usage >&2; exit 1; }
            PROFILE="$2"
            shift 2
            ;;
        --server)
            PROFILE="server"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            printf 'Unknown argument: %s\n\n' "$1" >&2
            usage >&2
            exit 1
            ;;
    esac
done

case "$PROFILE" in
    desktop|server) ;;
    *)
        printf 'Unknown profile: %s\n\n' "$PROFILE" >&2
        usage >&2
        exit 1
        ;;
esac

has()  { command -v "$1" &>/dev/null; }
info() { printf '\033[1;34m==> %s\033[0m\n' "$*"; }
ok()   { printf '\033[1;32m ✓ %s\033[0m\n' "$*"; }
skip() { printf '\033[2m · %s already installed\033[0m\n' "$*"; }
warn() { printf '\033[1;33m ! %s\033[0m\n' "$*"; }

PROFILE_FILE="${XDG_STATE_HOME:-$HOME/.local/state}/dotfiles/profile"

# ── OS & Package Manager ──────────────────────────────────────────────────────

OS="$(uname -s)"
PACKAGE_MANAGER="none"

run_as_root() {
    if [ "$(id -u)" -eq 0 ]; then
        "$@"
    else
        sudo "$@"
    fi
}

if [ "$OS" = "Darwin" ]; then
    if ! has brew; then
        info "Installing Homebrew..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    fi

    for brew_prefix in /home/linuxbrew/.linuxbrew /opt/homebrew /usr/local; do
        if [ -x "$brew_prefix/bin/brew" ]; then
            export PATH="$brew_prefix/bin:$PATH"
            eval "$("$brew_prefix/bin/brew" shellenv bash)"
            break
        fi
    done

    if has brew; then
        PACKAGE_MANAGER="brew"
    else
        warn "Homebrew is unavailable — install packages manually."
    fi
elif has pacman; then
    PACKAGE_MANAGER="pacman"
elif has apt-get; then
    PACKAGE_MANAGER="apt"
else
    warn "Unknown package manager — install packages manually."
fi

if [ "$PACKAGE_MANAGER" = "apt" ]; then
    info "Refreshing APT package lists..."
    if run_as_root apt-get update; then
        ok "APT package lists"
    else
        warn "APT package list refresh failed; package installation may be incomplete."
    fi
fi

package_available() {
    local package="$1"
    case "$PACKAGE_MANAGER" in
        brew) brew info --formula "$package" &>/dev/null ;;
        pacman) pacman -Si "$package" &>/dev/null ;;
        apt) apt-cache show "$package" &>/dev/null ;;
        *) return 1 ;;
    esac
}

pkg_install() {
    case "$PACKAGE_MANAGER" in
        brew) brew install "$@" ;;
        pacman) run_as_root pacman -S --needed --noconfirm "$@" ;;
        apt) run_as_root apt-get install -y "$@" ;;
        *)
            warn "Cannot auto-install: $*"
            return 1
            ;;
    esac
}

try_install() {
    local binary="$1" package="${2:-$1}" alternate="${3:-}"

    if has "$binary" || { [ -n "$alternate" ] && has "$alternate"; }; then
        skip "$binary"
        return 0
    fi

    if ! package_available "$package"; then
        warn "No $package package is available through $PACKAGE_MANAGER — install $binary manually."
        return 0
    fi

    info "Installing $binary..."
    if pkg_install "$package"; then
        ok "$binary"
    else
        warn "Failed to install $binary"
    fi
}

ensure_command_alias() {
    local expected="$1" actual="$2"

    if has "$expected" || ! has "$actual"; then
        return 0
    fi

    mkdir -p "$HOME/.local/bin"
    ln -sf "$(command -v "$actual")" "$HOME/.local/bin/$expected"
    export PATH="$HOME/.local/bin:$PATH"
    ok "$expected command alias"
}

try_cask_install() {
    local binary="$1" package="${2:-$1}"

    if has "$binary"; then
        skip "$binary"
    elif [ "$PACKAGE_MANAGER" = "brew" ] && brew info --cask "$package" &>/dev/null; then
        info "Installing $binary..."
        brew install --cask "$package" && ok "$binary" || warn "Failed to install $binary"
    else
        warn "No installable $package cask is available — install $binary manually."
    fi
}

try_aur_install() {
    local binary="$1" package="${2:-$1}"

    if has "$binary"; then
        skip "$binary"
    elif has yay; then
        info "Installing $binary from the AUR..."
        yay -S --noconfirm "$package" && ok "$binary" || warn "Failed to install $binary"
    elif has paru; then
        info "Installing $binary from the AUR..."
        paru -S --noconfirm "$package" && ok "$binary" || warn "Failed to install $binary"
    else
        warn "No AUR helper available — install $binary manually."
    fi
}

require_tools() {
    local missing=() tool

    for tool in "$@"; do
        if ! has "$tool"; then
            missing+=("$tool")
        fi
    done

    if [ "${#missing[@]}" -gt 0 ]; then
        warn "Required tools are missing: ${missing[*]}"
        return 1
    fi
}

BACKUP_DIR=""
BACKED_UP_TARGETS=()
BACKUP_PATHS=()

ensure_backup_dir() {
    if [ -z "$BACKUP_DIR" ]; then
        BACKUP_DIR="$HOME/.dotfiles-backups/$(date +%Y%m%d-%H%M%S)"
        mkdir -p "$BACKUP_DIR"
    fi
}

backup_target() {
    local target="$1" force="${2:-}" rel dest suffix
    if [ ! -e "$target" ] && [ ! -L "$target" ]; then
        return 0
    fi
    if [ -L "$target" ] && [ "$force" != "force" ]; then
        return 0
    fi

    ensure_backup_dir
    rel="${target#$HOME/}"
    dest="$BACKUP_DIR/$rel"
    if [ -e "$dest" ] || [ -L "$dest" ]; then
        suffix=1
        while [ -e "$dest.$suffix" ] || [ -L "$dest.$suffix" ]; do
            suffix=$((suffix + 1))
        done
        dest="$dest.$suffix"
    fi
    mkdir -p "$(dirname "$dest")"
    mv "$target" "$dest"
    BACKED_UP_TARGETS+=("$target")
    BACKUP_PATHS+=("$dest")
    warn "Moved existing $target to $dest"
}

restore_stow_targets() {
    local index target backup

    set +e
    for ((index=${#BACKED_UP_TARGETS[@]} - 1; index >= 0; index--)); do
        target="${BACKED_UP_TARGETS[$index]}"
        backup="${BACKUP_PATHS[$index]}"
        rm -rf "$target"
        mkdir -p "$(dirname "$target")"
        mv "$backup" "$target"
        warn "Restored $target after setup failure"
    done
}

rollback_pending_stow() {
    local status=$?
    trap - ERR
    if [ "${STOW_PENDING:-0}" -eq 1 ]; then
        restore_stow_targets
    fi
    exit "$status"
}

has_symlink_ancestor() {
    local path="$1" parent
    parent="$(dirname "$path")"
    while [ "$parent" != "$HOME" ] && [ "$parent" != "/" ]; do
        if [ -L "$parent" ]; then
            return 0
        fi
        parent="$(dirname "$parent")"
    done
    return 1
}

prepare_stow_targets() {
    local source target source_dir

    # Unlink existing managed directories before inspecting their children.
    for source_dir in "$DOTFILES_DIR/.config" "$DOTFILES_DIR/.config"/* "$DOTFILES_DIR/.claude" "$DOTFILES_DIR/.pi" "$DOTFILES_DIR/.scripts"; do
        source="${source_dir#$DOTFILES_DIR/}"
        target="$HOME/$source"
        if [ -L "$target" ]; then
            backup_target "$target" force
        fi
    done

    # These are ignored by Stow and should never be moved into the backup.
    while IFS= read -r source; do
        case "$source" in
            README.md|LICENSE|setup.sh|setup-server.sh|.secrets.example|.gitmodules|.skills|.node-version|skills-lock.json|.stow-local-ignore)
                continue
                ;;
        esac

        target="$HOME/$source"
        if [ -e "$DOTFILES_DIR/$source" ] || [ -L "$DOTFILES_DIR/$source" ]; then
            if ! has_symlink_ancestor "$target"; then
                backup_target "$target" force
            fi
        fi
    done < <(git -C "$DOTFILES_DIR" ls-files)
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
    local source_dir="$1" target_dir="$2" child name existing link_target
    if [ ! -d "$source_dir" ]; then
        return 0
    fi

    mkdir -p "$target_dir"

    # Remove stale links previously generated from this source directory.
    for existing in "$target_dir"/*; do
        if [ ! -L "$existing" ]; then
            continue
        fi
        link_target="$(readlink "$existing")"
        case "$link_target" in
            "$source_dir"/*)
                name="$(basename "$existing")"
                if [ ! -e "$source_dir/$name" ] && [ ! -L "$source_dir/$name" ]; then
                    rm "$existing"
                    warn "Removed stale generated link $existing"
                fi
                ;;
        esac
    done

    for child in "$source_dir"/*; do
        if [ ! -e "$child" ] && [ ! -L "$child" ]; then
            continue
        fi
        name="$(basename "$child")"
        if [ -L "$target_dir/$name" ]; then
            link_target="$(readlink "$target_dir/$name")"
            case "$link_target" in
                "$source_dir"/*) rm -f "$target_dir/$name" ;;
                *)
                    warn "Preserving unrelated symlink $target_dir/$name"
                    continue
                    ;;
            esac
        elif [ -e "$target_dir/$name" ]; then
            backup_target "$target_dir/$name"
        fi
        ln -s "$child" "$target_dir/$name"
    done
}

skill_repo_from_line() {
    local entry="$1" repo

    entry="${entry%%#*}"
    repo="${entry%%|*}"
    repo="${repo#"${repo%%[![:space:]]*}"}"
    repo="${repo%"${repo##*[![:space:]]}"}"
    printf '%s' "$repo"
}

prune_skill_lock() {
    local sources_json selected_json full_sources_json repo entry selections skill

    if [ ! -f "$DOTFILES_DIR/skills-lock.json" ] || ! has jq; then
        return 0
    fi

    sources_json="$(
        while IFS= read -r line; do
            repo="$(skill_repo_from_line "$line")"
            [ -n "$repo" ] && printf '%s\n' "$repo"
        done < "$DOTFILES_DIR/.skills" \
            | jq -R -s 'split("\n") | map(select(length > 0))'
    )"

    selected_json="$(
        while IFS= read -r line; do
            entry="${line%%#*}"
            entry="${entry#"${entry%%[![:space:]]*}"}"
            entry="${entry%"${entry##*[![:space:]]}"}"
            [[ "$entry" == *"|"* ]] || continue
            selections="${entry#*|}"
            IFS=',' read -r -a selected_skills <<< "$selections"
            for skill in "${selected_skills[@]}"; do
                skill="${skill#"${skill%%[![:space:]]*}"}"
                skill="${skill%"${skill##*[![:space:]]}"}"
                [ -n "$skill" ] && printf '%s\n' "$skill"
            done
        done < "$DOTFILES_DIR/.skills" \
            | jq -R -s 'split("\n") | map(select(length > 0))'
    )"

    full_sources_json="$(
        while IFS= read -r line; do
            entry="${line%%#*}"
            entry="${entry#"${entry%%[![:space:]]*}"}"
            entry="${entry%"${entry##*[![:space:]]}"}"
            [[ -n "$entry" && "$entry" != *"|"* ]] || continue
            repo="$(skill_repo_from_line "$entry")"
            [ -n "$repo" ] && printf '%s\n' "$repo"
        done < "$DOTFILES_DIR/.skills" \
            | jq -R -s 'split("\n") | map(select(length > 0))'
    )"

    jq --argjson sources "$sources_json" \
        --argjson selected "$selected_json" \
        --argjson full_sources "$full_sources_json" \
        '.skills |= with_entries(select(
            .key as $name |
            .value.source as $source |
            (($sources | index($source)) != null) and
            ((($full_sources | index($source)) != null) or (($selected | index($name)) != null))
        ))' \
        "$DOTFILES_DIR/skills-lock.json" > "$DOTFILES_DIR/skills-lock.json.tmp"
    mv "$DOTFILES_DIR/skills-lock.json.tmp" "$DOTFILES_DIR/skills-lock.json"
}

prune_skill_artifacts() {
    local skill_dir name child

    prune_skill_lock

    if [ ! -d "$DOTFILES_DIR/.agents/skills" ]; then
        return 0
    fi

    if [ -f "$DOTFILES_DIR/skills-lock.json" ] && has jq; then
        for skill_dir in "$DOTFILES_DIR/.agents/skills"/*; do
            if [ ! -d "$skill_dir" ]; then
                continue
            fi
            name="$(basename "$skill_dir")"
            if ! jq -e --arg name "$name" '.skills[$name]' "$DOTFILES_DIR/skills-lock.json" &>/dev/null; then
                rm -rf "$skill_dir"
                warn "Removed stale skill $name"
            fi
        done
    fi

    for child in "$DOTFILES_DIR/.claude/skills"/* "$DOTFILES_DIR/.pi/skills"/*; do
        if [ -L "$child" ] && [ ! -e "$child" ]; then
            rm "$child"
        fi
    done
}

# ── Core Tools ────────────────────────────────────────────────────────────────

info "Checking core tools for the $PROFILE profile..."
try_install curl
try_install git
try_install stow
try_install zsh
try_install jq
try_install wget
try_install unzip
try_install gh
try_install nvim neovim
if [ "$PACKAGE_MANAGER" = "brew" ]; then
    try_install tree-sitter tree-sitter
else
    try_install tree-sitter tree-sitter-cli
fi
try_install bat bat batcat
try_install lsd
try_install fzf
try_install zoxide
if [ "$PROFILE" = "desktop" ]; then
    try_install tmux
else
    info "Skipping tmux for the server profile."
fi
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
if [ "$PACKAGE_MANAGER" = "apt" ]; then
    try_install fd fd-find fdfind
    ensure_command_alias fd fdfind
else
    try_install fd
fi
try_install starship
try_install ffmpeg

if ! require_tools curl git stow zsh jq; then
    exit 1
fi

if [ "$PROFILE" = "desktop" ]; then
    if [ "$OS" = "Darwin" ]; then
        try_cask_install ghostty
    elif [ "$OS" = "Linux" ]; then
        info "Checking Linux Wayland desktop tools..."
        try_install grim
        try_install slurp
        try_install satty
        try_install solaar
        try_install wl-copy wl-clipboard
        if [ "$PACKAGE_MANAGER" = "apt" ]; then
            try_install notify-send libnotify-bin
        else
            try_install notify-send libnotify
        fi
        try_install file
        if [ "$PACKAGE_MANAGER" = "pacman" ]; then
            try_aur_install dualsensectl
        else
            info "Skipping dualsensectl outside Arch Linux."
        fi
    fi
fi

# ── pokemon-colorscripts ──────────────────────────────────────────────────────

if [ "$PROFILE" = "desktop" ]; then
    if ! has pokemon-colorscripts; then
        info "Installing pokemon-colorscripts..."
        if [ "$OS" = "Darwin" ]; then
            brew install --no-quarantine phisch/tap/pokemon-colorscripts \
                && ok "pokemon-colorscripts" \
                || warn "Failed to install pokemon-colorscripts"
        elif has yay; then
            yay -S --noconfirm pokemon-colorscripts-git \
                && ok "pokemon-colorscripts" \
                || warn "Failed to install pokemon-colorscripts"
        elif has paru; then
            paru -S --noconfirm pokemon-colorscripts-git \
                && ok "pokemon-colorscripts" \
                || warn "Failed to install pokemon-colorscripts"
        else
            warn "Install pokemon-colorscripts manually: https://gitlab.com/phoneybadger/pokemon-colorscripts"
        fi
    else
        skip "pokemon-colorscripts"
    fi
else
    info "Skipping pokemon-colorscripts for the server profile."
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

# ── FNM / Node.js ─────────────────────────────────────────────────────────────

NODE_VERSION_FILE="$DOTFILES_DIR/.node-version"

install_fnm() {
    if has fnm; then
        skip "fnm"
        return 0
    fi

    info "Installing fnm..."
    if [ "$PACKAGE_MANAGER" = "brew" ] && pkg_install fnm; then
        ok "fnm"
    elif [ "$PACKAGE_MANAGER" = "pacman" ] && package_available fnm && pkg_install fnm; then
        ok "fnm"
    elif curl -fsSL https://fnm.vercel.app/install | bash -s -- --skip-shell; then
        ok "fnm"
    else
        warn "Failed to install fnm"
        return 0
    fi

    if [ -x "$HOME/.local/share/fnm/fnm" ]; then
        export PATH="$HOME/.local/share/fnm:$PATH"
    fi
}

check_node_lts_update() {
    local current_version current_major latest_lts latest_major

    if ! has node || ! has jq || ! has curl; then
        return 0
    fi

    current_version="$(node --version 2>/dev/null || true)"
    current_major="${current_version#v}"
    current_major="${current_major%%.*}"
    latest_lts="$(curl -fsSL https://nodejs.org/dist/index.json 2>/dev/null \
        | jq -r '[.[] | select(.lts != false)][0].version' 2>/dev/null || true)"
    latest_major="${latest_lts#v}"
    latest_major="${latest_major%%.*}"

    if [[ "$current_major" =~ ^[0-9]+$ ]] \
        && [[ "$latest_major" =~ ^[0-9]+$ ]] \
        && [ "$latest_major" -gt "$current_major" ]; then
        warn "Node.js LTS $latest_lts is available (current: $current_version). Update $NODE_VERSION_FILE when ready."
    fi
}

install_fnm
if has fnm; then
    eval "$(fnm env --shell bash)"

    if [ ! -f "$NODE_VERSION_FILE" ]; then
        info "Installing the latest Node.js LTS..."
        if fnm install --lts --use; then
            NODE_VERSION="$(node --version)"
            printf '%s\n' "$NODE_VERSION" > "$NODE_VERSION_FILE"
            fnm default "$NODE_VERSION"
            ok "Node.js $NODE_VERSION (pinned in .node-version)"
            check_node_lts_update
        else
            warn "Failed to install Node.js LTS"
        fi
    else
        NODE_VERSION="$(tr -d '[:space:]' < "$NODE_VERSION_FILE")"
        if [ -n "$NODE_VERSION" ] && fnm install "$NODE_VERSION" --use; then
            fnm default "$NODE_VERSION"
            ok "Node.js $NODE_VERSION"
            check_node_lts_update
        else
            warn "Failed to activate Node.js from $NODE_VERSION_FILE"
        fi
    fi
else
    warn "fnm unavailable — npm-based tools may need manual installation."
fi

# ── Pi Extension Dependencies ────────────────────────────────────────────────

PI_EXTENSIONS_DIR="$DOTFILES_DIR/.pi/agent/extensions"
if [ -f "$PI_EXTENSIONS_DIR/package-lock.json" ]; then
    if has npm; then
        info "Installing Pi extension dependencies..."
        npm --prefix "$PI_EXTENSIONS_DIR" ci --omit=dev \
            && ok "Pi extension dependencies" \
            || warn "Failed to install Pi extension dependencies"
    else
        warn "npm unavailable — Pi extensions may need manual dependency installation."
    fi
fi

# ── Claude Code ───────────────────────────────────────────────────────────────

info "Installing or refreshing Claude Code..."
if (set -o pipefail; curl -fsSL https://claude.ai/install.sh | bash); then
    ok "claude"
else
    warn "Failed to install Claude Code with the official installer"
fi

# ── Pi ────────────────────────────────────────────────────────────────────────

info "Installing or refreshing Pi..."
if (set -o pipefail; curl -fsSL https://pi.dev/install.sh | sh); then
    ok "pi"
else
    warn "Failed to install Pi with the official installer"
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

info "Installing or refreshing Codex..."
if (set -o pipefail; curl -fsSL https://chatgpt.com/codex/install.sh | sh); then
    ok "codex"
else
    warn "Failed to install Codex with the official installer"
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
# Append |skill1,skill2 to install only selected skills from a repo bundle.
# Generated skill artifacts stay local to the dotfiles repo and are gitignored.
# ─────────────────────────────────────────────────────────────────────────────

if has npx; then
    info "Installing agent skills..."
    (
        cd "$DOTFILES_DIR"
        while IFS= read -r line; do
            entry="${line%%#*}"
            entry="${entry#"${entry%%[![:space:]]*}"}"
            entry="${entry%"${entry##*[![:space:]]}"}"
            [ -z "$entry" ] && continue

            repo="$(skill_repo_from_line "$entry")"
            if [[ "$entry" == *"|"* ]]; then
                selections="${entry#*|}"
                IFS=',' read -r -a selected_skills <<< "$selections"
                for skill in "${selected_skills[@]}"; do
                    skill="${skill#"${skill%%[![:space:]]*}"}"
                    skill="${skill%"${skill##*[![:space:]]}"}"
                    [ -z "$skill" ] && continue
                    npx -y skills add "$repo" --skill "$skill" -y </dev/null \
                        && ok "$repo@$skill" || warn "Failed to install: $repo@$skill"
                done
            else
                npx -y skills add "$repo" -y </dev/null && ok "$repo" || warn "Failed to install: $repo"
            fi
        done < "$DOTFILES_DIR/.skills"
    )
else
    warn "npx not found — skipping skills installation."
fi

# Pruning only needs the tracked declarations and jq; it must still run when
# Node.js setup fails so removed skills do not leave stale local artifacts.
prune_skill_artifacts

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
# Delay destructive moves until every fallible installer has completed, and
# roll them back if preparation or Stow itself fails.
STOW_PENDING=1
trap rollback_pending_stow ERR
prepare_stow_targets
backup_generated_children "$DOTFILES_DIR/.claude/skills" "$HOME/.claude/skills"
backup_generated_children "$DOTFILES_DIR/.pi/skills" "$HOME/.pi/skills"
ok "stow targets"

# ── Stow dotfiles ────────────────────────────────────────────────────────────

info "Stowing dotfiles..."
stow -d "$DOTFILES_DIR" -t "$TARGET_DIR" .
STOW_PENDING=0
trap - ERR
ok "dotfiles"

# Persist the profile only after setup succeeds far enough to install configs.
# .zshrc uses this to avoid desktop-only behavior on headless machines.
mkdir -p "$(dirname "$PROFILE_FILE")"
printf '%s\n' "$PROFILE" > "$PROFILE_FILE"

# ── Link generated skills ────────────────────────────────────────────────────

info "Linking generated skills..."
link_generated_children "$DOTFILES_DIR/.claude/skills" "$HOME/.claude/skills"
link_generated_children "$DOTFILES_DIR/.pi/skills" "$HOME/.pi/skills"
ok "skills"

# ── Done ─────────────────────────────────────────────────────────────────────

echo ""
info "All done!"
echo "  · Fill in ~/.secrets with your credentials"
echo "  · Run 'pi' and '/login' if you want to use your existing Pi subscription"
echo "  · Restart your shell or: source ~/.zshrc"
