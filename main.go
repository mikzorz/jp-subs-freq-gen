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

//var particles  = []string{"の", "は", "て", "に", "が", "た", "を", "だ", "で", "な", "と", "よ"}
var junkTokens = []string{"(", " ", "一", "-", "（", "）", ")", "｡", "､", ".", "％", "～", "？", "　", "…", "！", "”", "“", "･", "—", "➡", "♪", "≪", "≫", "＞", "＜", "!", "‼", "?", "〞", "「", "｢", "｣", "[", "]", "♬", "１", "２", "３", "４", "５", "６", "７", "８", "９", "０", "\\"}
var root string
var outPath string
var verbose bool

func main() {
	// Get subfile extension(s) from cli args (not done)
	parseFlags()

	files := getFiles(root, true)

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
	flag.BoolVar(&verbose, "v", false, "verbosity")
	flag.Parse()
	if root == "" {
		log.Println("Must provide a filepath with -in")
		os.Exit(1)
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
func checkIfSubFile() (fs.WalkDirFunc, *[]string) {
	files := []string{}
	return func(path string, d fs.DirEntry, err error) error {
		// If file ext matches with specified ext, return true.
		// Split filename by ".", get last segment.
		if err == nil && !d.IsDir() {
			_, err := astisub.OpenFile(path) // Copy this to walk func
			if err != nil {
				if verbose {
					log.Println(err)
				}
				return nil
			}
			// USE filepath.Ext(string) instead
			//segs := strings.Split(d.Name(), ".")
			//if len(segs) <= 1 {
			//	return nil
			//}
			//ext := segs[len(segs)-1]
			//if ext == "srt" || ext == "ass" {
			files = append(files, path)
			//}
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

func isJunkToken(token string) bool {
	for _, jt := range junkTokens {
		if token == jt {
			return true
		}
	}
	return false
}

func removeJunkFromToken(token string) string {
	cleanToken := token
	for _, r := range token {
		if isJunkToken(string(r)) {
			cleanToken = strings.ReplaceAll(cleanToken, string(r), "")
		}
	}
	return cleanToken
}
