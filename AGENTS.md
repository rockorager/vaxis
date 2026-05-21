# Agent instructions

This repository uses `mise` for tool versions and common development tasks.

- Run the full check with `mise run check`.
- Fix Go formatting with `mise run fix`.
- Prefer running Go commands through mise, e.g. `mise exec -- go test ./...`, so the pinned Go version and environment from `mise.toml` are applied.

Use Linux kernel-style commit subjects:

- Format subjects as `subsystem: imperative summary`, e.g. `ui: add profile overlay toggle`.
- Keep subjects concise and lowercase after the subsystem unless a proper noun requires capitalization.
- Use the imperative mood: `add`, `fix`, `remove`, `update`; avoid `added`, `adds`, or gerunds.
