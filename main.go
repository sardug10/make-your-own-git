package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
)

// Usage: your_program.sh <command> <arg1> <arg2> ...
func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: mygit <command> [<args>...]\n")
		os.Exit(1)
	}

	switch command := os.Args[1]; command {
	case "init":
		for _, dir := range []string{".git", ".git/objects", ".git/refs"} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			}
		}

		headFileContents := []byte("ref: refs/heads/main\n")
		if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
		}

		fmt.Println("Initialized git directory")

	case "cat-file":
		// Implement the cat-file command here
		// check if the len(args) < 4
		if len(os.Args) != 4 {
			fmt.Fprintf(os.Stderr, "usage: mygit cat-file -p <object-hash>\n")
			os.Exit(1)
		}

		// check if the third argument is -p and fourth argument is a valid object hash
		readFlag := os.Args[2]
		objectHash := os.Args[3]
		if readFlag != "-p" && len(objectHash) != 40 {
			fmt.Fprintf(os.Stderr, "usage: mygit cat-file -p <object-hash>\n")
			os.Exit(1)
		}

		// create the file path
		dirName := objectHash[0:2]
		fileName := objectHash[2:]
		filePath := fmt.Sprintf("./.git/objects/%s/%s", dirName, fileName)

		// read the file
		fileContents, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %s\n", err)
			os.Exit(1)
		}

		// decompress the file contents
		b := bytes.NewReader(fileContents)
		r, err := zlib.NewReader(b)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decompressing the file: %s\n", err)
			os.Exit(1)
		}

		decompressedData, err := io.ReadAll(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading decompressed data: %s\n", err)
			os.Exit(1)
		}
		r.Close()

		nullIndex := bytes.IndexByte(decompressedData, 0)
		if nullIndex == -1 {
			fmt.Fprintf(os.Stderr, "Invalid object format: missing metadata separator\n")
			os.Exit(1)
		}

		// Extract and print the actual content (everything after the null byte)
		content := decompressedData[nullIndex+1:]
		fmt.Print(string(content))

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}
