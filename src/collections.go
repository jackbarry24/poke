package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolveCollectionFilePaths(input string) []string {
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
		filepaths = append(filepaths, path)
	}
	return filepaths
}

func sendCollection(filepaths []string, verbose bool) {
	for i, path := range filepaths {
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("Request %d/%d: %s\n", i+1, len(filepaths), path)
		fmt.Println(strings.Repeat("-", 40))
		req, err := loadRequest(path)
		if err != nil {
			fmt.Printf("File '%s' is not a valid request: %v\n", path, err)
		} else {
			RunRequest(req, verbose)
		}
		fmt.Println()
	}
}

func ListCollections() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error finding home directory: %v\n", err)
		return
	}

	dirs := []string{
		filepath.Join(home, ".poke", "collections"),
		filepath.Join(".", ".poke"),
	}

	fmt.Println("======== Available Collections =========")
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		fmt.Printf("\nFrom: %s\n", dir)
		fmt.Println(strings.Repeat("-", 40))
		for _, entry := range entries {
			if entry.IsDir() {
				fmt.Printf("   - %s\n", entry.Name())
			}
		}
	}
	fmt.Println(strings.Repeat("=", 40))
}

func ListCollection(collectionName string) {
	collectionPath, err := resolveCollectionPath(collectionName)
	if err != nil {
		fmt.Printf("Collection '%s' not found: %v\n", collectionName, err)
		return
	}
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("Collection %s from %s\n", collectionName, collectionPath)
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
