# photo-select

Swipe-to-select photo culling: runs on the computer holding a big photo library and serves a swipe UI to phones on the same Wi-Fi (often mirrored to a TV) so people can pick ~500 keepers out of ~25,000.

## Stack
- Go 1.22+, standard library only (no third-party modules). Single binary with the web UI embedded (`//go:embed web`).
- Frontend: plain HTML/CSS/JS in `web/`, no build step.
- No database: progress is an append-only JSONL log in `<output>/.photo-select/decisions.jsonl`.
- No LLM calls.

## Commands
- Dev: `go run .` (opens http://localhost:8787; `-no-browser`, `-port N`)
- Test: `go test -race ./...`
- Lint/typecheck: `gofmt -l . && go vet ./...`
- Release builds: `scripts/build.sh vX.Y.Z` → `dist/*.zip` (CI does this on `v*` tags)

## Structure
- `main.go`: flags, server start, prints phone URLs, opens the browser
- `app.go`: project config, scan orchestration, decide/undo, queue order, state for the UI
- `session.go`: the decision log (replay, undo lines, torn-line tolerance)
- `library.go`: folder scan + natural sort; `fileops.go`: copy/move/unique names/path cleanup
- `meta.go`: EXIF capture date + duplicate fingerprints (cached in `.photo-select/meta.json`), dedupe of identical files and RAW+JPEG pairs
- `preview.go`: macOS `sips` previews (HEIC/RAW/large JPEG → 2048px JPEG), cached in the user cache dir
- `server.go`: HTTP routes; `web/`: swipe page (`index.html`, `app.js`), Mac setup page (`setup.*`)
- `site/`: friends homepage (React + Vite, separate from the Go app) with `install.sh`/`install.ps1`; `scripts/build-site.sh` bundles the app zips into it; deployed to Azure Static Web Apps by `.github/workflows/site.yml` (docs/AZURE.md)

## Conventions
- Photos are identified by slash-separated path relative to the source folder.
- File op first, then append to the log; undo reverses the file op, then appends `undo`.
- Settings endpoints (`/api/setup`, `/api/pick-folder`) are loopback-only; phones can only swipe.
- Never modify or delete originals except in the user-chosen "move" modes.

## Gotchas
- macOS filesystems are case-insensitive; Linux CI is not — name-collision tests must use identical case.
- exFAT drives contain `._*` AppleDouble files; the scanner skips hidden files for this reason.
- In move mode, moved photos vanish from the source scan; `App.scan` re-adds them from the log.
- `sips` only exists on macOS; elsewhere originals are served as-is.
- File mtimes on copied drives are meaningless; date order uses EXIF DateTimeOriginal.
- Picks/rejects keep their source sub-folder under `Selected/` / `Rejected/`.

## Out of bounds
- Don't touch a user's real photo folders when testing; generate fixtures in temp dirs.
