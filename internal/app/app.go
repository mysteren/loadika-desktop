package app

import (
	"io/fs"

	"github.com/wailsapp/wails/v3/pkg/application"

	"loadika-desktop/internal/config"
	"loadika-desktop/internal/deps"
	"loadika-desktop/internal/download"
	"loadika-desktop/internal/history"
	"loadika-desktop/internal/paths"
	"loadika-desktop/internal/proxy"
	"loadika-desktop/internal/services"
)

const (
	appName        = "Loadika"
	appDescription = "Десктопная обёртка над yt-dlp с поддержкой пула прокси"

	// linuxAppID — идентификатор приложения для Linux (GtkApplication id).
	// На Linux иконку окна задаёт не приложение, а рабочий стол: он сопоставляет
	// окно с .desktop-файлом по этому id. Поэтому имя .desktop и иконки в теме
	// должно совпадать с linuxAppID (см. build/linux/*).
	linuxAppID = "ru.mysterentc.loadika-desktop"
)

// Run создаёт приложение Wails, открывает главное окно и блокируется до завершения
// работы приложения. assets — встроенная сборка фронтенда, icon — иконка приложения (PNG).
func Run(assets fs.FS, icon []byte) error {
	cfgStore, err := loadConfig()
	if err != nil {
		return err
	}

	depsDir, err := paths.DepsDir()
	if err != nil {
		return err
	}
	depsManager := deps.NewManager(depsDir)
	depsManager.SetPolicyResolver(func(name string) string {
		return cfgStore.Get().DepSources[name]
	})

	downloadManager := download.NewManager()

	logsDir, err := paths.LogsDir()
	if err != nil {
		return err
	}
	// Логи команд yt-dlp — рядом с настройками, чтобы их можно было открыть из UI.
	downloadManager.SetLogDir(logsDir)

	historyPath, err := paths.HistoryFile()
	if err != nil {
		return err
	}
	historyStore, err := history.Load(historyPath, history.DefaultMaxEntries)
	if err != nil {
		return err
	}

	proxiesPath, err := paths.ProxiesFile()
	if err != nil {
		return err
	}
	proxyStore, err := proxy.Load(proxiesPath)
	if err != nil {
		return err
	}
	proxyPool := proxy.NewPool(proxyStore)
	proxyPool.SetStrategy(cfgStore.Get().ProxyStrategy)
	downloadManager.SetProxyPool(proxyPool)

	app := application.New(application.Options{
		Name:        appName,
		Description: appDescription,
		Icon:        icon,
		Linux: application.LinuxOptions{
			ApplicationID: linuxAppID,
		},
		Services: []application.Service{
			application.NewService(&services.AppService{}),
			application.NewService(services.NewSettingsService(cfgStore)),
			application.NewService(services.NewDepsService(depsManager, cfgStore)),
			application.NewService(services.NewDownloadService(downloadManager, depsManager, cfgStore, historyStore, proxyPool)),
			application.NewService(services.NewHistoryService(historyStore)),
			application.NewService(services.NewSystemService()),
			application.NewService(services.NewProxyService(proxyPool, proxyStore, cfgStore)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            appName,
		Width:            1100,
		Height:           720,
		BackgroundColour: application.NewRGB(24, 24, 27),
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		URL: "/",
	})

	// При выходе корректно останавливаем все выполняющиеся загрузки (без «висящих» процессов).
	app.OnShutdown(downloadManager.Shutdown)

	return app.Run()
}

// loadConfig загружает настройки из стандартного каталога ОС, подставляя значения
// по умолчанию (в частности — папку загрузок ~/Downloads).
func loadConfig() (*config.Store, error) {
	configPath, err := paths.ConfigFile()
	if err != nil {
		return nil, err
	}
	defaultsDir, err := paths.DefaultDownloadsDir()
	if err != nil {
		return nil, err
	}

	defaults := config.Config{
		SchemaVersion: config.CurrentSchemaVersion,
		DownloadsDir:  defaultsDir,
		YtDlpChannel:  config.DefaultYtDlpChannel,
		ProxyStrategy: config.DefaultProxyStrategy,
		UseProxy:      true,
		DepSources: map[string]string{
			deps.NameYtDlp:  deps.SourceAuto,
			deps.NameDeno:   deps.SourceAuto,
			deps.NameFfmpeg: deps.SourceAuto,
		},
	}
	return config.Load(configPath, defaults)
}
