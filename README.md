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

# Sign in with Google for Drive sync
kvit auth

# Add your first expenses
kvit add netto ground-beef:39.95 milk:12.50 bread:22

# Or use the interactive mode
kvit i

# Sync to Google Drive
kvit sync push
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

### `kvit auth` — Sign in with Google

```bash
kvit auth              # opens browser for Google sign-in
kvit auth --force      # re-authenticate
kvit auth logout       # remove stored credentials
```

Opens your browser, you sign in with Google, and kvit stores a token locally at `~/.config/kvit/token.json`.

### `kvit sync` — Sync with Google Drive

```bash
kvit sync push   # upload local CSVs to Google Drive (overwrites remote)
kvit sync pull   # download from Google Drive to local (overwrites local)
kvit sync open   # open the kvit folder on Google Drive in your browser
kvit sync link   # link to a shared folder (for family sharing)
```

Files are stored in a `kvit` folder on your Google Drive. `push` overwrites remote, `pull` overwrites local — always pick the right direction.

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

These CSVs are designed to be opened directly in Google Sheets for viewing, editing, or building charts.

## Family sharing

Multiple people can share the same expense data through a shared Google Drive folder.

### Setup (folder owner)

```bash
kvit sync push             # push your data to Drive
kvit sync open             # opens the kvit folder in your browser
```

In Google Drive, right-click the `kvit` folder → **Share** → add your family member's email as **Editor**.

Copy the folder URL from the browser (e.g. `https://drive.google.com/drive/folders/1aBcD...`).

### Setup (family member)

```bash
# Install kvit
curl -sSL https://raw.githubusercontent.com/datsfain/kvit/main/install.sh | bash

# Create a local folder and enter it
mkdir ~/expenses && cd ~/expenses

# Sign in with their own Google account
kvit auth

# Link to the shared folder
kvit sync link
# Paste the shared folder URL when prompted

# Pull the existing data
kvit sync pull
```

### Daily use (both people)

```bash
kvit sync pull             # get latest changes
kvit add netto milk:12     # add expenses
kvit sync push             # upload changes
```

Always `pull` before adding expenses to avoid overwriting each other's changes.
