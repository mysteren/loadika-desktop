package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// CurrentSchemaVersion — текущая версия схемы файла настроек.
const CurrentSchemaVersion = 1

// DefaultYtDlpChannel — канал обновления yt-dlp по умолчанию.
const DefaultYtDlpChannel = "stable"

// YtDlpChannels — допустимые каналы обновления yt-dlp.
var YtDlpChannels = []string{"stable", "nightly", "master"}

// DefaultProxyStrategy — стратегия выбора прокси по умолчанию.
const DefaultProxyStrategy = "round-robin"

// Config — сохраняемые настройки приложения.
type Config struct {
	SchemaVersion  int               `json:"schemaVersion"`
	DownloadsDir   string            `json:"downloadsDir"`
	YtDlpChannel   string            `json:"ytDlpChannel"`
	ProxyStrategy  string            `json:"proxyStrategy"`  // стратегия выбора прокси
	UseProxy       bool              `json:"useProxy"`       // использовать пул прокси для загрузок
	CookiesBrowser string            `json:"cookiesBrowser"` // браузер для --cookies-from-browser ("" — не использовать)
	CookiesFile    string            `json:"cookiesFile"`    // путь к файлу cookies ("" — не использовать)
	DepSources     map[string]string `json:"depSources"`     // политика источника по имени зависимости
}

// Store хранит настройки в памяти и синхронизирует их с файлом.
// Безопасен для конкурентного доступа.
type Store struct {
	mu   sync.RWMutex
	path string
	cfg  Config
}

// Load читает настройки из файла. Если файла нет — берёт defaults; defaults
// также используются для заполнения пропущенных полей.
func Load(path string, defaults Config) (*Store, error) {
	cfg := defaults

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if uerr := json.Unmarshal(data, &cfg); uerr != nil {
			return nil, fmt.Errorf("разбор настроек %q: %w", path, uerr)
		}
	case errors.Is(err, os.ErrNotExist):
		// Файла ещё нет — работаем со значениями по умолчанию.
	default:
		return nil, fmt.Errorf("чтение настроек %q: %w", path, err)
	}

	return &Store{path: path, cfg: withDefaults(cfg, defaults)}, nil
}

// Get возвращает текущие настройки (копию).
func (s *Store) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.clone()
}

// clone возвращает глубокую копию настроек (карта источников не шарится между вызовами).
func (c Config) clone() Config {
	clone := c
	if c.DepSources != nil {
		clone.DepSources = make(map[string]string, len(c.DepSources))
		for k, v := range c.DepSources {
			clone.DepSources[k] = v
		}
	}
	return clone
}

// Update применяет изменение к копии настроек, валидирует, сохраняет на диск и
// только затем фиксирует новое значение. При ошибке состояние не меняется.
func (s *Store) Update(fn func(*Config)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.cfg.clone()
	fn(&next)

	if err := next.validate(); err != nil {
		return err
	}
	if err := save(s.path, next); err != nil {
		return err
	}
	s.cfg = next
	return nil
}

// IsValidChannel сообщает, допустим ли канал обновления yt-dlp.
func IsValidChannel(channel string) bool {
	for _, c := range YtDlpChannels {
		if c == channel {
			return true
		}
	}
	return false
}

func (c Config) validate() error {
	if c.DownloadsDir == "" {
		return errors.New("папка загрузки не может быть пустой")
	}
	if !IsValidChannel(c.YtDlpChannel) {
		return fmt.Errorf("неизвестный канал обновления yt-dlp: %q", c.YtDlpChannel)
	}
	return nil
}

// withDefaults заполняет пропущенные поля значениями по умолчанию.
func withDefaults(cfg, defaults Config) Config {
	if cfg.SchemaVersion == 0 {
		cfg.SchemaVersion = CurrentSchemaVersion
	}
	if cfg.DownloadsDir == "" {
		cfg.DownloadsDir = defaults.DownloadsDir
	}
	if !IsValidChannel(cfg.YtDlpChannel) {
		cfg.YtDlpChannel = defaults.YtDlpChannel
	}
	if cfg.ProxyStrategy == "" {
		cfg.ProxyStrategy = defaults.ProxyStrategy
	}
	if cfg.ProxyStrategy == "" {
		cfg.ProxyStrategy = DefaultProxyStrategy
	}
	merged := make(map[string]string, len(defaults.DepSources)+len(cfg.DepSources))
	for name, def := range defaults.DepSources {
		merged[name] = def
	}
	for name, val := range cfg.DepSources {
		merged[name] = val
	}
	cfg.DepSources = merged
	return cfg
}

// save атомарно записывает настройки: сначала во временный файл, затем rename.
func save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("создание каталога настроек: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("сериализация настроек: %w", err)
	}
	data = append(data, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("запись настроек: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("фиксация настроек: %w", err)
	}
	return nil
}
