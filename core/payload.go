package core

import (
	"fmt"
	"io"
	"os"
)

type PayloadResolver interface {
	Resolve(data string, dataFile string, dataStdin bool, edit bool) (string, error)
}

type PayloadResolverImpl struct{}

// prefill, prefill is from file, error
func (r *PayloadResolverImpl) Resolve(data string, dataFile string, dataStdin bool, edit bool) (string, bool, error) {
	var prefill string
	fromFile := false

	switch {
	case data != "":
		prefill = data
	case dataFile != "":
		bytes, err := os.ReadFile(dataFile)
		if err != nil {
			return "", false, fmt.Errorf("failed to read data file: %w", err)
		}
		prefill = string(bytes)
		fromFile = true
	case dataStdin:
		bytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", false, fmt.Errorf("failed to read from stdin: %w", err)
		}
		prefill = string(bytes)
	default:
		prefill = data
	}

	if edit {
		editor := &EditorImpl{}
		prefill, _ = editor.Open(prefill)
	}
	return prefill, fromFile, nil
}
