package main

import (
	"maps"
	"math/rand/v2"
	"os"
	"path"
	"strconv"
	"sync"
	"testing"
)

func Test_collectJSONPaths(t *testing.T) {
	// generate some test files in a temp dir
	tDir := t.TempDir()
	dir1 := path.Join(tDir, "dir1")
	if err := os.Mkdir(dir1, 0755); err != nil {
		t.Fatal(err)
	}
	var dir1FilePaths = make([]string, rand.UintN(20)+1)
	for i := range dir1FilePaths {
		dir1FilePaths[i] = path.Join(dir1, strconv.FormatUint(uint64(i), 10)+".json")
		if f, err := os.Create(dir1FilePaths[i]); err != nil {
			t.Fatal(err)
		} else {
			f.Close()
			t.Log("created file " + dir1FilePaths[i])
		}
	}

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		argPaths []string
		want     []string
		wantErr  bool
	}{
		{"a single .json file", []string{dir1FilePaths[0]}, []string{dir1FilePaths[0]}, false},
		{"many individual .json files", dir1FilePaths, dir1FilePaths, false},
		{"slurp dir1", []string{dir1}, dir1FilePaths, false},
		{"dir and json", []string{dir1, dir1FilePaths[0]}, append(dir1FilePaths, dir1FilePaths[0]), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := collectJSONPaths(tt.argPaths)
			if (gotErr != nil) != tt.wantErr {
				t.Fatalf("error mismatch. Wanted err? %v | Actual err: %v", tt.wantErr, gotErr)
			}
			if !SlicesUnorderedEqual(tt.want, got) {
				t.Fatalf("bad result.\nExpected %v\nactual %v", tt.want, got)
			}
		})
	}
}

// SlicesUnorderedEqual compares the elements of the given slices for equality and equal count without taking order of the elements into account.
// Borrowed from https://github.com/rflandau/Orv (Slims implementation)
func SlicesUnorderedEqual[T comparable](a []T, b []T) bool {
	// convert each slice into a map of key --> count
	var wg sync.WaitGroup

	am := make(map[T]uint)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, k := range a {
			am[k] += 1
		}
	}()

	bm := make(map[T]uint)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, k := range b {
			bm[k] += 1
		}
	}()

	wg.Wait()
	return maps.Equal(am, bm)
}
