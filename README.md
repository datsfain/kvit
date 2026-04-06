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
kvit sync push                                            # sync to Google Drive
```

All sync commands guide you through sign-in and folder setup on first use — no manual setup needed.

## Scan receipts with AI

```bash
kvit prompt        # copies an AI prompt to your clipboard
```

Paste the prompt into any AI assistant (Gemini, ChatGPT, Claude, etc.) along with one or more photos of your receipts. The AI reads them (even in Danish) and generates a ready-to-run `kvit add` command.

## Sync with Google Drive

```bash
kvit sync push     # upload data
kvit sync pull     # download data
kvit sync open     # open folder in browser
kvit sync link     # link to a different folder
```

On first use, `push` or `pull` will prompt you to sign in and link a Google Drive folder — just follow the prompts. You can also set up manually with `kvit auth` and `kvit sync link`.

## All commands

| Command | Description |
|---|---|
| `kvit add <store> [date] <product:price>...` | Add expenses. Use `+` between stores for multi-store. |
| `kvit i` | Interactive mode with autocomplete and date picker. |
| `kvit summary` | Generate an interactive HTML spending report. |
| `kvit prompt` | Copy an AI receipt-scanning prompt to clipboard. |
| `kvit sync push/pull/open/link` | Google Drive sync. |
| `kvit auth` | Sign in with Google. `--force` to re-auth, `logout` to sign out. |
| `kvit exclude add/remove/list` | Control which products appear in the AI prompt. |
| `kvit update` | Update kvit to the latest version. |

## Spending report

`kvit summary` generates an interactive HTML report with:
- Spending by category and store (pie + bar charts)
- Daily and weekly/monthly trends with category drill-down
- Sortable, searchable expense table with expandable categories
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

**Owner setup:**
```bash
kvit sync push             # uploads data (prompts to sign in and link folder on first use)
kvit sync open             # open folder in browser, then share it with family
```

**Family member — quick setup:**
```bash
curl -sSL https://raw.githubusercontent.com/datsfain/kvit/main/install.sh | bash
mkdir ~/expenses && cd ~/expenses
kvit sync pull             # prompts to sign in, then paste the shared folder URL
```

**Family member — manual setup:**
```bash
kvit auth                  # sign in with Google
kvit sync link             # paste the shared folder URL
kvit sync pull             # download data
```

**Daily use:** `pull` before adding, `push` after.
