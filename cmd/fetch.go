package cmd

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"kvit/config"
	"kvit/drive"
	"kvit/storage"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Show last expense date locally and on remote",
	Run: func(cmd *cobra.Command, args []string) {
		requireReady()

		localISO := lastLocalISO()

		fmt.Print("Fetching remote... ")
		remoteISO, err := lastRemoteISO()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Print("\r                   \r")

		fmt.Printf("Local:  %s\n", fmtDateOrNone(localISO))
		fmt.Printf("Remote: %s\n", fmtDateOrNone(remoteISO))

		switch {
		case localISO == "" && remoteISO == "":
			fmt.Println("✓  in sync")
		case localISO > remoteISO:
			fmt.Println("⬆  push")
		case remoteISO > localISO:
			fmt.Println("⬇  pull")
		default:
			fmt.Println("✓  in sync")
		}
	},
}

func init() {
	syncCmd.AddCommand(fetchCmd)
}

func lastLocalISO() string {
	expenses, err := storage.LoadExpenses()
	if err != nil {
		return ""
	}
	last := ""
	for _, e := range expenses {
		if e.Date > last {
			last = e.Date
		}
	}
	return last
}

func lastRemoteISO() (string, error) {
	data, err := drive.FetchFileContent(config.ExpensesFile)
	if err != nil {
		return "", fmt.Errorf("remote: %w", err)
	}
	if data == nil {
		return "", nil
	}

	r := csv.NewReader(bytes.NewReader(data))
	records, err := r.ReadAll()
	if err != nil {
		return "", fmt.Errorf("parse remote CSV: %w", err)
	}

	last := ""
	for i, row := range records {
		if i == 0 || len(row) < 1 {
			continue
		}
		if row[0] > last {
			last = row[0]
		}
	}
	return last, nil
}

func fmtDateOrNone(iso string) string {
	if iso == "" {
		return "no data"
	}
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return fmt.Sprintf("%d %s %d", t.Day(), t.Month().String()[:3], t.Year())
}
