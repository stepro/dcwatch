package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/docker/docker/builder/dockerignore"
	"github.com/docker/docker/pkg/fileutils"
)

// File represents a file in a docker context
type File struct {
	path    string
	modTime time.Time
}

// Files represents the files in a docker context
type Files []File

func (a Files) Len() int           { return len(a) }
func (a Files) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Files) Less(i, j int) bool { return a[i].path < a[j].path }

// interval is the time to wait between polls
const interval = 200 * time.Millisecond

func find2(files []File, dir string, pm *fileutils.PatternMatcher) []File {
	file, err := os.Open("./" + dir)
	if err != nil {
		return nil
	}
	infos, err := file.Readdir(-1)
	file.Close()
	if err != nil {
		return nil
	}
	for _, info := range infos {
		path := dir + info.Name()
		exclude, _ := pm.Matches(path)
		if !exclude {
			files = append(files, File{
				path:    path,
				modTime: info.ModTime(),
			})
		}
		if info.IsDir() && (!exclude || pm.Exclusions()) {
			files = find2(files, path+"/", pm)
		}
	}
	return files
}

func find(pm *fileutils.PatternMatcher) []File {
	var files []File
	files = find2(files, "", pm)
	if files != nil {
		sort.Sort(Files(files))
	}
	return files
}

func compare(baseline []File, latest []File) {
	var b, l int
	for b < len(baseline) && l < len(latest) {
		if baseline[b].path < latest[l].path {
			fmt.Println("-", baseline[b].path)
			b++
		} else if baseline[b].path == latest[l].path {
			if baseline[b].modTime.UnixNano() < latest[l].modTime.UnixNano() {
				fmt.Println("*", latest[l].path)
			}
			b++
			l++
		} else {
			fmt.Println("+", latest[l].path)
			l++
		}
	}
	for b < len(baseline) {
		fmt.Println("-", baseline[b].path)
		b++
	}
	for l < len(latest) {
		fmt.Println("+", latest[l].path)
		l++
	}
}

func main() {
	// Read patterns from .dockerignore file
	var patterns []string
	f, err := os.Open(".dockerignore")
	if err == nil {
		patterns, err = dockerignore.ReadAll(f)
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
	}

	// Create pattern matcher
	pm, err := fileutils.NewPatternMatcher(patterns)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize baseline
	baseline := find(pm)

	// Loop and watch for changes
	for {
		time.Sleep(interval)
		latest := find(pm)
		if latest != nil {
			if baseline != nil {
				compare(baseline, latest)
			}
			baseline = latest
		}
	}
}
