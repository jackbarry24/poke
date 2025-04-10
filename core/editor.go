package core

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

type Editor interface {
	Open(initial string) (string, error)
}

type EditorImpl struct{}

func (e *EditorImpl) Open(initial string) (string, error) {
	tmpfile, err := os.CreateTemp("", "poke_edit_*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	if initial != "" {
		if _, err := tmpfile.WriteString(initial); err != nil {
			return "", fmt.Errorf("failed to write to temp file: %w", err)
		}
		tmpfile.Sync()
	}
	tmpfile.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	var cmd *exec.Cmd
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			return "", fmt.Errorf("failed to open /dev/tty: %w", err)
		}
		defer tty.Close()
		cmd = exec.Command(editor, tmpfile.Name())
		cmd.Stdin = tty
		cmd.Stdout = tty
		cmd.Stderr = tty
	} else {
		cmd = exec.Command(editor, tmpfile.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	editedBytes, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read edited file: %w", err)
	}

	return strings.TrimSpace(string(editedBytes)), nil
}
