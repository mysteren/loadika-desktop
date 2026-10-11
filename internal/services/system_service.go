package services

import (
	"errors"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

	"loadika-desktop/internal/paths"
	"loadika-desktop/internal/shell"
)

// SystemService выполняет системные действия над путями (открыть файл/папку).
type SystemService struct{}

// NewSystemService создаёт сервис системных действий.
func NewSystemService() *SystemService {
	return &SystemService{}
}

// OpenPath открывает файл или каталог приложением по умолчанию.
func (s *SystemService) OpenPath(path string) error {
	if path == "" {
		return errors.New("не задан путь")
	}
	return shell.Open(path)
}

// RevealPath открывает файловый менеджер с выделением файла (на Linux — содержащий каталог).
func (s *SystemService) RevealPath(path string) error {
	if path == "" {
		return errors.New("не задан путь")
	}
	return shell.Reveal(path)
}

// OpenDownloadsDir открывает папку загрузок в проводнике. Принимает пустой путь: если UI не
// передал папку (или она недоступна), раскрываем «Показать в папке» сам — иначе кнопка молча
// отправляла пустоту и Windows открывала корневой вид («Этот компьютер» с дисками).
func (s *SystemService) OpenDownloadsDir(dir string) error {
	if dir != "" {
		if _, err := os.Stat(dir); err == nil {
			return shell.Open(dir)
		}
	}
	fallback, ferr := paths.DefaultDownloadsDir()
	if ferr != nil {
		return fmt.Errorf("папка загрузок недоступна (%q): %w", dir, ferr)
	}
	if err := os.MkdirAll(fallback, 0o755); err != nil {
		return fmt.Errorf("создать папку загрузок %q: %w", fallback, err)
	}
	return shell.Open(fallback)
}

// CopyText помещает текст в системный буфер обмена (для кнопки «Копировать»).
func (s *SystemService) CopyText(text string) error {
	if app := application.Get(); app != nil {
		app.Clipboard.SetText(text)
	}
	return nil
}
