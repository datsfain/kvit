package drive

import (
	"fmt"
	"strings"
)

const barWidth = 24

// PrintProgress renders a single-line progress bar that updates in-place.
// Gold while in progress, green when complete.
func PrintProgress(done, total int, label string) {
	pct := done * 100 / total
	filled := barWidth * done / total
	empty := barWidth - filled

	bar := strings.Repeat("━", filled)
	if filled < barWidth {
		bar += "╸" + strings.Repeat("─", empty-1)
	}

	color := "\033[33m" // gold
	if done == total {
		color = "\033[32m" // green
	}
	reset := "\033[0m"
	dim := "\033[90m"

	fmt.Printf("\r  %s%s%s %s%3d%%%s %s%s%s",
		color, bar, reset,
		dim, pct, reset,
		dim, label, reset,
	)
	fmt.Print("\033[K") // clear rest of line
	if done == total {
		fmt.Println()
	}
}
