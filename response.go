package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func readResponse(resp *http.Response) []byte {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	return bodyBytes
}

func resolveRequestPath(input string) string {
	if strings.Contains(input, "/") {
		return input
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return input
	}

	dir := filepath.Join(home, ".poke", "requests")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, input)
}
