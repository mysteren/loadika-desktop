package services

import (
	"errors"
	"fmt"
	"os"

	"loadika-desktop/internal/history"
)

// HistoryService отдаёт фронтенду историю завершённых загрузок.
type HistoryService struct {
	store *history.Store
}

// NewHistoryService создаёт сервис истории.
func NewHistoryService(store *history.Store) *HistoryService {
	return &HistoryService{store: store}
}

// GetHistory возвращает историю (свежие сверху).
func (s *HistoryService) GetHistory() []history.Entry {
	return s.store.List()
}

// ClearHistory очищает историю.
func (s *HistoryService) ClearHistory() error {
	return s.store.Clear()
}

// RemoveHistory удаляет одну запись истории.
func (s *HistoryService) RemoveHistory(id string) error {
	return s.store.Remove(id)
}

// RemoveHistoryWithFile удаляет запись истории вместе с её файлом (если файл есть). Файл удаляется
// до записи: если удалить файл не удалось, запись сохраняется.
func (s *HistoryService) RemoveHistoryWithFile(id string) error {
	e, ok := s.store.Get(id)
	if !ok {
		return fmt.Errorf("запись не найдена: %q", id)
	}
	if e.FilePath != "" {
		if err := os.Remove(e.FilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("удаление файла: %w", err)
		}
	}
	return s.store.Remove(id)
}
