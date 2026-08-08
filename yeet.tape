Output yeet.gif

Require git
Require go
Require codex
Require jq

Set Shell "zsh"
Set FontFamily "Google Sans Code NF"
Set FontSize 26
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set Padding 24
Set Margin 24
Set MarginFill "#11111b"
Set BorderRadius 12
Set WindowBar "ColorfulRight"
Set TypingSpeed 1ms
Set Framerate 30

Hide
Type "source /Users/ali/dotfiles/.zshrc"
Enter
Type "export PROMPT='%F{cyan}❯%f ' RPROMPT=''"
Enter
Type "unset DOTFILES_CODEX_ACCOUNT"
Enter
Type "rm -rf /tmp/yeet-vhs-demo && mkdir -p /tmp/yeet-vhs-demo/bin"
Enter
Type "(cd /Users/ali/dotfiles && go build -o /tmp/yeet-vhs-demo/bin/dotfiles ./cmd/dotfiles)"
Enter
Type "export PATH=/tmp/yeet-vhs-demo/bin:$PATH"
Enter
Type "git init --bare -q /tmp/yeet-vhs-demo/remote.git"
Enter
Type "git clone -q /tmp/yeet-vhs-demo/remote.git /tmp/yeet-vhs-demo/repo"
Enter
Type "cd /tmp/yeet-vhs-demo/repo"
Enter
Type "git config user.name 'VHS Demo' && git config user.email 'vhs@example.com'"
Enter
Type "printf '# Yeet\n\nA tiny git helper.\n' > README.md"
Enter
Type "git add README.md && git commit -qm 'chore: initialize demo' && git push -qu origin HEAD"
Enter
Type "printf '\nGenerate Conventional Commits messages automatically with Codex.\n' >> README.md"
Enter
Type "clear"
Enter
Sleep 2s
Show
Sleep 1s
Set TypingSpeed 35ms

Type "git diff --stat"
Enter
Sleep 2s

Type "yeet"
Enter
Sleep 2s
Down
Enter
Sleep 12s
Enter
Sleep 6s

Type "git log -1 --oneline"
Enter
Sleep 4s
