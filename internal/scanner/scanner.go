package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/yyeltsyn/find-heavy-dirs/internal/core"
)

func Scan(dir string, results chan<- core.FileWithSize) {
	var wg sync.WaitGroup

	wg.Add(1)
	go scan(dir, results, &wg)

	wg.Wait()

	close(results)
}

var sema = make(chan int, 20)

func scan(dir string, results chan<- core.FileWithSize, wg *sync.WaitGroup) {
	defer wg.Done()
	sema <- 1
	entries, err := os.ReadDir(dir)
	<-sema
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan %q: %s\n", dir, err)
	}
	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())
		if fullPath == "/proc" { // dirty hack
			continue
		}
		if entry.IsDir() {
			wg.Add(1)
			go scan(fullPath, results, wg)
		} else if entry.Type().IsRegular() {
			finfo, err := entry.Info()
			if err != nil {
				fmt.Fprintf(os.Stderr, "info %q: %s\n", fullPath, err)
				continue
			}
			results <- core.FileWithSize{
				Path: fullPath,
				Size: finfo.Size(),
			}
		}
	}
}
