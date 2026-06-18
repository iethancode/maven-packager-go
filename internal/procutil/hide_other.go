//go:build !windows

package procutil

import "os/exec"

// HideWindow 在非 Windows 平台是空实现。
func HideWindow(_ *exec.Cmd) {}
