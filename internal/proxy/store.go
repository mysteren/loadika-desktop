package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// CurrentSchemaVersion — версия схемы файла пула прокси.
const CurrentSchemaVersion = 1

// file — формат файла пула.
type file struct {
	SchemaVersion int     `json:"schemaVersion"`
	Proxies       []Proxy `json:"proxies"`
}

// Store хранит определения прокси в памяти и синхронизирует их с файлом. Безопасен для
// конкурентного доступа.
type Store struct {
	mu      sync.Mutex
	path    string
	proxies []Proxy
}

// Load читает пул из файла. Если файла нет — пул пустой.
func Load(path string) (*Store, error) {
	s := &Store{path: path, proxies: []Proxy{}}

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		var f file
		if uerr := json.Unmarshal(data, &f); uerr != nil {
			return nil, fmt.Errorf("разбор пула прокси %q: %w", path, uerr)
		}
		if f.Proxies != nil {
			s.proxies = f.Proxies
		}
	case errors.Is(err, os.ErrNotExist):
		// Файла ещё нет — пул пустой.
	default:
		return nil, fmt.Errorf("чтение пула прокси %q: %w", path, err)
	}
	return s, nil
}

// List возвращает копию списка прокси.
func (s *Store) List() []Proxy {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Proxy, len(s.proxies))
	copy(out, s.proxies)
	return out
}

// Add добавляет прокси, пропуская уже присутствующие (совпадение по ID) и включая только новые.
// Возвращает число фактически добавленных записей.
func (s *Store) Add(list []Proxy) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing := make(map[string]bool, len(s.proxies))
	for _, p := range s.proxies {
		existing[p.ID] = true
	}

	added := 0
	for _, p := range list {
		if p.ID == "" || existing[p.ID] {
			continue
		}
		existing[p.ID] = true
		s.proxies = append(s.proxies, p)
		added++
	}
	if added == 0 {
		return 0, nil
	}
	return added, s.saveLocked()
}

// Remove удаляет прокси по идентификатору.
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := make([]Proxy, 0, len(s.proxies))
	removed := false
	for _, p := range s.proxies {
		if p.ID == id {
			removed = true
			continue
		}
		next = append(next, p)
	}
	if !removed {
		return fmt.Errorf("прокси не найден: %q", id)
	}
	s.proxies = next
	return s.saveLocked()
}

// SetEnabled включает/выключает прокси.
func (s *Store) SetEnabled(id string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.proxies {
		if s.proxies[i].ID == id {
			s.proxies[i].Enabled = enabled
			return s.saveLocked()
		}
	}
	return fmt.Errorf("прокси не найден: %q", id)
}

// Clear очищает пул.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.proxies = []Proxy{}
	return s.saveLocked()
}

// saveLocked атомарно записывает пул (вызывается при удержанном mu).
func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("создание каталога пула прокси: %w", err)
	}

	data, err := json.MarshalIndent(file{SchemaVersion: CurrentSchemaVersion, Proxies: s.proxies}, "", "  ")
	if err != nil {
		return fmt.Errorf("сериализация пула прокси: %w", err)
	}
	data = append(data, '\n')

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("запись пула прокси: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("фиксация пула прокси: %w", err)
	}
	return nil
}
