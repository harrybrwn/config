package config

import "os/exec"

func runEditor(file string) (*exec.Cmd, error) {
	editor, err := findEditor()
	if err != nil {
		return nil, err
	}
	return exec.Command(editor, file), nil
}
