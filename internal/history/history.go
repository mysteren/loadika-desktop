// Package history хранит записи о завершённых задачах загрузки (JSON-файл) и отдаёт их
// интерфейсу. Не зависит от Wails и от домена загрузок; безопасен для конкурентного доступа.
package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// CurrentSchemaVersion — версия схемы файла истории.
const CurrentSchemaVersion = 1

// DefaultMaxEntries — сколько последних записей храним (защита от неограниченного роста).
const DefaultMaxEntries = 500

// Entry — запись о завершённой задаче загрузки.
type Entry struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Title      string `json:"title"`    // заголовок видео (чтобы понимать, что за видео)
	Duration   int64  `json:"duration"` // длительность в секундах (0 — неизвестно)
	Quality    string `json:"quality"`  // метка качества (например, "1080p" / "Только аудио")
	Details    string `json:"details"`  // расшифровка (кодеки, fps, битрейт)
	OutputDir  string `json:"outputDir"`
	Format     string `json:"format"`
	State      string `json:"state"` // completed | failed | canceled
	Error      string `json:"error"`
	FilePath   string `json:"filePath"`   // итоговый путь файла (после post-processing); пусто, если неизвестен
	LogPath    string `json:"logPath"`    // путь к логу команды yt-dlp; пусто, если логи отключены
	Proxy      string `json:"proxy"`      // использованный прокси (без пароля); пусто — напрямую
	Size       int64  `json:"size"`       // итоговый размер в байтах (0 — неизвестно)
	CreatedAt  int64  `json:"createdAt"`  // UnixMilli
	FinishedAt int64  `json:"finishedAt"` // UnixMilli
}

// file — формат файла истории.
type file struct {
	SchemaVersion int     `json:"schemaVersion"`
	Entries       []Entry `json:"entries"`
}

// Store хранит историю в памяти и синхронизирует её с файлом.
type Store struct {
	mu      sync.Mutex
	path    string
	max     int
	entries []Entry
}

// Load читает историю из файла. Если файла нет — история пустая. max — предел числа записей
// (<= 0 — значение по умолчанию).
func Load(path string, max int) (*Store, error) {
	if max <= 0 {
		max = DefaultMaxEntries
	}
	s := &Store{path: path, max: max, entries: []Entry{}}

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		var f file
		if uerr := json.Unmarshal(data, &f); uerr != nil {
			return nil, fmt.Errorf("разбор истории %q: %w", path, uerr)
		}
		s.entries = f.Entries
		if s.entries == nil {
			s.entries = []Entry{}
		}
	case errors.Is(err, os.ErrNotExist):
		// Файла ещё нет — история пустая.
	default:
		return nil, fmt.Errorf("чтение истории %q: %w", path, err)
	}
	return s, nil
}

// List возвращает копию записей (свежие сверху).
func (s *Store) List() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}

// Add добавляет запись сверху, обрезает историю до лимита и сохраняет на диск.
func (s *Store) Add(e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = append([]Entry{e}, s.entries...)
	if len(s.entries) > s.max {
		s.entries = s.entries[:s.max]
	}
	return s.saveLocked()
}

// Clear очищает историю и сохраняет пустое состояние.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = []Entry{}
	return s.saveLocked()
}

// Get возвращает запись по идентификатору.
func (s *Store) Get(id string) (Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range s.entries {
		if e.ID == id {
			return e, true
		}
	}
	return Entry{}, false
}

// Remove удаляет запись по идентификатору (только первую запись с таким id) и сохраняет изменение.
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, e := range s.entries {
		if e.ID == id {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			break
		}
	}
	return s.saveLocked()
}

// saveLocked атомарно записывает историю (вызывается при удержанном mu).
func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("создание каталога истории: %w", err)
	}

	data, err := json.MarshalIndent(file{SchemaVersion: CurrentSchemaVersion, Entries: s.entries}, "", "  ")
	if err != nil {
		return fmt.Errorf("сериализация истории: %w", err)
	}
	data = append(data, '\n')

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("запись истории: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("фиксация истории: %w", err)
	}
	return nil
}
