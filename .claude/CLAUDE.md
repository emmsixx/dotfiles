# Git

- Always use the Conventional Commits specification (https://www.conventionalcommits.org/) for commit messages unless otherwise stated in a project's local CLAUDE.md.
- Do not attribute commits to Claude or mention Claude (no Co-Authored-By, no "Generated with Claude", etc.) unless `.claude`, `CLAUDE.md`, or `AGENTS.md` are tracked in the repository.
- Before committing or pushing, verify `gh auth status` is set to the correct account.
  Account names are in env vars — run `echo $VAR` to resolve them:
  - `~/repos/Work/**` → **$GH_WORK_USER**
  - `~/repos/GitHub/**` → **$GH_PERSONAL_USER**
  - `~/repos/GitLab/**` → **$GH_PERSONAL_USER**
  - If the wrong account is active, run `gh auth switch --user <resolved_username>` first.
