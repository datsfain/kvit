# kvit

A fast expense tracker for the terminal. Track groceries, scan receipts with AI, sync to Google Drive, and visualize spending — all from the command line.

## Install

```bash
curl -sSL https://raw.githubusercontent.com/datsfain/kvit/main/install.sh | bash
```

## Quick start

```bash
mkdir ~/expenses && cd ~/expenses

kvit add netto ground-beef:39.95 milk:12.50 bread:22    # add expenses
kvit i                                                    # or use interactive mode
kvit summary                                              # view spending report
```

## Scan receipts with AI

```bash
kvit prompt        # copies an AI prompt to your clipboard
```

Paste the prompt + a photo of your receipt into [Gemini](https://gemini.google.com) or ChatGPT. The AI reads the receipt (even in Danish) and generates a `kvit add` command you can paste and run.

## Sync with Google Drive

```bash
kvit sync link     # link to a Google Drive folder (one-time setup)
kvit sync push     # upload data
kvit sync pull     # download data
kvit sync open     # open folder in browser
```

Create a folder on Google Drive, run `kvit sync link`, and paste the URL. kvit will prompt you to sign in and link a folder if you haven't yet.

## All commands

| Command | Description |
|---|---|
| `kvit add <store> [date] <product:price>...` | Add expenses. Use `+` between stores for multi-store. |
| `kvit i` | Interactive mode with autocomplete and date picker. |
| `kvit summary` | Generate an interactive HTML spending report. |
| `kvit prompt` | Copy an AI receipt-scanning prompt to clipboard. |
| `kvit sync link/push/pull/open` | Google Drive sync. |
| `kvit auth` | Sign in with Google. `--force` to re-auth, `logout` to sign out. |
| `kvit exclude add/remove/list` | Control which products appear in the AI prompt. |
| `kvit update` | Update kvit to the latest version. |

## Spending report

`kvit summary` generates an interactive HTML report with:
- Spending by category and store (pie + bar charts)
- Daily and weekly/monthly trends
- Drill-down from category to individual products
- Sortable, searchable expense table
- Date range presets (this month, last 3 months, etc.)
- Customizable category colors — pick colors in the report and save `colors.csv`

## Data format

Plain CSV files in your working directory:

| File | Contents |
|---|---|
| `expenses.csv` | `date,store,product,price` — one row per item |
| `definitions.csv` | `product,category` — maps products to categories |
| `exclusions.csv` | Products hidden from the AI prompt |
| `colors.csv` | Optional category colors for the report |

Open any CSV in Google Sheets for viewing, editing, or building charts.

## Family sharing

Share expenses with family through a shared Google Drive folder.

**Owner:**
```bash
kvit sync link             # link your Drive folder
kvit sync push             # upload data
kvit sync open             # share the folder with family
```

**Family member:**
```bash
curl -sSL https://raw.githubusercontent.com/datsfain/kvit/main/install.sh | bash
mkdir ~/expenses && cd ~/expenses
kvit sync link             # paste the shared folder URL
kvit sync pull             # download data
```

**Daily use:** `pull` before adding, `push` after.
