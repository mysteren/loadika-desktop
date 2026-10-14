package services

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"loadika-desktop/internal/config"
	"loadika-desktop/internal/deps"
	"loadika-desktop/internal/download"
	"loadika-desktop/internal/events"
	"loadika-desktop/internal/history"
	"loadika-desktop/internal/pagetitle"
	"loadika-desktop/internal/proxy"
)

// progressThrottle — минимальный интервал между событиями прогресса (защита webview от шторма).
const progressThrottle = 100 * time.Millisecond

// formatsTimeout — предел ожидания метаданных (`-J`) для одной ссылки.
const formatsTimeout = 90 * time.Second

// preflightAttemptTimeout — таймаут проверки одного прокси в предварительной проверке.
const preflightAttemptTimeout = 3 * time.Second

// preflightTimeout — общий предел предварительной проверки прокси (все — параллельно).
const preflightTimeout = 4 * time.Second

// pageTitleTimeout — предел ожидания запасного запроса заголовка страницы.
const pageTitleTimeout = 8 * time.Second

// healthChecker проверяет прокси; вынесен в поле для офлайн-тестов.
type healthChecker func(ctx context.Context, proxies []proxy.Proxy, target string, timeout time.Duration) []proxy.HealthResult

// DownloadMeta — отображаемые метаданные загрузки (собираются во фронтенде из `-J`), чтобы в истории
// было понятно, что за видео и с каким качеством.
type DownloadMeta struct {
	Title    string `json:"title"`
	Duration int64  `json:"duration"` // длительность в секундах (0 — неизвестно)
	Quality  string `json:"quality"`  // метка качества («1080p», «Только аудио»)
	Details  string `json:"details"`  // расшифровка (кодеки, fps, битрейт)
}

// DownloadService запускает и отменяет задачи загрузки, отдаёт список форматов и пресеты.
type DownloadService struct {
	manager     *download.Manager
	depsManager *deps.Manager
	configStore *config.Store
	history     *history.Store
	pool        *proxy.Pool   // пул прокси (для предварительной проверки); может быть nil
	checkAll    healthChecker // проверка прокси; по умолчанию proxy.CheckAll
}

// NewDownloadService создаёт сервис загрузок. history может быть nil — тогда история не ведётся.
func NewDownloadService(manager *download.Manager, depsManager *deps.Manager, cfg *config.Store, historyStore *history.Store, pool *proxy.Pool) *DownloadService {
	return &DownloadService{
		manager:     manager,
		depsManager: depsManager,
		configStore: cfg,
		history:     historyStore,
		pool:        pool,
		checkAll:    proxy.CheckAll,
	}
}

// Inspect запрашивает метаданные ссылки (`yt-dlp -J`). Возвращает разобранные форматы и сырой
// ответ библиотеки (команда + stdout/stderr). useProxy — идти ли через пул прокси (значение
// конкретной загрузки; по умолчанию во фронтенде берётся из глобальной настройки).
// Ошибка команды не роняет вызов: она попадает в поле Error результата.
func (s *DownloadService) Inspect(url string, useProxy bool) (download.Inspect, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return download.Inspect{}, errors.New("не задана ссылка")
	}
	tc, err := s.toolchain()
	if err != nil {
		return download.Inspect{}, err
	}

	// Быстрая проверка прокси: отсеиваем мёртвые узлы, чтобы не ждать их таймаут при `-J`.
	if useProxy {
		s.preflightProxies(checkTarget(url))
	}

	ctx, cancel := context.WithTimeout(context.Background(), formatsTimeout)
	defer cancel()
	res := s.manager.Inspect(ctx, url, tc, useProxy)
	// yt-dlp не дал внятного заголовка — пробуем достать со страницы (og:title/h1/title).
	if res.OK && strings.TrimSpace(res.Title) == "" {
		res.Title = s.fetchPageTitle(url, useProxy)
	}
	return res, nil
}

// GetTasks возвращает снимки всех задач загрузки.
func (s *DownloadService) GetTasks() []download.Task {
	return s.manager.List()
}

