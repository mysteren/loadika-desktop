package services

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"

	"loadika-desktop/internal/browsers"
	"loadika-desktop/internal/config"
)

// Settings — DTO настроек для фронтенда.
type Settings struct {
	DownloadsDir   string `json:"downloadsDir"`
	YtDlpChannel   string `json:"ytDlpChannel"`
	ProxyStrategy  string `json:"proxyStrategy"`
	UseProxy       bool   `json:"useProxy"`       // использовать пул прокси для загрузок
	CookiesBrowser string `json:"cookiesBrowser"` // браузер для --cookies-from-browser ("" — не использовать)
	CookiesFile    string `json:"cookiesFile"`    // путь к файлу cookies ("" — не использовать)
}

// SettingsService предоставляет фронтенду чтение и изменение настроек.
type SettingsService struct {
	store *config.Store
}

// NewSettingsService создаёт сервис настроек поверх хранилища.
func NewSettingsService(store *config.Store) *SettingsService {
	return &SettingsService{store: store}
}

// GetSettings возвращает текущие настройки.
func (s *SettingsService) GetSettings() Settings {
	cfg := s.store.Get()
	return Settings{
		DownloadsDir:   cfg.DownloadsDir,
		YtDlpChannel:   cfg.YtDlpChannel,
		ProxyStrategy:  cfg.ProxyStrategy,
		UseProxy:       cfg.UseProxy,
		CookiesBrowser: cfg.CookiesBrowser,
		CookiesFile:    cfg.CookiesFile,
	}
}

// SetDownloadsDir задаёт папку загрузки.
func (s *SettingsService) SetDownloadsDir(dir string) error {
	return s.store.Update(func(c *config.Config) { c.DownloadsDir = dir })
}

// SetYtDlpChannel задаёт канал обновления yt-dlp.
func (s *SettingsService) SetYtDlpChannel(channel string) error {
	return s.store.Update(func(c *config.Config) { c.YtDlpChannel = channel })
}

// SetUseProxy включает/выключает использование пула прокси для загрузок.
func (s *SettingsService) SetUseProxy(enabled bool) error {
	return s.store.Update(func(c *config.Config) { c.UseProxy = enabled })
}

// SetCookiesBrowser задаёт браузер для импорта cookies (`--cookies-from-browser`).
// Пустая строка — отключить импорт из браузера. Допускается формат `браузер:профиль`.
func (s *SettingsService) SetCookiesBrowser(browser string) error {
	return s.store.Update(func(c *config.Config) { c.CookiesBrowser = browser })
}

// GetCookieBrowsers возвращает обнаруженные браузеры с сохранёнными cookies и их профили — для
// выбора в настройках (имена совместимы с `--cookies-from-browser`).
func (s *SettingsService) GetCookieBrowsers() []browsers.Browser {
	return browsers.Detect()
}

// SetCookiesFile задаёт путь к файлу cookies (`--cookies`). Пустая строка — отключить.
func (s *SettingsService) SetCookiesFile(path string) error {
	return s.store.Update(func(c *config.Config) { c.CookiesFile = path })
}

// PickCookiesFile открывает нативный диалог выбора файла cookies, сохраняет выбор и возвращает
// путь. Если пользователь отменил выбор — возвращает текущий путь.
func (s *SettingsService) PickCookiesFile() (string, error) {
	current := s.store.Get().CookiesFile

	selected, err := application.Get().Dialog.
		OpenFileWithOptions(&application.OpenFileDialogOptions{
			CanChooseDirectories: false,
			CanChooseFiles:       true,
			Title:                "Выберите файл cookies (формат Netscape)",
			Filters: []application.FileFilter{
				{DisplayName: "Cookies (*.txt)", Pattern: "*.txt"},
				{DisplayName: "Все файлы", Pattern: "*.*"},
			},
		}).
		PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("диалог выбора файла cookies: %w", err)
	}

	// Пустая строка — пользователь отменил диалог: ничего не меняем.
	if selected == "" {
		return current, nil
	}

	if err := s.store.Update(func(c *config.Config) { c.CookiesFile = selected }); err != nil {
		return "", err
	}
	return selected, nil
}

// PickDownloadsDir открывает нативный диалог выбора папки, сохраняет выбор и
// возвращает путь. Если пользователь отменил выбор — возвращает текущую папку.
func (s *SettingsService) PickDownloadsDir() (string, error) {
	current := s.store.Get().DownloadsDir

	selected, err := application.Get().Dialog.
		OpenFileWithOptions(&application.OpenFileDialogOptions{
			CanChooseDirectories: true,
			CanChooseFiles:       false,
			CanCreateDirectories: true,
			Title:                "Выберите папку для загрузок",
		}).
		SetDirectory(current).
		PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("диалог выбора папки: %w", err)
	}

	// Пустая строка — пользователь отменил диалог: ничего не меняем.
	if selected == "" {
		return current, nil
	}

	if err := s.store.Update(func(c *config.Config) { c.DownloadsDir = selected }); err != nil {
		return "", err
	}
	return selected, nil
}
