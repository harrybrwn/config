// +build !windows

package config

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func runEditor(file string) (*exec.Cmd, error) {
	editor, err := findEditor()
	if err != nil {
		return nil, err
	}
	var cmd *exec.Cmd

	stat, err := os.Stat(file)
	if err != nil {
		return nil, err
	}
	fstat, ok := stat.Sys().(*syscall.Stat_t)

	// if we are on linux and not part of the file's user
	// or user group, then edit as root
	if ok && (fstat.Uid != uint32(os.Getuid()) && fstat.Gid != uint32(os.Getgid())) {
		fmt.Printf("running \"sudo %s %s\"\n", editor, file)
		cmd = exec.Command("sudo", editor, file)
	} else {
		cmd = exec.Command(editor, file)
	}

	return cmd, nil
}