// Create создаёт и запускает задачу загрузки. format — селектор `-f` (разрешение/аудио); пусто —
// yt-dlp выберет лучшее. useProxy — идти ли через пул прокси (значение конкретной загрузки).
// meta — отображаемые метаданные для истории (заголовок, качество). Прогресс и смена состояния
// стримятся событиями task:progress/task:state.
func (s *DownloadService) Create(url, format string, useProxy bool, meta DownloadMeta) (download.Task, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return download.Task{}, errors.New("не задана ссылка")
	}
	tc, err := s.toolchain()
	if err != nil {
		return download.Task{}, err
	}

	// Быстрая проверка прокси перед загрузкой: мёртвые узлы сразу в cooldown.
	if useProxy {
		s.preflightProxies(checkTarget(url))
	}

	// Заголовок для отображения: из метаданных, а если его нет — со страницы (best-effort).
	title := strings.TrimSpace(meta.Title)
	if title == "" {
		title = s.fetchPageTitle(url, useProxy)
	}

	dir := s.configStore.Get().DownloadsDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return download.Task{}, fmt.Errorf("создание папки загрузки: %w", err)
	}

	th := newThrottle(progressThrottle)
	return s.manager.Start(download.Request{URL: url, Title: title, OutputDir: dir, Format: format, UseProxy: useProxy, Toolchain: tc},
		func(p download.Progress) {
			if !th.allow(p.TaskID) {
				return
			}
			emit(events.TaskProgressName, events.TaskProgress{
				TaskID:     p.TaskID,
				Downloaded: p.Downloaded,
				Total:      p.Total,
				Speed:      p.Speed,
				ETA:        p.ETA,
			})
		},
		func(t download.Task) {
			emit(events.TaskStateName, events.TaskState{TaskID: t.ID, State: t.State, Error: t.Error, Status: t.Status})
			// В историю попадают только завершённые задачи (успех/ошибка/отмена).
			if download.IsFinalState(t.State) {
				s.record(t, meta)
			}
		},
		func(taskID string, l download.LogLine) {
			emit(events.TaskLogName, events.TaskLog{TaskID: taskID, Time: l.Time, Level: l.Level, Message: l.Message})
		})
}

// GetTaskJournal возвращает журнал событий задачи (для просмотра в любой момент).
func (s *DownloadService) GetTaskJournal(taskID string) []download.LogLine {
	return s.manager.Journal(taskID)
}

// Cancel отменяет задачу загрузки.
func (s *DownloadService) Cancel(id string) error {
	return s.manager.Cancel(id)
}

// Pause ставит задачу на паузу (недокачанный файл сохраняется для докачки).
func (s *DownloadService) Pause(id string) error {
	return s.manager.Pause(id)
}

// Resume возвращает задачу с паузы (скачивание продолжится).
func (s *DownloadService) Resume(id string) error {
	return s.manager.Resume(id)
}

// record добавляет запись о завершённой задаче в историю. Ошибка записи не должна срывать
// загрузку — история вспомогательна, поэтому здесь она игнорируется.
func (s *DownloadService) record(t download.Task, meta DownloadMeta) {
	if s.history == nil {
		return
	}
	size := t.Total
	if size == 0 {
		size = t.Downloaded
	}
	// Реальный размер файла точнее прогресса (после склейки/post-processing).
	if t.FilePath != "" {
		if fi, err := os.Stat(t.FilePath); err == nil {
			size = fi.Size()
		}
	}
	_ = s.history.Add(history.Entry{
		ID:         t.ID,
		URL:        t.URL,
		Title:      meta.Title,
		Duration:   meta.Duration,
		Quality:    meta.Quality,
		Details:    meta.Details,
		OutputDir:  t.OutputDir,
		Format:     t.Format,
		State:      t.State,
		Error:      t.Error,
		FilePath:   t.FilePath,
		LogPath:    t.LogPath,
		Proxy:      t.Proxy,
		Size:       size,
		CreatedAt:  t.CreatedAt,
		FinishedAt: time.Now().UnixMilli(),
	})
}

