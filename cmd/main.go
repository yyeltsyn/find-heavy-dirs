package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yyeltsyn/find-heavy-dirs/internal/core"
	"github.com/yyeltsyn/find-heavy-dirs/internal/scanner"
	"github.com/yyeltsyn/find-heavy-dirs/internal/webui"
)

var limitFlag = flag.Int("top", 10, "How many top items show")
var verboseFlag = flag.Bool("v", false, "Show progress")
var webuiFlag = flag.Bool("w", false, "Open results in browser")
var directoryArg string

func main() {
	parseFlagsAndArguments()

	var results = make(chan core.FileWithSize)
	var scanDone = make(chan int)

	go core.Start(results)
	go scanner.Scan(directoryArg, results, scanDone)
	if *webuiFlag {
		go func() {
			err := webui.Start(directoryArg, *limitFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: start webui: %v\n", err)
			}
		}()
	}

	var ticker <-chan time.Time
	if *verboseFlag {
		ticker = time.NewTicker(time.Second).C
	}

LOOP:
	for {
		select {
		case <-ticker:
			printResults(directoryArg, *limitFlag)
			fmt.Println()
		case <-scanDone:
			printResults(directoryArg, *limitFlag)
			break LOOP
		}
	}

	if *webuiFlag {
		for webui.Active() {
			time.Sleep(100 * time.Millisecond)
		}
		fmt.Fprintln(os.Stderr, "No active web clients, exit")
	}
}

func parseFlagsAndArguments() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, "Usage: %s [OPTION]... [DIRECTORY]\n", os.Args[0])
		fmt.Fprintf(os.Stdout, "Scans DIRECTORY (by default current directory)\n\n")
		flag.CommandLine.SetOutput(os.Stdout)
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "Error: More than one directory given")
		flag.Usage()
		os.Exit(1)
	}
	directoryArg = flag.Arg(0)
	if directoryArg == "" {
		directoryArg = "."
	}
	var err error // for reliability
	directoryArg, err = filepath.Abs(directoryArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fileinfo, err := os.Stat(directoryArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if !fileinfo.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: is not directory: %s\n", directoryArg)
		os.Exit(1)
	}
}

func printResults(dir string, limit int) {
	pattern := "\t%8s\t%s\n"
	top, rest, total := core.Top(dir, limit)
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
