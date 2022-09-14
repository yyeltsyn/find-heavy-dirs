package core

import (
	"path/filepath"
	"sort"
	"strings"
)

type FileWithSize struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

type request struct {
	dir     string
	limit   int
	results chan<- FileWithSize
}

type Core struct {
	pathSize map[string][]FileWithSize
	sorted   map[string]bool
	requests chan request
}

func NewCore() *Core {
	return &Core{
		pathSize: make(map[string][]FileWithSize),
		sorted:   make(map[string]bool),
		requests: make(chan request),
	}
}

func (c *Core) Start(results <-chan FileWithSize) {
	for {
		select {
		case result, ok := <-results:
			if ok {
				c.appendResult(result)
			} else {
				results = nil
			}
		case request := <-c.requests:
			c.top(request)
		}
	}
}

func (c *Core) Top(dir string, limit int) (top []FileWithSize, rest FileWithSize, total FileWithSize) {
	resultsChannel := make(chan FileWithSize)
	c.requests <- request{dir, limit, resultsChannel}
	var results = make([]FileWithSize, 0, limit)
	for result := range resultsChannel {
		results = append(results, result)
	}
	n := len(results)
	top = results[:n-2]
	rest = results[n-2]
	total = results[n-1]
	return
}

func (c *Core) appendResult(res FileWithSize) {
	var path = res.Path
	var dir string
	for dir != "/" {
		dir, _ = filepath.Split(filepath.Clean(path))
		slice := c.pathSize[dir]
		var found bool
		for i := range slice {
			if slice[i].Path == path {
				slice[i].Size += res.Size
				found = true
				break
			}
		}
		if !found {
			slice = append(slice, FileWithSize{path, res.Size})
		}
		c.pathSize[dir] = slice
		c.sorted[dir] = false
		path = dir
	}
}

func (c *Core) top(req request) {
	dir := req.dir
	if !strings.HasSuffix(dir, string(filepath.Separator)) {
		dir += string(filepath.Separator)
	}
	slice := c.pathSize[dir]
	if !c.sorted[dir] {
		sort.Slice(slice, func(i, j int) bool {
			return slice[i].Size > slice[j].Size
		})
		c.sorted[dir] = true
	}

	var totalSize int64
	for _, fws := range slice {
		totalSize += fws.Size
	}
	var restSize = totalSize

	limit := req.limit
	if len(slice) < limit {
		limit = len(slice)
	}
	for i := 0; i < limit; i++ {
		req.results <- slice[i]
		restSize -= slice[i].Size
	}

	req.results <- FileWithSize{
		Path: "REST",
		Size: restSize,
	}

	req.results <- FileWithSize{
		Path: dir,
		Size: totalSize,
	}

	close(req.results)
}
