package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveCollectionFilePaths returns a slice of JSON file paths from the given collection directory.
func resolveCollectionFilePaths(input string) []string {
	// Attempt to resolve the collection path from the current directory first.
	path := filepath.Join(".", ".poke", input)
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		// If not found, try resolving from the home directory.
		home, homeErr := os.UserHomeDir()
		if homeErr == nil {
			path = filepath.Join(home, ".poke", "collections", input)
			info, err = os.Stat(path)
		}
	}
	if err != nil {
		return []string{}
	}

	var filepaths []string
	if info.IsDir() {
		// Walk the directory and collect .json files.
		filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".json") {
				filepaths = append(filepaths, p)
			}
			return nil
		})
	} else {
		// It's a file, so just return it.
		filepaths = append(filepaths, path)
	}
	return filepaths
}

// sendCollection iterates over each JSON file in the collection and runs the request.
func sendCollection(filepaths []string, verbose bool) {
	for i, path := range filepaths {
		fmt.Printf("Request %d/%d: %s\n", i+1, len(filepaths), path)
		fmt.Println(strings.Repeat("-", 40))
		req, err := loadRequest(path)
		if err != nil {
			fmt.Printf("Error loading %s: %v\n", path, err)
			continue
		}
		runRequest(req, verbose)
		fmt.Println(strings.Repeat("-", 40))
	}
}

// listCollections prints all collection directories from ~/.poke/collections and ./.poke.
func listCollections() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error finding home directory:", err)
		return
	}
	dirs := []string{
		filepath.Join(home, ".poke", "collections"),
		filepath.Join(".", ".poke"),
	}
	fmt.Println("Available Collections:")
	fmt.Println(strings.Repeat("=", 40))
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		fmt.Printf("From: %s\n", dir)
		fmt.Println(strings.Repeat("-", 40))
		for _, entry := range entries {
			if entry.IsDir() {
				fmt.Printf("  - %s\n", entry.Name())
			}
		}
		fmt.Println(strings.Repeat("-", 40))
	}
}

// listCollection lists all JSON files in the specified collection (recursively).
func listCollection(collectionName string) {
	collectionPath, err := resolveCollectionPath(collectionName)
	if err != nil {
		fmt.Printf("Collection '%s' not found: %v\n", collectionName, err)
		return
	}
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("JSON files in collection '%s':\n", collectionName)
	fmt.Println(strings.Repeat("-", 40))
	err = filepath.Walk(collectionPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".json") {
			relPath, _ := filepath.Rel(collectionPath, path)
			fmt.Printf("  - %s\n", relPath)
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error listing collection:", err)
	}
	fmt.Println(strings.Repeat("=", 40))
}

// resolveCollectionPath finds a collection directory from ~/.poke/collections or ./.poke.
func resolveCollectionPath(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	paths := []string{
		filepath.Join(home, ".poke", "collections", name),
		filepath.Join(".", ".poke", name),
	}
	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("collection not found")
}
