package main

import (
	"path/filepath"
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
)

const (
	defaultSearch  = "Vietnam Assets Pack by EightBall & Tobi"
	defaultReplace = "[VWV] Vietnam Assets Pack"
)


func usage() {
	fmt.Fprintf(os.Stderr,
		"Usage: %s <input.miz> <output.miz> <search> <replace>\n",
		os.Args[0],
	)
	os.Exit(1)
}

// Replace only inside requiredModules = { ... }
func replaceInRequiredModules(data []byte, find, replace string) ([]byte, bool) {
	key := []byte("requiredModules")
	idx := bytes.Index(data, key)
	if idx == -1 {
		return data, false
	}

	// find opening brace after key
	open := bytes.IndexByte(data[idx:], '{')
	if open == -1 {
		return data, false
	}
	open += idx

	depth := 0
	close := -1
	for i := open; i < len(data); i++ {
		switch data[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				close = i
				break
			}
		}
	}

	if close == -1 {
		return data, false
	}

	block := data[open : close+1]
	newBlock := bytes.ReplaceAll(block, []byte(find), []byte(replace))

	if bytes.Equal(block, newBlock) {
		return data, false
	}

	out := make([]byte, 0, len(data))
	out = append(out, data[:open]...)
	out = append(out, newBlock...)
	out = append(out, data[close+1:]...)

	return out, true
}

func main() {
	if len(os.Args) < 3 || len(os.Args) > 5 {
		fmt.Fprintf(os.Stderr,
			"Usage: %s <input.miz> <output.miz> [search] [replace]\n",
			os.Args[0],
		)
		os.Exit(1)
	}

	inMiz := os.Args[1]
	outMiz := os.Args[2]

	find := defaultSearch
	replace := defaultReplace

	if len(os.Args) < 4 {
		fmt.Println("Using default search string:", defaultSearch)
		fmt.Println("Using default replace string:", defaultReplace)
	}

	if len(os.Args) >= 4 {
		find = os.Args[3]
	}
	if len(os.Args) == 5 {
		replace = os.Args[4]
	}

	inPath, err := filepath.Abs(inMiz)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error resolving input path:", err)
		os.Exit(1)
	}

	outPath, err := filepath.Abs(outMiz)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error resolving output path:", err)
		os.Exit(1)
	}

	if inPath == outPath {
		fmt.Fprintln(os.Stderr, "Error: input and output .miz must be different files.")
		os.Exit(1)
	}

	r, err := zip.OpenReader(inMiz)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening input:", err)
		os.Exit(1)
	}
	defer r.Close()

	out, err := os.Create(outMiz)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating output:", err)
		os.Exit(1)
	}
	defer out.Close()

	w := zip.NewWriter(out)
	defer w.Close()

	replaced := false

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading file:", f.Name, err)
			os.Exit(1)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading file:", f.Name, err)
			os.Exit(1)
		}

		if f.Name == "mission" {
			data, replaced = replaceInRequiredModules(data, find, replace)
		}

		h := f.FileHeader
		h.Method = zip.Deflate

		wr, err := w.CreateHeader(&h)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error creating zip entry:", err)
			os.Exit(1)
		}
		_, err = wr.Write(data)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error writing zip entry:", err)
			os.Exit(1)
		}
	}

	if replaced {
		fmt.Println("Replacement done in requiredModules.")
	} else {
		fmt.Println("Warning: no replacement made (requiredModules or search string not found).")
	}

	fmt.Println("Output written to:", outMiz)
}
