package paths

import (
	"os"
	"path/filepath"
)

// appDirName — имя каталога приложения в стандартных каталогах ОС.
const appDirName = "loadika-desktop"

// ConfigDir возвращает каталог настроек приложения в стандартном месте ОС:
//
//	Linux:   ~/.config/loadika-desktop
//	Windows: %APPDATA%\loadika-desktop
//	macOS:   ~/Library/Application Support/loadika-desktop
func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appDirName), nil
}

// ConfigFile возвращает путь к файлу настроек приложения.
func ConfigFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// HistoryFile возвращает путь к файлу истории загрузок.
func HistoryFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.json"), nil
}

// ProxiesFile возвращает путь к файлу пула прокси.
func ProxiesFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "proxies.json"), nil
}

// DefaultDownloadsDir возвращает папку загрузок по умолчанию (~/Downloads).
func DefaultDownloadsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Downloads"), nil
}

// DepsDir возвращает каталог управляемых зависимостей (yt-dlp и др.).
func DepsDir() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "deps"), nil
}

// LogsDir возвращает каталог логов задач загрузки.
func LogsDir() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "logs"), nil
}
