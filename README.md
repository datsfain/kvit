# kvit

A fast CLI expense tracker built in Go. Track daily grocery expenses in DKK, analyze spending, and use AI (Gemini/ChatGPT) to scan receipts automatically.

Data is stored as plain CSV files in your working directory — sync them to Google Drive and analyze with Google Sheets, Grafana, or any tool you like.

## Install

```bash
curl -sSL https://raw.githubusercontent.com/datsfain/kvit/main/install.sh | bash
```

Or download a binary manually from [Releases](https://github.com/datsfain/kvit/releases).

**Linux clipboard support** (needed for `kvit prompt`):
```bash
# X11
sudo apt install xclip
# Wayland
sudo apt install wl-clipboard
```

## Quick start

```bash
mkdir ~/expenses && cd ~/expenses

# Add your first expenses
kvit add netto ground-beef:39.95 milk:12.50 bread:22

# Or use the interactive mode
kvit i
```

## How it works with AI

1. Run `kvit prompt` — copies a tailored AI prompt to your clipboard
2. Open [Gemini](https://gemini.google.com) or ChatGPT
3. Paste the prompt and attach a photo of your receipt
4. The AI reads the receipt (even in Danish) and generates a `kvit add` command
5. Paste the command into your terminal — done

The prompt includes your known product names so the AI reuses them consistently.

## Commands

### `kvit add` — Add expenses (one-liner)

```bash
kvit add netto ground-beef:200 cucumber:30              # today's date
kvit add netto 2026-04-05 ground-beef:200 cucumber:30   # specific date
kvit add netto ground-beef:200 + føtex milk:12 bread:25 # multiple stores
```

Shows a confirmation dialog (items sorted by price) before saving.

### `kvit interactive` (or `kvit i`) — Add expenses interactively

Guided TUI with:
- Date picker — up/down arrows to switch days, shows "Today, Monday" etc.
- Store autocomplete — Tab to accept, shows matching stores in gray
- Product autocomplete — Tab to accept from known products
- Confirmation dialog before saving
- Prompts you to categorize any new products

### `kvit prompt` — Generate AI receipt-scanning prompt

Copies a prompt to your clipboard that includes your known products and stores. The AI will match Danish receipt items to your existing product names and generate a ready-to-run command.

### `kvit exclude` — Manage prompt exclusions

Control which products are shared with the AI.

```bash
kvit exclude add shoes          # hide from AI prompt
kvit exclude remove shoes       # show again
kvit exclude list               # see all exclusions
```

### `kvit sync` — Sync with Google Drive

```bash
kvit sync push   # upload local CSVs to Google Drive (overwrites remote)
kvit sync pull   # download from Google Drive to local (overwrites local)
```

Requires [rclone](https://rclone.org). See [Google Drive setup](#google-drive-setup) below.

### `kvit config` — Manage settings

```bash
kvit config set remote "gdrive:kvit"   # set rclone remote path
kvit config get remote                  # show current value
kvit config show                        # show all settings
```

### `kvit update` — Self-update

```bash
kvit update         # downloads latest version from GitHub Releases
sudo kvit update    # if installed to /usr/local/bin
```

## Data format

All files are plain CSV in your working directory:

**`expenses.csv`** — one row per product purchased:
```csv
date,store,product,price
2026-04-05,netto,ground-beef,39.95
2026-04-05,netto,milk,12.50
```

**`definitions.csv`** — maps products to categories:
```csv
product,category
ground-beef,meat
milk,dairy
```

**`exclusions.csv`** — products hidden from the AI prompt:
```csv
product
shoes
```

**`kvit.json`** — local config (rclone remote path).

These CSVs are designed to be opened directly in Google Sheets for viewing, editing, or building charts.

## Google Drive setup

kvit uses [rclone](https://rclone.org) to sync files with Google Drive.

### 1. Install rclone

```bash
# macOS
brew install rclone

# Linux
sudo apt install rclone
```

### 2. Configure Google Drive

```bash
rclone config
```

Follow the prompts:
1. `n` — New remote
2. Name it `gdrive`
3. Choose `Google Drive`
4. Leave client_id and client_secret blank
5. Scope: `1` (Full access)
6. Leave remaining fields blank
7. `y` — Auto config (opens a browser to sign in with your Google account)
8. Confirm

Verify it works:
```bash
rclone ls gdrive:
```

### 3. Create a folder and configure kvit

```bash
rclone mkdir gdrive:kvit
kvit config set remote "gdrive:kvit"
```

### 4. Sync your data

```bash
kvit sync push   # after adding expenses
kvit sync pull   # on a new machine
```

### 5. View in Google Sheets

Open [Google Drive](https://drive.google.com), navigate to the `kvit` folder, and double-click any CSV to open it in Google Sheets. You can edit cells directly — just `kvit sync pull` to get the changes locally.
