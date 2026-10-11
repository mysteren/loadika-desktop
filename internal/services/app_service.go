package services

import (
	"runtime"

	"loadika-desktop/internal/version"
)

// AppInfo описывает запущенное приложение для отображения в интерфейсе.
type AppInfo struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Platform string `json:"platform"`
}

// AppService предоставляет фронтенду базовые метаданные приложения.
type AppService struct{}

// GetAppInfo возвращает базовые метаданные приложения.
func (s *AppService) GetAppInfo() AppInfo {
	return AppInfo{
		Name:     "Loadika",
		Version:  version.Version,
		Platform: runtime.GOOS,
	}
}
