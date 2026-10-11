package deps

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// manifestEntry описывает установленную зависимость.
type manifestEntry struct {
	Version   string `json:"version"`
	Channel   string `json:"channel"`
	Asset     string `json:"asset"`
	SHA256    string `json:"sha256"`
	UpdatedAt string `json:"updatedAt"` // RFC3339
}

// manifest сопоставляет имя зависимости и запись об установке.
type manifest map[string]manifestEntry

// loadManifest читает манифест; при отсутствии или повреждении возвращает пустой.
func loadManifest(path string) manifest {
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest{}
	}
	mf := manifest{}
	if err := json.Unmarshal(data, &mf); err != nil {
		return manifest{}
	}
	return mf
}

// saveManifest атомарно записывает манифест (temp + rename).
func saveManifest(path string, mf manifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("создание каталога манифеста: %w", err)
	}

	data, err := json.MarshalIndent(mf, "", "  ")
	if err != nil {
		return fmt.Errorf("сериализация манифеста: %w", err)
	}
	data = append(data, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("запись манифеста: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("фиксация манифеста: %w", err)
	}
	return nil
}
