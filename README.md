# kvit

A fast CLI expense tracker built in Go. Track daily grocery expenses, analyze spending, and use AI (Gemini/ChatGPT) to scan receipts automatically.

## How it works

1. Take a photo of your receipt
2. Run `kvit prompt` — it copies an AI prompt to your clipboard
3. Paste the prompt + receipt photo into Gemini or ChatGPT
4. The AI generates a `kvit add` command
5. Paste and run the command — done

Data is stored as simple CSV files in `./data/`, designed to be synced via Google Drive and analyzed with Google Sheets, Grafana, or any tool.

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

## Install

Download the latest binary from [Releases](https://github.com/datsfain/kvit/releases) and place it in your PATH.

Update to the latest version:
```bash
kvit update
```

## Currency

All prices are in DKK (Danish Krone).
