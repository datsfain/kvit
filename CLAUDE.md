# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**kvit** is a terminal-based expense tracker CLI written in Go. It tracks daily spending in plain CSV files, supports interactive and inline entry modes, generates HTML reports with charts, and syncs data to Google Drive for family sharing.

## Build & Run Commands

```bash
go build                    # Build the kvit binary
go run main.go              # Run without building
go test ./...               # Run all tests
goreleaser release --clean  # Build release artifacts
```

Manual testing (no test suite exists yet):
```bash
./kvit add netto ground-beef:39.95 milk:12.50
./kvit i                    # Interactive mode
./kvit summary              # Generate HTML report
./kvit sync push            # Upload to Google Drive
./kvit auth                 # Google OAuth2 sign-in
```

## Architecture

### Module Structure

- **cmd/** — Cobra CLI commands. Each file is one command (add, interactive, sync, summary, auth, exclude, prompt, update). `common.go` has shared TUI components (confirm dialog, categorization flow).
- **drive/** — Google Drive integration: OAuth2 with PKCE, push/pull sync, browser helpers, progress display.
- **storage/** — CSV file I/O for expenses and product definitions. Append-only operations.
- **models/** — Data types: `Expense`, `Definition`, `StoreEntry`. Each has a `CSVRow()` method.
- **config/** — File path constants and the list of syncable files.
- **report/** — HTML report generation. Uses `go:embed` to bundle `template.html`, `style.css`, `app.js`, and `prompt.txt`. Template uses `{{CSS}}`, `{{JS}}`, `{{DATA}}` placeholders replaced at generation time.

### User Config

Stored at `~/.config/kvit/config.json`. Managed by `config/config.go` with `KvitConfig` struct containing `FolderID`, `Currency`, and `Languages`. First-run setup (`cmd/setup.go`) triggers via `PersistentPreRun` in `root.go` when currency is not set. `kvit config` shows current settings, `kvit config --setup` re-runs setup.

Currency and languages are used throughout the CLI display, HTML report (via `DATA.currency` in JS), and AI prompt template (via `{{CURRENCY}}` and `{{LANGUAGES}}` placeholders).

### Data Flow

1. **Entry:** User input → parsed into `StoreEntry` → confirm TUI → `storage.AppendExpenses()` writes to `expenses.csv` → unknown products trigger categorization TUI → `storage.AppendDefinitions()` writes to `definitions.csv`
2. **Sync:** OAuth2+PKCE auth → token stored at `~/.config/kvit/token.json` → parallel upload/download of CSV files to/from a linked Google Drive folder
3. **Report:** Load CSVs → serialize to JSON → inject into HTML template with embedded CSS/JS → write `kvit-report.html` → open in browser

### Data Storage

All data is plain CSV in the working directory:
- `expenses.csv` — `date,store,product,price`
- `definitions.csv` — `product,category`
- `exclusions.csv` — products hidden from AI prompt
- `colors.csv` — category hex colors for reports

### TUI Pattern

Interactive components use Charmbracelet Bubbletea. Models implement `Init()`, `Update(msg)`, `View()`. Styling via Lipgloss.

### Release Pipeline

GitHub Actions (`.github/workflows/release.yml`): on tag push → GoReleaser builds for Linux/macOS (amd64/arm64) → cosign signs checksums → publishes GitHub release. Version injected via ldflags `-X kvit/cmd.Version`.
