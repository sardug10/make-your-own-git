package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func CalculateGitObjectHash(content []byte) string {
	// Prepare the header: "blob <content length>\x00"
	header := fmt.Sprintf("blob %d\x00", len(content))

	// Concatenate the header and the file content
	data := append([]byte(header), content...)

	// Compute the SHA-1 hash
	hash := sha1.Sum(data)

	// Return the hash as a 40-character hexadecimal string
	return fmt.Sprintf("%x", hash)
}

func CalculateTreeObjectSha(mode string, nameOfFile string, hexHashOfFile string) string {

	rawHash, err := hex.DecodeString(hexHashOfFile)
	if err != nil {
		panic(err)
	}

	entry := []byte(fmt.Sprintf("%s %s\x00", mode, nameOfFile))
	entry = append(entry, rawHash...)

	header := fmt.Sprintf("tree %d\x00", len(entry))
	full := append([]byte(header), entry...)

	fmt.Printf("FULL STRING %s\n", string(full))

	sum := sha1.Sum(full)

	return fmt.Sprintf("%x", sum[:])
}

func writeCompressedObject(filePath string, content []byte, objectType string) error {
	// Create the header: "blob <size>\0"
	header := fmt.Sprintf("%s %d\000", objectType, len(content))
	headerBytes := []byte(header)

	// Combine the header and content
	data := append(headerBytes, content...)

	// Create a file for the compressed object
	objectFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating object file: %s", err)
	}
	defer objectFile.Close()

	// Create a zlib writer that will compress the data and write it to the file
	writer := zlib.NewWriter(objectFile)

	// Write the data to the zlib writer
	_, err = writer.Write(data)
	if err != nil {
		return fmt.Errorf("error writing data to zlib writer: %s", err)
	}

	// Close the writer to finish the compression
	writer.Close()
	objectFile.Close()

	return nil
}

func getTimeZoneOffset() string {
	now := time.Now()
	timestamp := now.Unix()

	_, offset := now.Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	minutes := (offset % 3600) / 60

	timezone := fmt.Sprintf("%s%02d%02d", sign, hours, minutes)

	return fmt.Sprintf("%d %s", timestamp, timezone)
}

func BuildTree(filePath string) string {
	var tree []byte

	entries, _ := os.ReadDir(filePath)
	var entryNames []string
	for _, e := range entries {
		if e.Name() == ".git" {
			continue
		}
		entryNames = append(entryNames, e.Name())
	}
	sort.Strings(entryNames)

	for _, name := range entryNames {
		fullPath := filepath.Join(filePath, name)
		info, _ := os.Stat(fullPath)

		if info.IsDir() {
			subTreeHash := BuildTree(fullPath)
			rawHash, _ := hex.DecodeString(subTreeHash)
			entry := fmt.Appendf(nil, "40000 %s\x00", name)
			tree = append(tree, entry...)
			tree = append(tree, rawHash...)
		} else {
			content, _ := os.ReadFile(fullPath)
			blobHash := CalculateGitObjectHash(content)
			rawHash, _ := hex.DecodeString(blobHash)

			dirName := blobHash[:2]
			fileName := blobHash[2:]
			path := filepath.Join(".git", "objects", dirName, fileName)
			os.MkdirAll(filepath.Dir(path), 0755)
			writeCompressedObject(path, content, "blob")

			entry := fmt.Appendf(nil, "100644 %s\x00", name)
			tree = append(tree, entry...)
			tree = append(tree, rawHash...)
		}
	}

	header := fmt.Sprintf("tree %d\x00", len(tree))
	full := append([]byte(header), tree...)

	treeHash := sha1.Sum(full)
	treeHashStr := fmt.Sprintf("%x", treeHash[:])
	dir := filepath.Join(".git", "objects", treeHashStr[:2])
	file := filepath.Join(dir, treeHashStr[2:])
	os.MkdirAll(dir, 0755)
	writeCompressedObject(file, tree, "tree")

	return treeHashStr
}

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

	case "hash-object":
		// Implement the hash-object command here
		// check if we have got all the args
		if len(os.Args) != 4 {
			fmt.Fprintf(os.Stderr, "usage: mygit hash-object -w <file-name>\n")
			os.Exit(1)
		}

		// check if the third argument is -w
		writeFlag := os.Args[2]
		if writeFlag != "-w" {
			fmt.Fprintf(os.Stderr, "usage: mygit hash-object -w <file-name>\n")
			os.Exit(1)
		}

		//read the file content
		fileName := os.Args[3]
		fileContent, err := os.ReadFile(fileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading the file: %s\n", err)
			os.Exit(1)
		}

		// generate the hash
		objectHash := CalculateGitObjectHash(fileContent)

		// create the file path
		dirName := objectHash[0:2]
		hashedFileName := objectHash[2:]
		dirPath := fmt.Sprintf(".mygit/objects/%s", dirName)
		dirErr := os.MkdirAll(dirPath, 0755)
		if dirErr != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", dirErr)
			os.Exit(1)
		}

		filePath := fmt.Sprintf(".mygit/objects/%s/%s", dirName, hashedFileName)

		// write the compressed object to the file
		writeErr := writeCompressedObject(filePath, fileContent, "blob")
		if writeErr != nil {
			fmt.Println("Error writing blob file:", err)
			os.Exit(1)
		}

		fmt.Printf("%s\n", objectHash)

	case "write-tree":
		treeHash := BuildTree("./tree")

		fmt.Printf("%s\n", treeHash)

	case "commit-tree":
		if len(os.Args) != 7 {
			fmt.Fprintf(os.Stderr, "usage: mygit commit-tree <tree_sha> -p <commit_sha> -m <message>\n")
			os.Exit(1)
		}

		treeSha := os.Args[2]
		parentSha := os.Args[4]
		commitMessage := os.Args[6]
		timestamp := getTimeZoneOffset()

		author := fmt.Sprintf("John Doe <john.doe@example.com> %s", timestamp)
		committer := fmt.Sprintf("John Doe <john.doe@example.com> %s", timestamp)

		commitContent := fmt.Sprintf("tree %s\nparent %s\nauthor %s\ncommitter %s\n\n%s\n", treeSha, parentSha, author, committer, commitMessage)

		commitHash := CalculateGitObjectHash([]byte(commitContent))

		dirName := commitHash[0:2]
		fileName := commitHash[2:]
		dirPath := fmt.Sprintf(".git/objects/%s", dirName)
		os.MkdirAll(dirPath, 0755)

		filePath := fmt.Sprintf(".git/objects/%s/%s", dirName, fileName)
		writeErr := writeCompressedObject(filePath, []byte(commitContent), "commit")
		if writeErr != nil {
			fmt.Fprintf(os.Stderr, "Error writing commit file: %s\n", writeErr)
			os.Exit(1)
		}

		fmt.Printf("%s\n", commitHash)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}
