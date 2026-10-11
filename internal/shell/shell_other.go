//go:build !windows

package shell

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

func open(path string) error {
	if runtime.GOOS == "darwin" {
		return start(exec.Command("open", path))
	}
	return start(exec.Command("xdg-open", path))
}

func reveal(path string) error {
	if runtime.GOOS == "darwin" {
		return start(exec.Command("open", "-R", path))
	}
	return start(exec.Command("xdg-open", filepath.Dir(path)))
}

// start запускает команду, но не ждёт её завершения: окно/приложение должно открываться
// независимо от нашего процесса. Дочерний процесс «усыновляется» отдельной горутиной,
// чтобы не оставлять зомби (она живёт недолго и завершается вместе с командой).
func start(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("запуск %q: %w", cmd.Path, err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
