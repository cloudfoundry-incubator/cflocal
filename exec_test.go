// +build !windows

package main_test

import (
	"os/exec"
	"syscall"
)

func setpgid(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