// toolchain собирает пути к внешним инструментам (быстро, без запуска процессов) и параметры
// авторизации cookies из настроек. yt-dlp обязателен; необязательные deno/ffmpeg добавляются,
// только если они доступны.
func (s *DownloadService) toolchain() (download.Toolchain, error) {
	ytdlpPath, ok := s.depsManager.Resolve(deps.NameYtDlp)
	if !ok {
		return download.Toolchain{}, errors.New("yt-dlp не установлен")
	}

	cfg := s.configStore.Get()
	tc := download.Toolchain{
		YtDlpPath:      ytdlpPath,
		CookiesBrowser: cfg.CookiesBrowser,
		CookiesFile:    cfg.CookiesFile,
	}
	if path, ok := s.depsManager.Resolve(deps.NameDeno); ok {
		tc.DenoPath = path
	}
	if path, ok := s.depsManager.Resolve(deps.NameFfmpeg); ok {
		tc.FFmpegDir = filepath.Dir(path)
	}
	return tc, nil
}

// preflightProxies быстро проверяет включённые прокси (параллельно) и отсеивает нерабочие
// (Reject → cooldown), чтобы не тратить время на мёртвые узлы. target — адрес для проверки
// достижимости (обычно хост ссылки); пусто — стандартная цель проверки. Возвращает число живых из
// проверенных. Результаты по каждому прокси стримятся событием proxy:health.
func (s *DownloadService) preflightProxies(target string) (alive, checked int) {
	if s.pool == nil || s.checkAll == nil {
		return 0, 0
	}

	enabled := make([]proxy.Proxy, 0)
	for _, p := range s.pool.Proxies() {
		if p.Enabled {
			enabled = append(enabled, p)
		}
	}
	if len(enabled) == 0 {
		return 0, 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), preflightTimeout)
	defer cancel()

	for _, r := range s.checkAll(ctx, enabled, target, preflightAttemptTimeout) {
		if r.Err != nil {
			s.pool.Reject(r.ID)
			emit(events.ProxyHealthName, events.ProxyHealth{ProxyID: r.ID, OK: false})
			continue
		}
		s.pool.ReportSuccess(r.ID, r.LatencyMs)
		emit(events.ProxyHealthName, events.ProxyHealth{ProxyID: r.ID, OK: true, LatencyMs: r.LatencyMs})
		alive++
	}
	return alive, len(enabled)
}

// checkTarget строит адрес для проверки достижимости из ссылки: хост:443. Пустая строка — если
// хост не удалось выделить (тогда используется стандартная цель проверки прокси).
func checkTarget(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	return net.JoinHostPort(u.Hostname(), "443")
}

// fetchPageTitle — запасной вариант заголовка (og:title/twitter:title/h1/<title>). Best-effort:
// при любой ошибке возвращает пустую строку.
func (s *DownloadService) fetchPageTitle(rawURL string, useProxy bool) string {
	ctx, cancel := context.WithTimeout(context.Background(), pageTitleTimeout)
	defer cancel()
	title, err := pagetitle.Fetch(ctx, s.pageTitleClient(useProxy), rawURL)
	if err != nil {
		return ""
	}
	return title
}

// pageTitleClient собирает HTTP-клиент для запасного запроса заголовка: через доступный SOCKS5-прокси
// пула (если он используется), иначе напрямую.
func (s *DownloadService) pageTitleClient(useProxy bool) *http.Client {
	transport := &http.Transport{}
	if useProxy && s.pool != nil {
		for _, p := range s.pool.Proxies() {
			if p.Enabled && (p.Scheme == proxy.SchemeSOCKS5 || p.Scheme == proxy.SchemeSOCKS5H) {
				ref := p
				transport.DialContext = ref.DialContext
				break
			}
		}
	}
	return &http.Client{Transport: transport, Timeout: pageTitleTimeout}
}
