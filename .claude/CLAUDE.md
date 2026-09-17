# Git

- Always use the Conventional Commits specification (https://www.conventionalcommits.org/) for commit messages unless otherwise stated in a project's local CLAUDE.md.
- Do not attribute commits to Claude or mention Claude (no Co-Authored-By, no "Generated with Claude", etc.) unless `.claude`, `CLAUDE.md`, or `AGENTS.md` are tracked in the repository.
- Before committing or pushing, verify `gh auth status` is set to the correct account.
  Account names are in env vars — run `echo $VAR` to resolve them:
  - `~/repos/Work/**` → **$GH_WORK_USER**
  - `~/repos/GitHub/**` → **$GH_PERSONAL_USER**
  - `~/repos/GitLab/**` → **$GH_PERSONAL_USER**
  - If the wrong account is active, run `gh auth switch --user <resolved_username>` first.

# Web development

These are my defaults unless I say otherwise. Preserve an existing project's stack unless I ask to change it.

- When blueprinting a website, start with five distinct designs at `/1`, `/2`, `/3`, `/4`, and `/5`, with a small switcher on every version. Explore different directions for the same brief, not just different colors.
- No editorial style.
- Use the skills and tools from [Impeccable](https://impeccable.style/) for frontend and UI work.
- Use Astro for static or content-heavy sites such as portfolios, blogs, and marketing sites.
- Use TanStack Start for full apps or complex state, auth, or data fetching; do not default to Next.js.
- Use Tailwind CSS, TypeScript, and Bun.
- Deploy to Cloudflare by default, or Vercel when selected. Keep deployment configuration portable.
- When a backend is needed, use Hono for CRUD, API, or auth-oriented apps where I want ownership and portability. Use Convex for realtime or collaborative features, or fast delivery without managing infrastructure.
