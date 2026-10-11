package services

import (
	"errors"

	"github.com/wailsapp/wails/v3/pkg/application"

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

// CopyText помещает текст в системный буфер обмена (для кнопки «Копировать»).
func (s *SystemService) CopyText(text string) error {
	if app := application.Get(); app != nil {
		app.Clipboard.SetText(text)
	}
	return nil
}
