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

# Display a Pokémon logo when both optional tools and its Fastfetch profile are
# available; otherwise use the compact profile.
pokemon_config="$HOME/.config/fastfetch/config-pokemon.jsonc"
compact_config="$HOME/.config/fastfetch/config-compact.jsonc"
if command -v pokemon-colorscripts >/dev/null 2>&1 \
    && command -v fastfetch >/dev/null 2>&1 \
    && [[ -f "$pokemon_config" ]]; then
    pokemon-colorscripts --no-title -s -n "ampharos" \
        | fastfetch -c "$pokemon_config" \
            --logo-type file-raw --logo-height 10 --logo-width 5 --logo -
elif command -v fastfetch >/dev/null 2>&1 \
    && [[ -f "$compact_config" ]]; then
    fastfetch -c "$compact_config"
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
function cx() {
    if command -v dotfiles >/dev/null 2>&1; then
        dotfiles codex -- "$@"
    else
        codex "$@"
    fi
}
function yeet() {
    local commit_message="$*" commit_body=""

    if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
        echo "yeet: not inside a git repository" >&2
        return 1
    fi

    git add -A || return 1
    if git diff --cached --quiet; then
        echo "yeet: nothing to commit"
        return 1
    fi

    if [ -z "$commit_message" ]; then
        if ! command -v dotfiles >/dev/null 2>&1 || ! command -v codex >/dev/null 2>&1; then
            echo "yeet: dotfiles and codex are required to generate a commit message" >&2
            return 1
        fi
        if ! command -v jq >/dev/null 2>&1; then
            echo "yeet: jq is required to read the generated commit message" >&2
            return 1
        fi

        local model="${YEET_CODEX_MODEL:-gpt-5.6-luna}"
        local reasoning_effort="${YEET_CODEX_REASONING_EFFORT:-low}"
        local branch staged_summary staged_patch prompt temp_dir subject
        branch="$(git symbolic-ref --quiet --short HEAD 2>/dev/null || echo '(detached)')"
        staged_summary="$(git diff --cached --stat)"
        staged_patch="$(git diff --cached --no-ext-diff --binary)"
        prompt="You write concise git commit messages.
Return a JSON object with keys: subject, body.
Rules:
- subject must be imperative, <= 72 chars, and no trailing period
- body can be an empty string or short bullet points
- capture the primary user-visible or developer-visible change

Branch: ${branch}

Staged files:
${staged_summary[1,6000]}

Staged patch:
${staged_patch[1,40000]}"

        temp_dir="$(mktemp -d -t yeet.XXXXXX)" || return 1
        cat > "$temp_dir/schema.json" <<'EOF'
{
  "type": "object",
  "properties": {
    "subject": { "type": "string", "minLength": 1, "maxLength": 72 },
    "body": { "type": "string" }
  },
  "required": ["subject", "body"],
  "additionalProperties": false
}
EOF

        echo "Generating commit message with $model ($reasoning_effort)..."
        if ! dotfiles codex -- --ask-for-approval never exec \
            --ephemeral \
            --sandbox read-only \
            --model "$model" \
            -c "model_reasoning_effort=\"$reasoning_effort\"" \
            --output-schema "$temp_dir/schema.json" \
            --output-last-message "$temp_dir/response.json" \
            --color never \
            "$prompt" >/dev/null; then
            rm -rf -- "$temp_dir"
            echo "yeet: commit message generation failed" >&2
            return 1
        fi

        if ! jq -e 'type == "object" and (.subject | type == "string" and length > 0) and (.body | type == "string")' "$temp_dir/response.json" >/dev/null; then
            rm -rf -- "$temp_dir"
            echo "yeet: Codex returned an invalid commit message" >&2
            return 1
        fi
        subject="$(jq -r '.subject' "$temp_dir/response.json")"
        commit_body="$(jq -r '.body' "$temp_dir/response.json")"
        rm -rf -- "$temp_dir"
        commit_message="$subject"
    fi

    echo ""
    git diff --cached --stat
    echo ""
    echo "Commit message: $commit_message"
    if [ -n "$commit_body" ]; then
        echo ""
        echo "$commit_body"
    fi
    echo ""
    read "?Press Enter to yeet, Ctrl+C to abort..." || return 1

    if [ -n "$commit_body" ]; then
        git commit -m "$commit_message" -m "$commit_body" && git push
    else
        git commit -m "$commit_message" && git push
    fi
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
    if fzf_zsh="$(fzf --zsh 2>/dev/null)"; then
        source <(printf '%s\n' "$fzf_zsh")
    fi
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

# Auto-attach on desktop installs only. Server profiles must keep SSH and
# automation sessions as plain shells even if tmux is installed separately.
_dotfiles_profile_file="${XDG_STATE_HOME:-$HOME/.local/state}/dotfiles/profile"
_dotfiles_profile="${DOTFILES_PROFILE:-$(cat "$_dotfiles_profile_file" 2>/dev/null)}"
if [ "${_dotfiles_profile:-desktop}" = "desktop" ] \
    && [ -z "$SSH_CONNECTION" ] && [ -z "$SSH_TTY" ] \
    && command -v tmux >/dev/null 2>&1 \
    && [ -z "$TMUX" ] && [ -z "$CMUX_BUNDLE_ID" ]; then
    tmux new-session -A -s main
fi
unset _dotfiles_profile _dotfiles_profile_file

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

# Native completions for the dotfiles CLI are loaded after Carapace so they
# supplement its command coverage rather than being replaced by it.
if command -v dotfiles >/dev/null 2>&1; then
    source <(dotfiles completion zsh)
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

# >>> Codex installer >>>
export PATH="/home/ali/.local/bin:$PATH"
# <<< Codex installer <<<

# Pi
export PATH="/home/six/.local/share/fnm/node-versions/v24.18.0/installation/bin:$PATH"
