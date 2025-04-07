package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

func saveRequest(path string, req *PokeRequest, data string) error {
	originalBody := req.Body
	defer func() { req.Body = originalBody }()

	// If the data starts with '@', treat it as a file path
	// save the file path in the saved request not the content
	// this allows us to edit the file later
	if strings.HasPrefix(data, "@") {
		req.Body = data
	}

	buffer, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, buffer, 0644)
}

func loadRequest(path string) (*PokeRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var req PokeRequest
	err = json.Unmarshal(data, &req)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(req.Body, "@") {
		payloadPath := strings.TrimPrefix(req.Body, "@")
		if payloadPath == "-" {
			req.Body = "@-"
		} else {
			contents, err := os.ReadFile(payloadPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read payload file: %v", err)
			}
			req.Body = string(contents)
		}
	}
	return &req, nil
}

func resolveRequestPath(input string) string {
	if strings.Contains(input, "/") {
		return input
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return input
	}

	dir := filepath.Join(home, ".poke")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, input)
}

func mergeHeaders(base map[string]string, newHeaders map[string]string) {
	for k, v := range newHeaders {
		base[k] = v
	}
}

func parseHeaders(headerStr string) map[string]string {
	headers := make(map[string]string)
	if headerStr == "" {
		return headers
	}
	pairs := strings.Split(headerStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			headers[key] = value
		}
	}
	return headers
}

func colorStatus(code int) string {
	switch {
	case code >= 200 && code < 300:
		return color.New(color.FgGreen).Sprintf("%d OK", code)
	case code >= 300 && code < 400:
		return color.New(color.FgYellow).Sprintf("%d Redirect", code)
	case code >= 400:
		return color.New(color.FgRed).Sprintf("%d Error", code)
	default:
		return fmt.Sprintf("%d", code)
	}
}

func resolvePayload(data, dataFile string, dataStdin, editor bool) string {
	count := 0
	if data != "" {
		count++
	}
	if dataFile != "" {
		count++
	}
	if dataStdin {
		count++
	}
	if count > 1 {
		Error("Only one of --data, --data-file, or --data-stdin can be used", nil)
	}

	var prefill string
	if dataFile != "" {
		bytes, err := os.ReadFile(dataFile)
		if err != nil {
			Error("Failed to read file", err)
		}
		prefill = string(bytes)
	} else if dataStdin {
		bytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			Error("Failed to read stdin", err)
		}
		prefill = string(bytes)
	} else {
		prefill = data
	}

	if editor {
		edited, err := openEditorWithContent(prefill)
		if err != nil {
			Error("Failed to open editor", err)
		}
		return edited
	}
	return prefill
}

func Error(msg string, err error) {
	if err == nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	} else {
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	}
	os.Exit(1)
}
