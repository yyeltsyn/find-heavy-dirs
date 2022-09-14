package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/yyeltsyn/find-heavy-dirs/internal/core"
)

func Start(core1 *core.Core, dir string, limit int, verbose bool, scanDone <-chan int) {
	var ticker <-chan time.Time
	if verbose {
		ticker = time.NewTicker(time.Second).C
	}

LOOP:
	for {
		select {
		case <-ticker:
			printResults(core1, dir, limit)
			fmt.Println()
		case <-scanDone:
			break LOOP
		}
	}

	printResults(core1, dir, limit)
}

func printResults(core1 *core.Core, dir string, limit int) {
	pattern := "\t%8s\t%s\n"
	top, rest, total := core1.Top(dir, limit)
	for i, result := range top {
		fmt.Fprintf(os.Stdout, "% 2d."+pattern, i+1, bytesHumanReadable(result.Size), result.Path)
	}
	if rest.Size > 0 {
		fmt.Fprintf(os.Stdout, "..."+pattern, bytesHumanReadable(rest.Size), "others...")
	}
	fmt.Println()
	fmt.Fprintf(os.Stdout, "Total"+pattern, bytesHumanReadable(total.Size), total.Path)
}

func bytesHumanReadable(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
