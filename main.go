package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/asticode/go-astisub"
	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
)

// Should return array of filepaths from a filepath
// If 'root' is a file, return []{'root'}
// If 'root' is a directory and recurse is true, return all files in 'root' and child directories.
// Else, just return the files in 'root'
func getFiles(root string, recurse bool) *[]string {
	walkFunc, files := checkIfSubFile()
	err := filepath.WalkDir(root, walkFunc)
	if err != nil {
		log.Fatal(err)
	}
	return files
}
func checkIfSubFile() (fs.WalkDirFunc, *[]string) {
	files := []string{}
	return func(path string, d fs.DirEntry, err error) error {
		// If file ext matches with specified ext, return true.
		// Split filename by ".", get last segment.
		//		log.Println(path)
		if err == nil && !d.IsDir() {
			// USE filepath.Ext(string) instead
			segs := strings.Split(d.Name(), ".")
			if len(segs) <= 1 {
				return nil
			}
			ext := segs[len(segs)-1]
			if ext == "srt" || ext == "ass" {
				files = append(files, path)
			}
		}
		return nil
	}, &files

}

func main() {
	// Get subfile extension(s) from cli args (not done)
	root := flag.String("in", "", "file/root directory to process")
	outPath := flag.String("out", "", "where to save file")
	flag.Parse()
	if *root == "" {
		log.Println("must provide a filepath")
		os.Exit(1)
	}
	files := getFiles(*root, true)

	// Tokenize text

	t, err := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
	if err != nil {
		panic(err)
	}
	// wakati
	fmt.Println("---wakati---")
	frequencies := make(map[string]int)

	// Get text from files
	for _, f := range *files {
		fmt.Println(f)
		subs, err := astisub.OpenFile(f) // Copy this to walk func
		if err != nil {
			log.Println(err)
		}
		var subsString string
		for _, item := range subs.Items {
			subsString += item.String()
		}
		seg := t.Wakati(subsString)

		for _, token := range seg {
			frequencies[token]++
		}
	}

	// Sort tokens by frequency

	keys := make([]string, 0, len(frequencies))
	for k := range frequencies {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		// desc order
		return frequencies[keys[i]] > frequencies[keys[j]]
	})

	var out string
	for _, k := range keys {
		out = fmt.Sprintln(out, k, frequencies[k])
	}

	// Save result to file
	if *outPath == "" {
		*outPath = *root // What happens if you don't use pointers?
	}
	actualOutPath, _ := filepath.Abs(*outPath)

	fi, err := os.Lstat(actualOutPath)
	if err != nil {
		log.Fatal(err)
	}

	switch mode := fi.Mode(); {
	case mode.IsRegular():
		actualOutPath = filepath.Dir(actualOutPath)
	}
	actualOutPath += "/freq.txt"

	err = WriteToFile(actualOutPath, out)
	if err != nil {
		log.Fatal(err)
	}
}

func WriteToFile(filename string, data string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.WriteString(file, data)
	if err != nil {
		return err
	}
	return file.Sync()
}
