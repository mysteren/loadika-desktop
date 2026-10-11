// Package shell открывает файлы и каталоги средствами ОС: приложением по умолчанию и
// файловым менеджером. Не зависит от Wails и домена загрузок.
package shell

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ErrEmptyPath возвращается при пустом пути.
var ErrEmptyPath = errors.New("не задан путь")

// Open открывает путь (файл или каталог) приложением по умолчанию.
func Open(path string) error {
	if path == "" {
		return ErrEmptyPath
	}

	switch runtime.GOOS {
	case "windows":
		// rundll32 не требует shell и корректно обрабатывает и файлы, и каталоги.
		return start(exec.Command("rundll32", "url.dll,FileProtocolHandler", path))
	case "darwin":
		return start(exec.Command("open", path))
	default:
		return start(exec.Command("xdg-open", path))
	}
}

// Reveal открывает файловый менеджер с выделением файла (где это поддерживает ОС).
// На Linux стандартного «выделения» нет — открываем содержащий каталог.
func Reveal(path string) error {
	if path == "" {
		return ErrEmptyPath
	}

	switch runtime.GOOS {
	case "windows":
		// explorer возвращает ненулевой код даже при успехе — поэтому не ждём завершения.
		return start(exec.Command("explorer", "/select,"+path))
	case "darwin":
		return start(exec.Command("open", "-R", path))
	default:
		return start(exec.Command("xdg-open", filepath.Dir(path)))
	}
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
