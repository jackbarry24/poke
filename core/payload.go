package core

import (
	"fmt"
	"io"
	"os"
)

type PayloadResolver interface {
	Resolve(data, dataFile string, dataStdin, edit bool) (string, error)
}

type PayloadResolverImpl struct{}

func (r *PayloadResolverImpl) Resolve(data, dataFile string, dataStdin, edit bool) (string, error) {
	sourceCount := 0
	if data != "" {
		sourceCount++
	}
	if dataFile != "" {
		sourceCount++
	}
	if dataStdin {
		sourceCount++
	}
	if sourceCount > 1 {
		return "", fmt.Errorf("only one of --data, --data-file, or --data-stdin can be used")
	}

	var prefill string

	switch {
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
		return editor.Open(prefill)
	}
	return prefill, nil
}
