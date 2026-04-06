# kvit

A fast CLI expense tracker built in Go. Track daily grocery expenses, analyze spending, and use AI (Gemini/ChatGPT) to scan receipts automatically.

## How it works

1. Take a photo of your receipt
2. Run `kvit prompt` — it copies an AI prompt to your clipboard
3. Paste the prompt + receipt photo into Gemini or ChatGPT
4. The AI generates a `kvit add` command
5. Paste and run the command — done

Data is stored as simple CSV files in your working directory, designed to be synced via Google Drive and analyzed with Google Sheets, Grafana, or any tool.

## Commands

### `kvit add` — Add expenses (one-liner)

```bash
kvit add netto ground-beef:200 cucumber:30           # today's date
kvit add netto 2026-04-05 ground-beef:200             # specific date
kvit add netto beef:200 + føtex milk:12 bread:25      # multiple stores
```

### `kvit interactive` (or `kvit i`) — Add expenses interactively

Guided TUI with:
- Date picker (up/down arrows to switch days, shows "Today", "Yesterday", etc.)
- Store autocomplete (Tab to accept)
- Product autocomplete from your definitions
- Confirmation dialog before saving
- Automatic prompting to categorize new products

### `kvit prompt` — Generate AI receipt-scanning prompt

Generates a prompt tailored to your known products and stores, and copies it to your clipboard. Paste it into Gemini/ChatGPT along with a receipt photo, and it will output a ready-to-run `kvit add` command.

### `kvit exclude` — Manage prompt exclusions

Control which products are shared in the AI prompt.

```bash
kvit exclude add shoes          # hide from AI prompt
kvit exclude remove shoes       # show again
kvit exclude list               # see all exclusions
```

## Data format

**`expenses.csv`** — all expenses:
```
date,store,product,price
2026-04-05,netto,ground-beef,200.00
```

**`definitions.csv`** — product categories:
```
product,category
ground-beef,meat
```

**`exclusions.csv`** — products hidden from AI prompt.

### `kvit sync` — Sync with Google Drive

Upload and download your CSV files via [rclone](https://rclone.org).

```bash
kvit sync push    # upload local CSVs to Google Drive (overwrites remote)
kvit sync pull    # download CSVs from Google Drive (overwrites local)
```

### `kvit config` — Manage settings

```bash
kvit config set remote "gdrive:kvit"   # set rclone remote path
kvit config get remote                  # show current remote
kvit config show                        # show all settings
```

### `kvit update` — Self-update

```bash
kvit update       # update to latest version from GitHub Releases
```

## Google Drive sync setup

kvit uses [rclone](https://rclone.org) to sync CSV files with Google Drive.

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
7. `y` — Auto config (opens browser to log in)
8. Confirm

Test it:
```bash
rclone ls gdrive:
```

### 3. Create a folder and configure kvit

```bash
rclone mkdir gdrive:kvit
kvit config set remote "gdrive:kvit"
```

### 4. Sync

```bash
kvit sync push    # after adding expenses locally
kvit sync pull    # on a new machine to get your data
```

`push` overwrites remote, `pull` overwrites local — always pick the right direction. You can also view and edit the CSVs directly in Google Sheets.

## Install

Download the latest binary from [Releases](https://github.com/datsfain/kvit/releases) and place it in your PATH.

## Currency

All prices are in DKK (Danish Krone).
