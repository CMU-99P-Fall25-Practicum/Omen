package main

import (
	"os"
	"path"
	"testing"
	"time"
)

func Test_findLatestDirectory(t *testing.T) {
	const (
		tempDirPerm  os.FileMode = 0755
		tempFilePerm os.FileMode = 0755
	)
	var (
		// mostRecent is the name of the highest timestamp in each directory
		mostRecent string = time.Now().Format(directoryNameFormat)
	)

	// generate a few directory structures to test on
	var (
		tDir                = t.TempDir()
		zeroFileDir  string = path.Join(tDir, "zero")
		oneFileDir   string = path.Join(tDir, "one")
		twoFileDir   string = path.Join(tDir, "two")
		threeFileDir string = path.Join(tDir, "three")
	)
	if err := os.Mkdir(zeroFileDir, tempDirPerm); err != nil {
		t.Fatal(err)
	}
	{
		if err := os.Mkdir(oneFileDir, tempDirPerm); err != nil {
			t.Fatal(err)
		} else if err := os.Mkdir(path.Join(oneFileDir, mostRecent), tempFilePerm); err != nil {
			t.Fatal(err)
		}
	}
	{
		if err := os.Mkdir(twoFileDir, tempDirPerm); err != nil {
			t.Fatal(err)
		} else if err := os.Mkdir(path.Join(twoFileDir, mostRecent), tempFilePerm); err != nil {
			t.Fatal(err)
		} else if err := os.Mkdir(path.Join(twoFileDir, time.Now().AddDate(0, -1, 0).Format(directoryNameFormat)), tempFilePerm); err != nil {
			t.Fatal(err)
		}
	}
	{
		if err := os.Mkdir(threeFileDir, tempDirPerm); err != nil {
			t.Fatal(err)
		} else if err := os.Mkdir(path.Join(threeFileDir, mostRecent), tempFilePerm); err != nil {
			t.Fatal(err)
		} else if err := os.Mkdir(path.Join(threeFileDir, time.Now().AddDate(0, -3, 0).Format(directoryNameFormat)), tempFilePerm); err != nil {
			t.Fatal(err)
		} else if err := os.Mkdir(path.Join(threeFileDir, time.Now().Add(-3*time.Minute).Format(directoryNameFormat)), tempFilePerm); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name     string
		dirToUse uint8 // must be 0, 1, 2, or 3; correlates to the above directories
		want     string
		wantErr  bool
	}{
		{"zero file dir; err", 0, "", true},
		{"single file dir", 1, mostRecent, false},
		{"two file dir", 2, mostRecent, false},
		{"three file dir", 3, mostRecent, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dir string
			switch tt.dirToUse {
			case 0:
				dir = zeroFileDir
			case 1:
				dir = oneFileDir
			case 2:
				dir = twoFileDir
			case 3:
				dir = threeFileDir
			default:
				t.Fatal("must use a pre-created directory enumerated to 0-3")
			}
			got, gotErr := findLatestDirectory(dir)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("findLatestDirectory() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("findLatestDirectory() succeeded unexpectedly")
			}
			if got != path.Join(dir, tt.want) {
				t.Errorf("findLatestDirectory() = %v, want %v", got, tt.want)
			}
		})
	}
}
