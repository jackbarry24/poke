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

func (r *PayloadResolverImpl) Resolve(data string, dataFile string, dataStdin bool, edit bool) ([]byte, error) {
	var prefill []byte

	switch {
	case data != "":
		prefill = []byte(data)
	case dataFile != "":
		bytes, err := os.ReadFile(dataFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read data file: %w", err)
		}
		prefill = bytes
	case dataStdin:
		bytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read from stdin: %w", err)
		}
		prefill = bytes
	default:
		prefill = []byte(data)
	}

	if edit {
		editor := &EditorImpl{}
		prefill, _ = editor.Open(prefill)
	}
	return prefill, nil
}
