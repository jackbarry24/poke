package core

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/term"
)

type Editor interface {
	Open(initial string) ([]byte, error)
}

type EditorImpl struct{}

func (e *EditorImpl) Open(initial []byte) ([]byte, error) {
	tmp, err := os.CreateTemp("", "poke_edit_*.tmp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if len(initial) > 0 {
		if _, err := tmp.Write(initial); err != nil {
			return nil, fmt.Errorf("failed to write to temp file: %w", err)
		}
		tmp.Sync()
	}
	tmp.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "vim"
		}
	}

	var cmd *exec.Cmd
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		var in, out *os.File
		if runtime.GOOS == "windows" {
			in, err = os.OpenFile("CONIN$", os.O_RDWR, 0)
			if err != nil {
				return nil, fmt.Errorf("failed to open CONIN$: %w", err)
			}
			out, err = os.OpenFile("CONOUT$", os.O_RDWR, 0)
			if err != nil {
				in.Close()
				return nil, fmt.Errorf("failed to open CONOUT$: %w", err)
			}
		} else {
			in, err = os.OpenFile("/dev/tty", os.O_RDWR, 0)
			if err != nil {
				return nil, fmt.Errorf("failed to open /dev/tty: %w", err)
			}
			out = in
		}
		defer in.Close()
		if out != in {
			defer out.Close()
		}

		cmd = exec.Command(editor, tmp.Name())
		cmd.Stdin = in
		cmd.Stdout = out
		cmd.Stderr = out

	} else {
		cmd = exec.Command(editor, tmp.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("editor exited with error: %w", err)
	}

	edited, err := os.ReadFile(tmp.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read edited file: %w", err)
	}
	return edited, nil
}
