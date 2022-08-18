package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/yyeltsyn/heavy-files/core"
	"github.com/yyeltsyn/heavy-files/scanner"
)

var limit = flag.Int("top", 10, "How many top items show")
var verbose = flag.Bool("v", false, "Show progress")

var results = make(chan core.FileWithSize)
var done = make(chan int)

func main() {
	flag.Parse()
	dir := flag.Arg(0)

	go scanner.Scan(dir, results, done)
	go core.Start(results)

	var ticker = time.NewTicker(time.Second)

outerLoop:
	for {
		select {
		case <-ticker.C:
			if *verbose {
				printResults(dir, *limit)
				fmt.Println()
			}
		case <-done:
			printResults(dir, *limit)
			break outerLoop
		}
	}
}

func printResults(dir string, limit int) {
	top, rest, total := core.Top(dir, limit)
	for i, result := range top {
		fmt.Fprintf(os.Stdout, "% 2d.\t%8s\t%s\n", i+1, ByteCountSI(result.Size), result.Path)
	}
	if rest.Size > 0 {
		fmt.Fprintf(os.Stdout, "...\t%8s\t%s\n", ByteCountSI(rest.Size), "others...")
	}
	fmt.Println()
	fmt.Fprintf(os.Stdout, "\t%8s\t%s\n", ByteCountSI(total.Size), total.Path)
}

func ByteCountSI(b int64) string {
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
