//go:build windows

// Package procutil 封装跨平台进程控制细节。
package procutil

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// HideWindow 在 Windows 上阻止 exec.Cmd 弹出终端窗口。
func HideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}
