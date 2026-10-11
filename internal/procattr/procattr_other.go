//go:build !windows

package procattr

import "syscall"

// HideConsole на не-Windows не требуется: атрибутов нет.
func HideConsole() *syscall.SysProcAttr {
	return nil
}
