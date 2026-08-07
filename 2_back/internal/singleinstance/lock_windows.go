//go:build windows

package singleinstance

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func processAlive(pid int) bool {
	out, err := exec.Command(
		"powershell.exe",
		"-NoProfile",
		"-Command",
		fmt.Sprintf("(Get-Process -Id %d -ErrorAction SilentlyContinue) -ne $null", pid),
	).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "True"
}

func killProcess(pid int) error {
	cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Run()
	return nil
}
