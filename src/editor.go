package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

func isTerminal(fd uintptr) bool {
	return term.IsTerminal(int(fd))
}

func openEditorWithContent(initial string) (string, error) {
	// Create a temporary file for editor input.
	tmpfile, err := os.CreateTemp("", "poke_edit_*.tmp")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpfile.Name())

	if initial != "" {
		if _, err := tmpfile.WriteString(initial); err != nil {
			return "", err
		}
		tmpfile.Sync()
	}
	tmpfile.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	var cmd *exec.Cmd
	// If stdin is not a terminal, use /dev/tty.
	if !isTerminal(os.Stdin.Fd()) {
		tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			return "", fmt.Errorf("failed to open /dev/tty: %v", err)
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
		return "", fmt.Errorf("editor exited with error: %v", err)
	}

	editedBytes, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(editedBytes)), nil
}
