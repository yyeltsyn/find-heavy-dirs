package core

import (
	"path/filepath"
	"strconv"
	"testing"
)

func TestTop(t *testing.T) {
	c := NewCore()
	results := make(chan FileWithSize)
	go c.Start(results)
	results <- FileWithSize{"/var/www/a", 1}
	results <- FileWithSize{"/var/www/b", 2}
	results <- FileWithSize{"/var/www/c", 3}
	results <- FileWithSize{"/var/a", 1}
	results <- FileWithSize{"/var/b", 2}
	results <- FileWithSize{"/var/c", 3}
	results <- FileWithSize{"/a", 1}
	results <- FileWithSize{"/b", 2}
	results <- FileWithSize{"/c", 3}
	type inputStruct struct {
		dir   string
		limit int
	}
	type outputStruct struct {
		top   []FileWithSize
		rest  FileWithSize
		total FileWithSize
	}
	var tests = []struct {
		input  inputStruct
		expect outputStruct
	}{
		{inputStruct{"/var/www/", 2}, outputStruct{[]FileWithSize{{"/var/www/c", 3}, {"/var/www/b", 2}}, FileWithSize{"REST", 1}, FileWithSize{"/var/www/", 6}}},
		{inputStruct{"/var/", 2}, outputStruct{[]FileWithSize{{"/var/www/", 6}, {"/var/c", 3}}, FileWithSize{"REST", 3}, FileWithSize{"/var/", 12}}},
		{inputStruct{"/", 2}, outputStruct{[]FileWithSize{{"/var/", 12}, {"/c", 3}}, FileWithSize{"REST", 3}, FileWithSize{"/", 18}}},
	}

	slicesEqual := func(a, b []FileWithSize) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	for _, test := range tests {
		top, rest, total := c.Top(test.input.dir, test.input.limit)
		if !slicesEqual(top, test.expect.top) {
			t.Errorf("top: f(%v) = %v, expect %v", test.input, top, test.expect.top)
		}
		if rest != test.expect.rest {
			t.Errorf("rest: f(%v) = %v, expect %v", test.input, rest, test.expect.rest)
		}
		if total != test.expect.total {
			t.Errorf("total: f(%v) = %v, expect %v", test.input, total, test.expect.total)
		}
	}
}

func BenchmarkTop(b *testing.B) {
	c := NewCore()
	results := make(chan FileWithSize)
	go c.Start(results)
	dirs := make([]string, 20)
	files := make([]string, 20)
	for i := 0; i < len(dirs); i++ {
		dirs[i] = "subdir" + strconv.Itoa(i)
	}
	for i := 0; i < len(files); i++ {
		files[i] = "file" + strconv.Itoa(i)
	}
	for _, l1 := range dirs {
		for _, l2 := range dirs {
			for _, l3 := range dirs {
				for _, f := range files {
					results <- FileWithSize{
						Path: filepath.Join("/", l1, l2, l3, f),
						Size: 1,
					}
				}
			}
		}
	}

	path := filepath.Join(dirs[0], dirs[1]) + "/"
	for i := 0; i < b.N; i++ {
		_, _, _ = c.Top(path, 10)
	}
}
