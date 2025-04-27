package core

import (
	"fmt"
	"io"
	"os"
)

type PayloadResolver interface {
	Resolve(data, dataFile string, dataStdin, edit bool) ([]byte, error)
}

type PayloadResolverImpl struct{}

func (r *PayloadResolverImpl) Resolve(data string, dataFile string, dataStdin bool, edit bool) (string, error) {
	var prefill string

	switch {
	case data != "":
		prefill = data
	case dataFile != "":
		bytes, err := os.ReadFile(dataFile)
		if err != nil {
			return "", fmt.Errorf("failed to read data file: %w", err)
		}
		prefill = string(bytes)
	case dataStdin:
		bytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		prefill = string(bytes)
	default:
		prefill = data
	}

	if edit {
		editor := &EditorImpl{}
		prefill, _ = editor.Open(prefill)
	}
	return prefill, nil
}
