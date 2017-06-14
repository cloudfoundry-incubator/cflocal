// +build !windows

package main_test

import (
	"os/exec"
	"syscall"
)

func setpgid(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func kill(cmd *exec.Cmd) {
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGINT); err != nil {
		panic(err)
	}
}
