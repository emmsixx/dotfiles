# Pi extensions

This directory contains the selected parts of [davis7dotsh/my-pi-setup](https://github.com/davis7dotsh/my-pi-setup), copied with the author's permission and consolidated for this dotfiles repository.

Included extensions:

- `ask-user`
- `background-terminals`
- `copy-all`
- `file-search`
- `firecrawl-search`
- `git-info`
- `model-info` (supports the custom footer)
- `subagents`
- `summaries`
- `ui-customization`
- `shared` support modules

Intentionally excluded:

- workflows
- bundled themes

Dependencies are consolidated in this directory's `package.json`. Run:

```sh
npm install
npm run check
npm test
```

The Firecrawl tools require `FIRECRAWL_API_KEY` in the environment. Store it in `~/.secrets`, not in this repository.
