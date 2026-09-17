# Agent Guidelines

## TypeScript

- Add packages with the package manager's install command instead of editing `package.json` manually.
- Run the project's check, format, lint, and test commands after making changes. If they do not exist, suggest adding them.
- Avoid explicit return types unless they are necessary; prefer type inference.
- Treat `as any` as an absolute last resort. Preserve real type safety and avoid duplicating inferred types.

## Svelte and SvelteKit

- Use modern Svelte practices and load the available Svelte best-practices skills before editing Svelte components or modules.

## Communication

- When clarification is needed, ask one question at a time.

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
