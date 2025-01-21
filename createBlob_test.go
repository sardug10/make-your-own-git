package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func RunGitHashObject(filePath string) (string, error) {
	cmd := exec.Command("git", "hash-object", "-w", filePath)

	// Capture stdout
	var out bytes.Buffer
	cmd.Stdout = &out

	// Run the command
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	// Return the hash (trim whitespace)
	return out.String(), nil
}

func RunMainFuncWithHashObject(fileName string) (string, error) {
	cmd := exec.Command("go", "run", "main.go", "hash-object", "-w", fileName)

	// Capture stdout
	var out bytes.Buffer
	cmd.Stdout = &out

	// Run the command
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	// Return the hash (trim whitespace)
	return out.String(), nil
}


func TestHashObject (t *testing.T) {
	// Create a file with some content
	fileName := "text.txt"
	fileContents := []byte("Hello, World!")

	if err := os.WriteFile(fileName, fileContents, 0644); err != nil {
		t.Fatalf("Error writing to test file: %s\n", err)
	}

	// Run the hash-object command
	wantHash, gitErr := RunGitHashObject(fileName)
	if gitErr != nil {
		t.Fatalf("Error implementing git command: %s\n", gitErr)
	}

	// call main function with the hash-object command
	gotHash, myGitErr := RunMainFuncWithHashObject(fileName)
	if myGitErr != nil {
		t.Fatalf("Error implementing mygit command: %s\n", myGitErr)
	}

	t.Run("Testing Hash creation", func(t *testing.T) {
		if gotHash != wantHash {
			t.Errorf("got %q want %q", gotHash, wantHash)
		}
	})

	t.Run("Testing the blob object creation", func(t *testing.T) {
		gotHash = strings.TrimSpace(gotHash)
		filePath := fmt.Sprintf(".mygit/objects/%s/%s", gotHash[0:2], gotHash[2:])
		// read the file
		_, err := os.Stat(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Error finding file: ", err)
				os.Exit(1)
			}
		}
	})
}