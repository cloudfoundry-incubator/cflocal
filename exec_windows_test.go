package main_test

import (
	"os/exec"
	"strconv"
	"syscall"
)

func setpgid(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func kill(cmd *exec.Cmd) {
	err := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
	if err != nil {
		panic(err)
	}
	if err := cmd.Process.Kill(); err != nil {
		panic(err)
	}
}
