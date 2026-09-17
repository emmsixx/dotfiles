package ui

import (
	"os"
	"testing"
)

func TestDevNullIsNotInteractive(t *testing.T) {
	file, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	original := os.Stdin
	os.Stdin = file
	defer func() { os.Stdin = original }()
	if Interactive() {
		t.Fatal("/dev/null was treated as a terminal")
	}
}
