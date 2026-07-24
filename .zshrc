# If you come from bash you might have to change your $PATH.
# export PATH=$HOME/bin:/usr/local/bin:$PATH

if command -v nvim >/dev/null 2>&1; then
    export EDITOR="nvim"
    export VISUAL="nvim"
else
    export EDITOR="vi"
    export VISUAL="vi"
fi
export ZSH="$HOME/.oh-my-zsh"

# secrets
[ -f "$HOME/.secrets" ] && source "$HOME/.secrets"

# Homebrew (Linux, macOS Apple Silicon, macOS Intel)
for brew_prefix in /home/linuxbrew/.linuxbrew /opt/homebrew /usr/local; do
    if [[ -x "$brew_prefix/bin/brew" ]]; then
        eval "$("$brew_prefix/bin/brew" shellenv zsh)"
        break
    fi
done

# FNM / Node.js
[ -x "$HOME/.local/share/fnm/fnm" ] && export PATH="$HOME/.local/share/fnm:$PATH"
if command -v fnm >/dev/null 2>&1; then
    eval "$(fnm env --use-on-cd --shell zsh)"
fi

ZSH_THEME="" # disabled in favour of starship

plugins=(
    git
    sudo
    zsh-autosuggestions
    zsh-syntax-highlighting
)
command -v pacman >/dev/null 2>&1 && plugins+=(archlinux)

[ -r "$ZSH/oh-my-zsh.sh" ] && source "$ZSH/oh-my-zsh.sh"

# Check archlinux plugin commands here
# https://github.com/ohmyzsh/ohmyzsh/tree/master/plugins/archlinux

# Display a Pokémon logo when both optional tools are available; otherwise use Fastfetch.
if command -v pokemon-colorscripts >/dev/null 2>&1 \
    && command -v fastfetch >/dev/null 2>&1; then
    pokemon-colorscripts --no-title -s -n "ampharos" \
        | fastfetch -c "$HOME/.config/fastfetch/config-pokemon.jsonc" \
            --logo-type file-raw --logo-height 10 --logo-width 5 --logo -
elif command -v fastfetch >/dev/null 2>&1; then
    fastfetch -c "$HOME/.config/fastfetch/config-compact.jsonc"
fi

# Set-up icons for files/directories in terminal using lsd when available.
if command -v lsd >/dev/null 2>&1; then
    alias ls='lsd'
    alias l='ls -l'
    alias la='ls -a'
    alias lla='ls -la'
    alias lt='ls --tree'
else
    alias l='ls -l'
    alias la='ls -a'
    alias lla='ls -la'
fi

command -v nvim >/dev/null 2>&1 && alias n='nvim'
if command -v bat >/dev/null 2>&1; then
    alias b='bat'
elif command -v batcat >/dev/null 2>&1; then
    alias b='batcat'
fi
command -v claude >/dev/null 2>&1 && alias cc=claude
command -v claude >/dev/null 2>&1 && alias ccsp="claude --dangerously-skip-permissions"
command -v codex >/dev/null 2>&1 && alias cx=codex
function yeet() {
	if [ -z "$1" ]; then
		echo "Usage: yeet \"commit message\""
		return 1
	fi
	git add -A
	echo ""
	git diff --cached --stat
	echo ""
	echo "Commit message: $1"
	echo ""
	read "?Press Enter to yeet, Ctrl+C to abort..."
	git commit -m "$1" && git push
}
function y() {
	local tmp="$(mktemp -t "yazi-cwd.XXXXXX")" cwd
	yazi "$@" --cwd-file="$tmp"
	if cwd="$(command cat -- "$tmp")" && [ -n "$cwd" ] && [ "$cwd" != "$PWD" ]; then
		builtin cd -- "$cwd"
	fi
	rm -f -- "$tmp"
}


# Set-up FZF key bindings (CTRL R for fuzzy history finder)
if command -v fzf >/dev/null 2>&1; then
    source <(fzf --zsh)
fi

HISTFILE=~/.zsh_history
HISTSIZE=10000
SAVEHIST=10000
setopt appendhistory

# More PATH stuff
export PATH="$HOME/.local/bin:$PATH"
export PATH="$HOME/.cargo/bin:$PATH"

# Zoxide Init
if command -v zoxide >/dev/null 2>&1; then
    eval "$(zoxide init zsh)"
fi

# Auto-attach to tmux main session if not already inside tmux or cmux
if command -v tmux >/dev/null 2>&1 \
    && [ -z "$TMUX" ] && [ -z "$CMUX_BUNDLE_ID" ]; then
    tmux new-session -A -s main
fi

# pnpm
export PNPM_HOME="$HOME/.local/share/pnpm"
case ":$PATH:" in
  *":$PNPM_HOME/bin:"*) ;;
  *) export PATH="$PNPM_HOME/bin:$PATH" ;;
esac
# pnpm end

# bun completions
[ -s "$HOME/.bun/_bun" ] && source "$HOME/.bun/_bun"

# bun
export BUN_INSTALL="$HOME/.bun"
export PATH="$BUN_INSTALL/bin:$PATH"
export PATH="$HOME/.local/bin:$PATH"

## [Completion]
## Completion scripts setup. Remove the following line to uninstall
# [[ -f /home/six/.dart-cli-completion/zsh-config.zsh ]] && . /home/six/.dart-cli-completion/zsh-config.zsh || true
## [/Completion]

# ${UserConfigDir}/zsh/.zshrc
autoload -U compinit && compinit
if command -v carapace >/dev/null 2>&1; then
    export CARAPACE_BRIDGES='zsh,fish,bash,inshellisense'
    zstyle ':completion:*' format $'\e[2;37mCompleting %d\e[m'
    source <(carapace _carapace)
fi

# Starship prompt
if command -v starship >/dev/null 2>&1; then
    eval "$(starship init zsh)"
fi

[ -d /opt/homebrew/opt/kleopatra/bin ] && export PATH="/opt/homebrew/opt/kleopatra/bin:$PATH"

# Vite+ bin (https://viteplus.dev)
# Disabled to avoid Vite+ overriding your global node/npm versions.
# Use manually when needed:
#   . "$HOME/.vite-plus/env"
