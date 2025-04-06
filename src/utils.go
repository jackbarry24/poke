package main

import (
	// "io"
	// "net/http"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"io"

	"github.com/TylerBrock/colorjson"
	"github.com/fatih/color"
)


func saveRequest(path string, req *SavedRequest, data string) error {
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

func loadSavedRequest(path string) (*SavedRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var req SavedRequest
	err = json.Unmarshal(data, &req)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(req.Body, "@") {
		payloadPath := strings.TrimPrefix(req.Body, "@")
		contents, err := os.ReadFile(payloadPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read payload file: %v", err)
		}
		req.Body = string(contents)
	}
	return &req, nil
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

func printBody(body []byte, contentType string) {
	if strings.Contains(contentType, "application/json") {
		var obj interface{}
		err := json.Unmarshal(body, &obj)
		if err != nil {
			fmt.Println(string(body)) // fallback raw
			return
		}
		f := colorjson.NewFormatter()
		f.Indent = 2
		s, _ := f.Marshal(obj)
		fmt.Println(string(s))
	} else {
		fmt.Println(string(body))
	}
}

func resolvePayload(data string, editor bool) string {
	var prefill string

	if strings.HasPrefix(data, "@") {
		path := strings.TrimPrefix(data, "@")

		if path == "-" {
			input, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read from stdin: %v\n", err)
				os.Exit(1)
			}
			prefill = string(input)
		} else {
			fileBytes, err := os.ReadFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read payload file %s: %v\n", path, err)
				os.Exit(1)
			}
			prefill = string(fileBytes)
		}
	} else {
		prefill = data
	}

	if editor {
		edited, err := openEditorWithContent(prefill)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open editor: %v\n", err)
			os.Exit(1)
		}
		return edited
	}

	return prefill
}

func openEditorWithContent(initial string) (string, error) {
	tmpfile, err := os.CreateTemp("", "poke_edit_*.tmp")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpfile.Name())

	if initial != "" {
		tmpfile.WriteString(initial)
		tmpfile.Sync()
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	cmd := exec.Command(editor, tmpfile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	editedContent, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(editedContent)), nil
}

func Error(msg string, err error) {
	if err == nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	} else {	
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	}
	os.Exit(1)
}
