package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"poke/util"
)

type CollectionHandler interface {
	ListAll() error
	List(name string) error
	Send(name string, verbose bool) error
}

type DefaultCollectionHandlerImpl struct{}

func (c *DefaultCollectionHandlerImpl) ListAll() error {
	dirs := []string{
		filepath.Join(userHomeDir(), ".poke", "collections"),
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
	return nil
}

func (c *DefaultCollectionHandlerImpl) List(name string) error {
	collectionPath, err := resolveCollectionPath(name)
	if err != nil {
		return fmt.Errorf("collection '%s' not found: %w", name, err)
	}

	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("Collection %s from %s\n", name, collectionPath)
	fmt.Println(strings.Repeat("-", 40))

	err = filepath.Walk(collectionPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".json") {
			rel, _ := filepath.Rel(collectionPath, path)
			fmt.Printf("  - %s\n", rel)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error listing collection: %v\n", err)
	}
	fmt.Println(strings.Repeat("=", 40))
	return nil
}

func (c *DefaultCollectionHandlerImpl) Send(name string, verbose bool) error {
	runner := &DefaultRequestRunnerImpl{}
	resolver := &DefaultPayloadResolverImpl{}

	// Is this a single .json file?
	if strings.HasSuffix(name, ".json") {
		path := resolveSingleFilePath(name)
		if _, err := os.Stat(path); err == nil {
			req, err := runner.Load(path)
			if err != nil {
				return fmt.Errorf("failed to load request: %w", err)
			}
			body, err := resolver.Resolve(req.Body, req.BodyFile, req.BodyStdin, false)
			if err != nil {
				return fmt.Errorf("failed to resolve payload: %w", err)
			}
			req.Body = body
			req.BodyFile = ""
			req.BodyStdin = false
			return runner.Execute(req, verbose)
		}
		// If the file doesn't exist, fall through to collection resolution
	}

	// Try treating as a collection
	paths, err := resolveCollectionFilePaths(name)
	if err != nil {
		return fmt.Errorf("could not resolve collection: %w", err)
	}
	if len(paths) == 0 {
		return fmt.Errorf("no .json files found for '%s'", name)
	}

	for i, path := range paths {
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("Request %d/%d: %s\n", i+1, len(paths), path)
		fmt.Println(strings.Repeat("-", 40))

		req, err := runner.Load(path)
		if err != nil {
			fmt.Printf("File '%s' is not a valid request: %v\n", path, err)
			continue
		}

		payloadResolver := &DefaultPayloadResolverImpl{}
		body, err := payloadResolver.Resolve(req.Body, req.BodyFile, req.BodyStdin, false)
		if err != nil {
			fmt.Printf("Failed to resolve body for '%s': %v\n", path, err)
			continue
		}
		req.Body = body
		req.BodyFile = ""
		req.BodyStdin = false

		err = runner.Execute(req, verbose)
		if err != nil {
			fmt.Printf("Request failed: %v\n", err)
		}
		fmt.Println()
	}
	return nil
}

func resolveCollectionFilePaths(input string) ([]string, error) {
	path := filepath.Join(".", ".poke", input)
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		path = filepath.Join(userHomeDir(), ".poke", "collections", input)
		info, err = os.Stat(path)
	}
	if err != nil {
		return nil, err
	}

	var paths []string
	if info.IsDir() {
		err := filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !fi.IsDir() && strings.HasSuffix(fi.Name(), ".json") {
				paths = append(paths, p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		paths = append(paths, path)
	}
	return paths, nil
}

func resolveCollectionPath(name string) (string, error) {
	candidates := []string{
		filepath.Join(userHomeDir(), ".poke", "collections", name),
		filepath.Join(".", ".poke", name),
	}
	for _, p := range candidates {
		info, err := os.Stat(p)
		if err == nil && info.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("collection not found")
}

func resolveSingleFilePath(input string) string {
	if strings.Contains(input, "/") {
		return input
	}
	home := userHomeDir()
	return filepath.Join(home, ".poke", input)
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		util.Error("Could not determine home directory", err)
	}
	return home
}
