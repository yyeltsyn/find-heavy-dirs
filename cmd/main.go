package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yyeltsyn/find-heavy-dirs/internal/cli"
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
	var webuiDone = make(chan int)

	core1 := core.NewCore()
	go core1.Start(results)

	go func() {
		scanner.Scan(directoryArg, results)
		close(scanDone)
	}()

	go cli.Start(core1, directoryArg, *limitFlag, *verboseFlag, scanDone)

	go func() {
		if *webuiFlag {
			err := webui.Start(core1, directoryArg, *limitFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: webui: %v\n", err)
			}
		}
		close(webuiDone)
	}()

	<-scanDone
	<-webuiDone
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
