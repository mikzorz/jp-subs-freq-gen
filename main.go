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
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/asticode/go-astisub"
	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
	"golang.org/x/text/width"
)

type unicodeRange struct {
	start, end int
}

var junkRanges = []unicodeRange{
	unicodeRange{'　', '〿'},     // Japanese-style punctuation
	unicodeRange{65280, 65504}, //Full-width roman characters and half-width katakana (ff00 - ffef)
	unicodeRange{'!', '~'},     // C0 Controls and Basic Latin
	unicodeRange{'←', '⇿'},     // Arrows
	unicodeRange{'☀', '⛿'},     // Miscellaneous Symbols
	unicodeRange{' ', ' '},     // Whitespace
	unicodeRange{'…', '…'},
}

var hiraganaRange = unicodeRange{'ぁ', 'ゖ'}

var root string
var outPath string
var recurse bool
var verbose bool

func main() {
	parseFlags()

	files := getFiles(root, recurse)
	if len(*files) == 0 {
		fmt.Println("No files found")
		os.Exit(0)
	}

	// Tokenize text
	t, err := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
	if err != nil {
		panic(err)
	}

	// wakati
	frequencies := make(map[string]int)
	longestTokenLen := 0

	// Get text from files
	for _, f := range *files {
		if verbose {
			fmt.Println("Processing", f)
		}
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
			cleanToken := removeJunkFromToken(token)
			if cleanToken == "" {
				continue
			}
			frequencies[cleanToken]++
			if tokenLen := utf8.RuneCountInString(token); tokenLen > longestTokenLen {
				longestTokenLen = tokenLen
			}
		}
	}

	// Sort tokens by frequency

	heighestFreq := 0

	keys := make([]string, 0, len(frequencies))
	for token, freq := range frequencies {
		keys = append(keys, token)
		if freq > heighestFreq {
			heighestFreq = freq
		}
	}

	sort.Slice(keys, func(i, j int) bool {
		// desc order
		return frequencies[keys[i]] > frequencies[keys[j]]
	})

	var out string
	tColW, fColW := 2*longestTokenLen, len(strconv.Itoa(heighestFreq))+4

	// Table Headers
	out += fmt.Sprint("|" + strings.Repeat("-", tColW) + "|" + strings.Repeat("-", fColW) + "|" + "\n")
	out += fmt.Sprintf("|%-"+strconv.Itoa(tColW)+"s|%-"+strconv.Itoa(fColW)+"s|\n", "Token", "Freq")
	out += fmt.Sprint("|" + strings.Repeat("-", tColW) + "|" + strings.Repeat("-", fColW) + "|" + "\n")

	// The actual useful info.
	for _, k := range keys {
		shortenBy := 0
		// If character is a fullwidth char, add 1 to shortenBy.
		for _, r := range k {
			p := width.LookupRune(r)
			if p.Kind() == width.EastAsianWide || p.Kind() == width.EastAsianFullwidth {
				shortenBy++
			}
		}
		curTColW := strconv.Itoa(tColW - shortenBy)
		fmtstr := "|%-" + curTColW + "s|%-" + strconv.Itoa(fColW) + "d|\n"
		out += fmt.Sprintf(fmtstr, k, frequencies[k])
	}

	// Save result to file
	if outPath == "" {
		outPath = root // What happens if you don't use pointers?
	}
	actualOutPath, _ := filepath.Abs(outPath)

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

func parseFlags() {
	flag.StringVar(&root, "in", "", "filepath/root directory to parse")
	flag.StringVar(&outPath, "out", "", "destination of output file")
	flag.BoolVar(&recurse, "r", true, "search through child directories?")
	flag.BoolVar(&verbose, "v", false, "verbosity")
	flag.Parse()
	if root == "" {
		log.Println("Must provide a filepath with -in")
		os.Exit(1)
	}
	if verbose {
		log.Printf("\"recurse\" set to %t\n", recurse)
	}
}

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

var alreadyWalked = false

// Should this return an error at some point?
func checkIfSubFile() (fs.WalkDirFunc, *[]string) {
	files := []string{}
	return func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() && !recurse && alreadyWalked {
			return fs.SkipDir
		}
		alreadyWalked = true

		if err == nil && !d.IsDir() {
			_, err := astisub.OpenFile(path)
			if err != nil {
				if verbose {
					log.Println(err)
				}
				return nil
			}
			files = append(files, path)
		}
		return nil
	}, &files

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

// If token is just a single kana, return empty string.
// If token contains junk, remove the junk from the token.
func removeJunkFromToken(token string) string {
	cleanToken := token
	for i := hiraganaRange.start; i <= hiraganaRange.end; i++ {
		if token == string(rune(i)) {
			return ""
		}
	}

	for _, char := range token {
		for _, r := range junkRanges {
			for i := r.start; i <= r.end; i++ {
				if int(char) == i {
					cleanToken = strings.ReplaceAll(cleanToken, string(char), "")
				}
			}
		}
	}
	return cleanToken
}
