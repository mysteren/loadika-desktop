// Package download управляет задачами загрузки: создание, прогресс, отмена. Доменный пакет:
// не зависит от Wails, о ходе работы сообщает через колбэки (события формирует сервис).
package download

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"loadika-desktop/internal/proxy"
	"loadika-desktop/internal/ytdlp"
)

// Состояния задачи загрузки.
const (
	StateQueued    = "queued"
	StateRunning   = "running"
	StatePaused    = "paused"
	StateCompleted = "completed"
	StateFailed    = "failed"
	StateCanceled  = "canceled"
)

// IsFinalState сообщает, что задача завершена (успех/ошибка/отмена) — финальное состояние.
func IsFinalState(state string) bool {
	return state == StateCompleted || state == StateFailed || state == StateCanceled
}

// DefaultConcurrency — сколько задач качать одновременно (последовательно = 1).
const DefaultConcurrency = 1

// DefaultMaxAttempts — сколько раз пробовать загрузку при сетевых сбоях (со сменой прокси).
const DefaultMaxAttempts = 3

// inspectAttemptTimeout — предел ожидания метаданных (`-J`) для одной попытки (одного прокси).
// Ограничивает «зависший» прокси (--socket-timeout не срабатывает на этапе SOCKS-рукопожатия),
// чтобы он не израсходовал весь общий бюджет и остальным прокси досталась честная попытка.
// Метаданные на исправном прокси приходят за секунды, поэтому держим предел небольшим — так
// быстрее переключаемся между прокси. На само скачивание этот предел НЕ действует (см. attempt).
const inspectAttemptTimeout = 15 * time.Second

// Параметры адаптивного переключения прокси по скорости загрузки.
const (
	// defaultMinSpeedBps — порог «медленно», байт/с.
	defaultMinSpeedBps = 100 * 1024
	// defaultSlowDuration — сколько низкая скорость должна держаться, чтобы сменить прокси.
	defaultSlowDuration = 20 * time.Second
	// defaultSpeedInterval — период проверки скорости.
	defaultSpeedInterval = 2 * time.Second
)

// errTooSlow — причина отмены попытки сторожем скорости (прокси рабочий, но слишком медленный).
var errTooSlow = errors.New("прокси слишком медленный")

// Toolchain — разрешённые пути к внешним инструментам и параметры авторизации для запуска yt-dlp.
// Резолвинг зависимостей и чтение настроек выполняет вызывающий слой (сервис), чтобы домен не
// знал о deps/config.
type Toolchain struct {
	YtDlpPath      string // путь к бинарнику yt-dlp
	DenoPath       string // путь к deno (--js-runtimes); пусто — не передаём
	FFmpegDir      string // каталог ffmpeg (--ffmpeg-location); пусто — не передаём
	CookiesBrowser string // браузер для --cookies-from-browser; пусто — не передаём
	CookiesFile    string // файл cookies для --cookies; пусто — не передаём
}

// Request — параметры одной загрузки.
type Request struct {
	URL       string
	Title     string // отображаемый заголовок (для UI); пусто — показываем ссылку
	OutputDir string
	Format    string // селектор формата (-f); пусто — yt-dlp выберет лучший
	UseProxy  bool   // использовать пул прокси (если он настроен)
	Toolchain Toolchain
}

// Task — задача загрузки (снимок для фронтенда).
type Task struct {
	ID         string  `json:"id"`
	URL        string  `json:"url"`
	Title      string  `json:"title"` // отображаемый заголовок; пусто — показываем ссылку
	OutputDir  string  `json:"outputDir"`
	Format     string  `json:"format"`
	State      string  `json:"state"`
	Error      string  `json:"error"`
	FilePath   string  `json:"filePath"`  // итоговый путь файла (after_move); пусто, пока неизвестен
	LogPath    string  `json:"logPath"`   // путь к логу команды yt-dlp; пусто, если логи отключены
	Proxy      string  `json:"proxy"`     // использованный прокси (без пароля); пусто — напрямую
	Status     string  `json:"status"`    // что происходит сейчас (для информирования в UI)
	CreatedAt  int64   `json:"createdAt"` // UnixMilli
	Downloaded int64   `json:"downloaded"`
	Total      int64   `json:"total"`
	Speed      float64 `json:"speed"`
	ETA        int64   `json:"eta"`
}

// Progress — прогресс задачи (payload события task:progress).
type Progress struct {
	TaskID     string  `json:"taskId"`
	Downloaded int64   `json:"downloaded"`
	Total      int64   `json:"total"`
	Speed      float64 `json:"speed"`
	ETA        int64   `json:"eta"`
}

// Уровни строк журнала задачи.
const (
	LogInfo  = "info"
	LogWarn  = "warn"
	LogError = "error"
)

// maxJournalLines — предел числа строк журнала на задачу (защита от роста).
const maxJournalLines = 500

// LogLine — строка журнала задачи (подробное информирование в клиенте).
type LogLine struct {
	Time    int64  `json:"time"`  // UnixMilli
	Level   string `json:"level"` // info | warn | error
	Message string `json:"message"`
}

// LogFunc — приёмник строк журнала (задача + строка).
type LogFunc func(taskID string, line LogLine)

// runner запускает процесс загрузки, передавая разобранный вывод. Вынесен в поле, чтобы тесты
// подменяли реальный процесс yt-dlp.
type runner func(ctx context.Context, bin string, args []string, h ytdlp.Handlers) error

// querier запускает процесс и возвращает stdout, stderr и ошибку (для `-J`).
type querier func(ctx context.Context, bin string, args []string) (stdout []byte, stderr string, err error)

// entry — внутреннее состояние задачи (мутируется под mu).
type entry struct {
	task     Task
	cancel   context.CancelFunc
	pauseReq bool      // для запущенной задачи: запрошена пауза (а не отмена)
	logs     []LogLine // журнал событий задачи

	// pid запущенного процесса yt-dlp текущей попытки; 0 — процесса нет. procStarted — момент
	// запуска процесса (для проверки возраста при определении живости). Нужны, чтобы на Windows
	// отличить ложную «отмену контекста» (сигнал консоли помечает ctx отменённым, но процесс
	// продолжает качать) от реальной остановки: перед сменой статуса сверяем живость по PID.
	pid         int
	procStarted time.Time

	// Задание для планировщика (запускается, когда освободится слот).
	bin        string
	opts       ytdlp.Options
	useProxy   bool
	onProgress func(Progress)
	onState    func(Task)
	onLog      LogFunc
}

// Manager управляет жизненным циклом задач загрузки. Безопасен для конкурентного доступа.
type Manager struct {
	mu          sync.Mutex
	order       []string // порядок создания задач
	tasks       map[string]*entry
	seq         int
	run         runner
	query       querier
	logDir      string      // каталог логов задач; пусто — логи не ведутся
	pool        *proxy.Pool // пул прокси; nil — ходим напрямую
	maxAttempts int         // число попыток при сетевых сбоях
	concurrency int         // сколько задач качать одновременно
	running     int         // сколько задач сейчас выполняется
	closing     bool        // идёт завершение приложения — новые задачи не запускаем
	// attemptTimeout — предел ожидания метаданных для одной попытки (`-J`); тесты задают короче.
	attemptTimeout time.Duration
	// Адаптивное переключение прокси по скорости загрузки.
	adaptiveSwitch bool          // менять прокси при низкой скорости
	minSpeedBps    float64       // порог «медленно», байт/с
	slowDuration   time.Duration // сколько держится низкая скорость до смены
	speedInterval  time.Duration // период проверки скорости
}

// NewManager создаёт менеджер задач, использующий настоящий yt-dlp.
func NewManager() *Manager {
	return &Manager{
		tasks:          make(map[string]*entry),
		run:            ytdlp.Run,
		query:          ytdlp.RunInfo,
		maxAttempts:    DefaultMaxAttempts,
		concurrency:    DefaultConcurrency,
		attemptTimeout: inspectAttemptTimeout,
		adaptiveSwitch: true,
		minSpeedBps:    defaultMinSpeedBps,
		slowDuration:   defaultSlowDuration,
		speedInterval:  defaultSpeedInterval,
	}
}

// SetLogDir задаёт каталог, в который пишутся логи команд yt-dlp (пусто — не вести).
// Вызывается один раз при инициализации, до запуска задач.
func (m *Manager) SetLogDir(dir string) {
	m.logDir = dir
}

// SetProxyPool задаёт пул прокси для выбора и ротации (nil — работать напрямую).
func (m *Manager) SetProxyPool(pool *proxy.Pool) {
	m.pool = pool
}

// SetMaxAttempts задаёт число попыток загрузки при сетевых сбоях.
func (m *Manager) SetMaxAttempts(n int) {
	if n >= 1 {
		m.maxAttempts = n
	}
}

// SetAdaptiveSwitching включает/выключает смену прокси при низкой скорости загрузки.
func (m *Manager) SetAdaptiveSwitching(enabled bool) {
	m.adaptiveSwitch = enabled
}

// SetConcurrency задаёт, сколько задач качать одновременно (>= 1). Значение 1 — строго
// последовательно. После изменения сразу пробуем запустить задачи из очереди.
func (m *Manager) SetConcurrency(n int) {
	if n < 1 {
		n = 1
	}
	m.mu.Lock()
	m.concurrency = n
	m.mu.Unlock()
	m.schedule()
}

// Start создаёт задачу и ставит её в очередь на загрузку (не более m.concurrency одновременно).
// Прогресс сообщается через onProgress, смены состояния — через onState, строки журнала — через
// onLog (все могут быть nil; onLog — необязательный, чтобы не менять существующие вызовы).
// Возвращает снимок задачи в состоянии queued. Ошибки валидации (нет ссылки/каталога/yt-dlp)
// возвращаются синхронно, без постановки в очередь.
func (m *Manager) Start(req Request, onProgress func(Progress), onState func(Task), onLog ...LogFunc) (Task, error) {
	if req.Toolchain.YtDlpPath == "" {
		return Task{}, errors.New("не найден yt-dlp")
	}
	var logFn LogFunc
	if len(onLog) > 0 {
		logFn = onLog[0]
	}
	opts := ytdlp.Options{
		URL:            req.URL,
		OutputDir:      req.OutputDir,
		Format:         req.Format,
		DenoPath:       req.Toolchain.DenoPath,
		FFmpegDir:      req.Toolchain.FFmpegDir,
		CookiesBrowser: req.Toolchain.CookiesBrowser,
		CookiesFile:    req.Toolchain.CookiesFile,
	}
	// Проверяем параметры без прокси (прокси подставляется на каждой попытке).
	if _, err := ytdlp.BuildArgs(opts); err != nil {
		return Task{}, err
	}

	task := Task{
		ID:        m.nextID(),
		URL:       req.URL,
		Title:     req.Title,
		OutputDir: req.OutputDir,
		Format:    req.Format,
		State:     StateQueued,
		CreatedAt: time.Now().UnixMilli(),
	}
	e := &entry{
		task:       task,
		bin:        req.Toolchain.YtDlpPath,
		opts:       opts,
		useProxy:   req.UseProxy,
		onProgress: onProgress,
		onState:    onState,
		onLog:      logFn,
	}

	m.mu.Lock()
	m.tasks[task.ID] = e
	m.order = append(m.order, task.ID)
	m.mu.Unlock()

	if onState != nil {
		onState(task)
	}
	m.schedule()
	return task, nil
}

// schedule запускает задачи из очереди, пока есть свободные слоты (не более concurrency).
func (m *Manager) schedule() {
	for {
		m.mu.Lock()
		if m.closing || m.running >= m.concurrency {
			m.mu.Unlock()
			return
		}
		id := m.nextQueuedLocked()
		if id == "" {
			m.mu.Unlock()
			return
		}
		e := m.tasks[id]
		e.task.State = StateRunning
		e.pauseReq = false
		ctx, cancel := context.WithCancel(context.Background())
		e.cancel = cancel
		m.running++
		task := e.task
		m.mu.Unlock()

		if e.onState != nil {
			e.onState(task)
		}
		go m.runEntry(ctx, id, e)
	}
}

// nextQueuedLocked возвращает id первой задачи в очереди (вызывается при удержанном mu).
func (m *Manager) nextQueuedLocked() string {
	for _, id := range m.order {
		if e := m.tasks[id]; e != nil && e.task.State == StateQueued {
			return id
		}
	}
	return ""
}

// Inspect запрашивает метаданные ссылки (`yt-dlp -J`), возвращая разобранные форматы и сырой
// ответ библиотеки (команда + stdout/stderr) для диагностики. useProxy — использовать ли пул
// прокси (с автопереключением при сетевых сбоях). Ошибки не роняют вызов — они кладутся в
// Inspect (OK=false).
func (m *Manager) Inspect(ctx context.Context, url string, tc Toolchain, useProxy bool) Inspect {
	result := Inspect{Formats: []ytdlp.Format{}, Qualities: []QualityOption{}}
	if tc.YtDlpPath == "" {
		result.Error = "не найден yt-dlp"
		return result
	}

	attempts := m.maxAttempts
	if attempts < 1 {
		attempts = 1
	}
	tried := make(map[string]bool)
	var outputs []string

	for n := 1; n <= attempts; n++ {
		// Общий бюджет (или отмена) исчерпан — дальнейшие попытки бессмысленны.
		if ctx.Err() != nil {
			break
		}

		proxyURL, proxyID, proxyDisplay := "", "", ""
		if m.pool != nil && useProxy {
			// «Умный» выбор для метаданных: здоровье, задержка, скорость (а не стратегия).
			if ref, ok := m.pool.PickBest(tried); ok {
				proxyURL, proxyID, proxyDisplay = ref.URL(), ref.ID, ref.Display()
			} else if n == 1 && m.pool.EnabledCount() > 0 {
				result.Error = "нет доступных прокси (все выключены или в cooldown)"
				return result
			}
		}

		args, err := ytdlp.BuildInfoArgs(ytdlp.InfoOptions{URL: url, DenoPath: tc.DenoPath, Proxy: proxyURL, CookiesBrowser: tc.CookiesBrowser, CookiesFile: tc.CookiesFile})
		if err != nil {
			result.Error = err.Error()
			return result
		}
		// Команду фиксируем по первой попытке — различия между попытками (прокси) видны в их отметках ниже.
		if result.Command == "" {
			result.Command = tc.YtDlpPath + " " + strings.Join(args, " ")
		}

		note := "напрямую"
		if proxyDisplay != "" {
			note = "через " + proxyDisplay
		}
		start := time.Now()
		// Каждой попытке — собственный предел ожидания, чтобы «зависший» прокси не съел весь бюджет.
		attemptCtx, cancelAttempt := context.WithTimeout(ctx, m.attemptTimeout)
		stdout, stderr, runErr := m.query(attemptCtx, tc.YtDlpPath, args)
		overallExpired := ctx.Err() != nil
		attemptTimedOut := !overallExpired && errors.Is(attemptCtx.Err(), context.DeadlineExceeded)
		cancelAttempt()

		attemptOut := combineOutput(stdout, stderr)
		if attemptTimedOut {
			attemptOut += fmt.Sprintf("\n(примечание: попытка не ответила за %.0f с — это наш таймаут попытки, а не действие пользователя)", m.attemptTimeout.Seconds())
		}
		if overallExpired {
			attemptOut += "\n(примечание: исчерпан общий лимит ожидания метаданных; «Interrupted by user» — наш graceful-стоп)"
		}
		outputs = append(outputs, fmt.Sprintf("# попытка %d (%s)\n%s", n, note, attemptOut))

		if runErr == nil {
			if m.pool != nil && proxyID != "" {
				m.pool.ReportSuccess(proxyID, time.Since(start).Milliseconds())
			}
			info, parseErr := ytdlp.ParseInfo(stdout)
			if parseErr != nil {
				result.Error = "не удалось разобрать ответ yt-dlp: " + parseErr.Error()
				result.Output = truncate(strings.Join(outputs, "\n\n"), 8000)
				return result
			}
			result.Title = info.Title
			result.Duration = info.Duration
			result.Formats = info.Formats
			result.Qualities = QualityOptions(info)
			result.OK = true
			result.Output = truncate(strings.Join(outputs, "\n\n"), 8000)
			return result
		}

		network := ytdlp.ClassifyError(runErr.Error()) == ytdlp.ErrorNetwork ||
			errors.Is(runErr, context.DeadlineExceeded) ||
			attemptTimedOut
		if proxyID == "" || !network {
			result.Error = runErr.Error()
			result.Output = truncate(strings.Join(outputs, "\n\n"), 8000)
			return result
		}
		m.pool.ReportFailure(proxyID)
		tried[proxyID] = true
	}

	result.Error = "не удалось получить метаданные"
	result.Output = truncate(strings.Join(outputs, "\n\n"), 8000)
	return result
}

// combineOutput объединяет stderr и stdout yt-dlp в читаемый текст (с ограничением длины).
func combineOutput(stdout []byte, stderr string) string {
	var b strings.Builder
	if strings.TrimSpace(stderr) != "" {
		b.WriteString("stderr:\n")
		b.WriteString(strings.TrimRight(stderr, "\n"))
	}
	if len(stdout) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("stdout:\n")
		b.Write(stdout)
	}
	return truncate(b.String(), 8000)
}

// truncate ограничивает строку по числу символов (без разрыва UTF-8).
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "\n… (вывод обрезан)"
}

// Pause ставит задачу на паузу. Запущенная задача мягко останавливается (недокачанный файл
// сохраняется — потом можно докачать), поставленная в очередь — просто помечается как на паузе.
func (m *Manager) Pause(id string) error {
	m.mu.Lock()
	e := m.tasks[id]
	if e == nil {
		m.mu.Unlock()
		return fmt.Errorf("задача не найдена: %q", id)
	}
	switch e.task.State {
	case StateRunning:
		e.pauseReq = true
		cancel := e.cancel
		m.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		return nil
	case StateQueued:
		e.task.State = StatePaused
		task, onState, onLog := e.task, e.onState, e.onLog
		m.mu.Unlock()
		if onState != nil {
			onState(task)
		}
		m.addLog(id, LogWarn, "Поставлено на паузу", onLog)
		return nil
	default:
		state := e.task.State
		m.mu.Unlock()
		return fmt.Errorf("нельзя поставить на паузу задачу в состоянии %q", state)
	}
}

// Resume возвращает задачу с паузы в очередь (при свободном слоте скачивание продолжится, yt-dlp
// докачает недокачанный файл).
func (m *Manager) Resume(id string) error {
	m.mu.Lock()
	e := m.tasks[id]
	if e == nil {
		m.mu.Unlock()
		return fmt.Errorf("задача не найдена: %q", id)
	}
	if e.task.State != StatePaused {
		state := e.task.State
		m.mu.Unlock()
		return fmt.Errorf("задача не на паузе: %q", state)
	}
	e.task.State = StateQueued
	task, onState := e.task, e.onState
	m.mu.Unlock()

	if onState != nil {
		onState(task)
	}
	m.schedule()
	return nil
}

// Cancel отменяет задачу: запущенную — мягко останавливает (состояние canceled фиксирует runEntry),
// поставленную в очередь или на паузе — сразу помечает как отменённую.
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	e := m.tasks[id]
	if e == nil {
		m.mu.Unlock()
		return fmt.Errorf("задача не найдена: %q", id)
	}
	switch e.task.State {
	case StateRunning:
		e.pauseReq = false
		cancel := e.cancel
		m.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		return nil
	case StateQueued, StatePaused:
		e.task.State = StateCanceled
		task, onState, onLog := e.task, e.onState, e.onLog
		m.mu.Unlock()
		if onState != nil {
			onState(task)
		}
		m.addLog(id, LogWarn, "Отменено пользователем", onLog)
		return nil
	default:
		state := e.task.State
		m.mu.Unlock()
		return fmt.Errorf("задача уже завершена: %q", state)
	}
}

// List возвращает снимки всех задач в порядке создания.
func (m *Manager) List() []Task {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]Task, 0, len(m.order))
	for _, id := range m.order {
		if e := m.tasks[id]; e != nil {
			out = append(out, e.task)
		}
	}
	return out
}

// Get возвращает снимок задачи по идентификатору.
func (m *Manager) Get(id string) (Task, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	e := m.tasks[id]
	if e == nil {
		return Task{}, false
	}
	return e.task, true
}

// Shutdown отменяет все незавершённые задачи (например, при закрытии приложения) и запрещает
// запуск новых из очереди.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	m.closing = true
	cancels := make([]context.CancelFunc, 0, len(m.tasks))
	for _, e := range m.tasks {
		if e.cancel != nil {
			cancels = append(cancels, e.cancel)
		}
	}
	m.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
}

// nextID выдаёт уникальный идентификатор задачи. Включает время создания, чтобы ID не совпадали
// между запусками приложения (иначе записи истории и логи разных запусков затирали бы друг друга).
func (m *Manager) nextID() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.seq++
	return fmt.Sprintf("task-%d-%d", time.Now().UnixMilli(), m.seq)
}

// runEntry выполняет одну задачу (с ретраями при сетевых сбоях), фиксирует итоговое состояние,
// освобождает слот и запускает следующую задачу из очереди.
func (m *Manager) runEntry(ctx context.Context, id string, e *entry) {
	h := ytdlp.Handlers{
		Progress: func(p ytdlp.Progress) {
			m.setProgress(id, p)
			if e.onProgress != nil {
				e.onProgress(Progress{
					TaskID:     id,
					Downloaded: p.Downloaded,
					Total:      p.Total,
					Speed:      p.Speed,
					ETA:        p.ETA,
				})
			}
		},
		File: func(path string) { m.setFilePath(id, path) },
	}

	// Если задан каталог логов, пишем туда полный вывод команды (для диагностики).
	var logFile *os.File
	if f, path, err := m.openLog(id, e.bin, e.opts); err == nil {
		logFile = f
		defer f.Close()
		m.setLogPath(id, path)
		h.Log = f
	}

	state, errMsg := m.attempt(ctx, id, e.bin, e.opts, e.useProxy, h, logFile, e.onState, e.onLog)

	// Если во время выполнения запрошена пауза — это не отмена, а пауза (файл остаётся, докачаем).
	m.mu.Lock()
	paused := e.pauseReq
	m.mu.Unlock()
	if paused && state == StateCanceled {
		state = StatePaused
		errMsg = ""
	}

	// Ложная «отмена» на Windows: контекст помечен отменённым (например, Ctrl+C/Ctrl+Break в
	// родительской консоли доставляется всем процессам), но процесс yt-dlp жив и продолжает качать.
	// Не даём такой задаче уйти в canceled/paused/failed с работающим процессом.
	if !paused && (state == StateCanceled || state == StateFailed) && ctx.Err() != nil && m.processAlive(id) {
		m.addLog(id, LogWarn, "Ложная остановка процесса — загрузка продолжается", e.onLog)
		m.mu.Lock()
		if ee := m.tasks[id]; ee != nil && ee.task.State == StateRunning {
			ee.task.Error = ""
		}
		m.mu.Unlock()
		return
	}

	final := m.finish(id, state, errMsg)

	// Итоговая запись журнала (детали попыток пишет attempt).
	switch state {
	case StateCompleted:
		msg := "Загрузка завершена"
		if final.FilePath != "" {
			msg += ": " + final.FilePath
		}
		m.addLog(id, LogInfo, msg, e.onLog)
	case StateCanceled:
		m.addLog(id, LogWarn, "Отменено пользователем", e.onLog)
	case StatePaused:
		m.addLog(id, LogWarn, "Поставлено на паузу", e.onLog)
	case StateFailed:
		m.addLog(id, LogError, "Не удалось: "+shortError(errors.New(errMsg)), e.onLog)
	}

	// Освобождаем слот и запускаем следующую задачу из очереди.
	m.mu.Lock()
	m.running--
	if m.running < 0 {
		m.running = 0
	}
	m.mu.Unlock()

	if e.onState != nil {
		e.onState(final)
	}
	m.schedule()
}

// attempt выполняет попытки загрузки, меняя прокси при сетевых сбоях и при слишком низкой скорости.
// useProxy — использовать ли пул прокси для этой задачи. onState вызывается при изменении статуса
// (информирование в UI). Возвращает итоговое состояние задачи и текст ошибки.
// Смена прокси по скорости безопасна: yt-dlp продолжает недокачанный файл (--continue включён по
// умолчанию), поэтому перезапуск на другом прокси не теряет прогресс.
func (m *Manager) attempt(ctx context.Context, id, bin string, opts ytdlp.Options, useProxy bool, h ytdlp.Handlers, logFile *os.File, onState func(Task), onLog LogFunc) (string, string) {
	attempts := m.maxAttempts
	if attempts < 1 {
		attempts = 1
	}
	tried := make(map[string]bool)
	slowNext := false
	proxyUses := 0
	var lastErr error

	for n := 1; n <= attempts; n++ {
		proxyURL, proxyID, proxyDisplay := "", "", ""
		if m.pool != nil && useProxy {
			var ref proxy.Proxy
			var ok bool
			if slowNext {
				ref, ok = m.pool.PickFastest(tried)
			} else {
				ref, ok = m.pool.Pick(tried)
			}
			if ok {
				proxyURL, proxyID, proxyDisplay = ref.URL(), ref.ID, ref.Display()
			} else if m.pool.EnabledCount() > 0 {
				// Пул непуст, но все прокси уже пробованы/в cooldown — «напрямую» не откатываемся.
				if n == 1 {
					msg := "нет доступных прокси (все выключены или в cooldown)"
					m.addLog(id, LogError, msg, onLog)
					return StateFailed, msg
				}
				break
			}
			// EnabledCount()==0 — прокси не настроены: идём напрямую.
		}

		m.emitStatus(id, statusStart(n, attempts, proxyDisplay), onState)

		// Журнал: показываем попытку, какое соединение и сколько прокси уже задействовано.
		where := "прямое соединение"
		if proxyDisplay != "" {
			proxyUses++
			where = fmt.Sprintf("прокси %s (использовано прокси: %d)", proxyDisplay, proxyUses)
		}
		m.addLog(id, LogInfo, fmt.Sprintf("Попытка %d/%d: %s", n, attempts, where), onLog)

		attemptOpts := opts
		attemptOpts.Proxy = proxyURL
		args, err := ytdlp.BuildArgs(attemptOpts)
		if err != nil {
			return StateFailed, err.Error()
		}
		if proxyDisplay != "" {
			m.setProxy(id, proxyDisplay)
		}
		if logFile != nil {
			note := "напрямую"
			if proxyDisplay != "" {
				note = "через " + proxyDisplay
			}
			fmt.Fprintf(logFile, "\n# попытка %d (%s)\n", n, note)
		}

		// Сторож скорости срабатывает, только если есть куда переключиться (другой доступный прокси).
		canSwitch := proxyID != "" && m.adaptiveSwitch && m.hasAlternativeProxy(tried, proxyID)

		attemptCtx, cancelAttempt := context.WithCancelCause(ctx)
		guard := &speedGuard{}
		prevProgress := h.Progress
		attemptHandlers := h
		attemptHandlers.Progress = func(p ytdlp.Progress) {
			guard.observe(p.Speed)
			if prevProgress != nil {
				prevProgress(p)
			}
		}

		stopWatch := func() {}
		if canSwitch {
			stopWatch = m.watchSpeed(attemptCtx, cancelAttempt, guard)
		}

		start := time.Now()
		// Публикуем PID процесса в задаче СРАЗУ после cmd.Start() (через h.Cmd): менеджер по нему
		// проверяет живость загрузки, чтобы отличить ложную «отмену контекста» на Windows от
		// реальной остановки. Если бы PID публиковался только после возврата m.run, проверка
		// живости в этот промежуток видела бы pid==0 и ложно считала процесс мёртвым.
		var curCmd *exec.Cmd
		attemptHandlers.Cmd = &curCmd
		witness := func() {
			if pid := procPID(curCmd); pid > 0 {
				m.setProcPID(id, pid, start)
			}
		}
		prevProg := attemptHandlers.Progress
		attemptHandlers.Progress = func(p ytdlp.Progress) {
			witness()
			if prevProg != nil {
				prevProg(p)
			}
		}
		prevFile := attemptHandlers.File
		attemptHandlers.File = func(path string) {
			witness()
			if prevFile != nil {
				prevFile(path)
			}
		}
		runErr := m.run(attemptCtx, bin, args, attemptHandlers)
		witness() // на случай процесса без единой строки вывода
		m.clearStalePID(id, curCmd)
		stopWatch()
		wasSlow := errors.Is(context.Cause(attemptCtx), errTooSlow)
		cancelAttempt(nil)

		// Телеметрия: сообщаем пиковую скорость прокси в пул (в т.ч. чтобы отличать быстрые).
		if m.pool != nil && proxyID != "" {
			m.pool.ReportSpeed(proxyID, guard.peak())
		}

		if runErr == nil {
			if m.pool != nil && proxyID != "" {
				m.pool.ReportSuccess(proxyID, time.Since(start).Milliseconds())
			}
			return StateCompleted, ""
		}

		lastErr = runErr

		if wasSlow && proxyID != "" {
			// Прокси рабочий, но медленный — меняем на более быстрый; загрузка продолжится (resume).
			tried[proxyID] = true
			slowNext = true
			m.addLog(id, LogWarn, fmt.Sprintf("Прокси %s медленный — переключаюсь на другой", proxyDisplay), onLog)
			m.emitStatus(id, "Прокси медленный — пробуем другой", onState)
			continue
		}
		if errors.Is(runErr, context.Canceled) {
			return StateCanceled, ""
		}

		// Сменить прокси можно, если он использовался и ошибка сетевая/таймаут.
		network := ytdlp.ClassifyError(runErr.Error()) == ytdlp.ErrorNetwork ||
			errors.Is(runErr, context.DeadlineExceeded)
		if proxyID == "" || !network {
			break
		}
		m.pool.ReportFailure(proxyID)
		tried[proxyID] = true
		m.addLog(id, LogWarn, fmt.Sprintf("Прокси %s: %s — пробуем другой прокси", proxyDisplay, shortError(runErr)), onLog)
		m.emitStatus(id, "Сбой: "+shortError(runErr)+" — пробуем другой прокси", onState)
	}

	if lastErr == nil {
		lastErr = errors.New("не удалось выполнить загрузку")
	}
	return StateFailed, lastErr.Error()
}

// hasAlternativeProxy сообщает, есть ли доступный прокси, отличный от текущего и уже пробованных.
func (m *Manager) hasAlternativeProxy(tried map[string]bool, currentID string) bool {
	if m.pool == nil {
		return false
	}
	exclude := make(map[string]bool, len(tried)+1)
	for id := range tried {
		exclude[id] = true
	}
	if currentID != "" {
		exclude[currentID] = true
	}
	return m.pool.HasAvailable(exclude)
}

// emitStatus сохраняет и сообщает текущий статус задачи (если onState задан).
func (m *Manager) emitStatus(id, status string, onState func(Task)) {
	task, ok := m.updateStatus(id, status)
	if ok && onState != nil {
		onState(task)
	}
}

// updateStatus сохраняет статус задачи и возвращает её снимок.
func (m *Manager) updateStatus(id, status string) (Task, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	e := m.tasks[id]
	if e == nil {
		return Task{}, false
	}
	e.task.Status = status
	return e.task, true
}

// addLog добавляет строку в журнал задачи и (если задан) передаёт её в onLog.
func (m *Manager) addLog(id, level, message string, onLog LogFunc) {
	line := LogLine{Time: time.Now().UnixMilli(), Level: level, Message: message}

	m.mu.Lock()
	if e := m.tasks[id]; e != nil {
		e.logs = append(e.logs, line)
		if len(e.logs) > maxJournalLines {
			e.logs = e.logs[len(e.logs)-maxJournalLines:]
		}
	}
	m.mu.Unlock()

	if onLog != nil {
		onLog(id, line)
	}
}

// Journal возвращает копию журнала задачи (для просмотра в любой момент).
func (m *Manager) Journal(id string) []LogLine {
	m.mu.Lock()
	defer m.mu.Unlock()

	e := m.tasks[id]
	if e == nil {
		return nil
	}
	out := make([]LogLine, len(e.logs))
	copy(out, e.logs)
	return out
}

// statusStart формирует стартовый статус попытки.
func statusStart(n, attempts int, proxyDisplay string) string {
	where := "напрямую"
	if proxyDisplay != "" {
		where = "через " + proxyDisplay
	}
	if attempts > 1 {
		return fmt.Sprintf("Попытка %d/%d (%s)", n, attempts, where)
	}
	return "Загрузка (" + where + ")"
}

// shortError делает текст ошибки пригодным для строки статуса (первая строка, с ограничением).
func shortError(err error) string {
	msg := err.Error()
	if i := strings.IndexByte(msg, '\n'); i >= 0 {
		msg = msg[:i]
	}
	const max = 140
	runes := []rune(msg)
	if len(runes) > max {
		msg = string(runes[:max]) + "…"
	}
	return msg
}

// openLog создаёт файл лога задачи и пишет в него заголовок с командой (без прокси — прокси
// фиксируется на каждой попытке). Если каталог логов не задан, возвращает ошибку.
func (m *Manager) openLog(id, bin string, opts ytdlp.Options) (*os.File, string, error) {
	if m.logDir == "" {
		return nil, "", errors.New("каталог логов не задан")
	}
	if err := os.MkdirAll(m.logDir, 0o755); err != nil {
		return nil, "", err
	}

	path := filepath.Join(m.logDir, id+".log")
	// Открываем на дозапись: при возобновлении с паузы/повторе лог задачи сохраняется.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, "", err
	}

	// Заголовок делает лог самодостаточным (что и когда запускалось).
	baseArgs, _ := ytdlp.BuildArgs(opts)
	fmt.Fprintf(f, "# %s\n# %s %s\n", time.Now().Format(time.RFC3339), bin, strings.Join(baseArgs, " "))
	return f, path, nil
}

// setProxy сохраняет представление использованного прокси (без пароля).
func (m *Manager) setProxy(id, display string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if e := m.tasks[id]; e != nil {
		e.task.Proxy = display
	}
}

// setLogPath сохраняет путь к логу задачи.
func (m *Manager) setLogPath(id, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if e := m.tasks[id]; e != nil {
		e.task.LogPath = path
	}
}

// setProgress сохраняет последний прогресс задачи — чтобы List/Get отдавали актуальный снимок.
func (m *Manager) setProgress(id string, p ytdlp.Progress) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if e := m.tasks[id]; e != nil {
		e.task.Downloaded = p.Downloaded
		e.task.Total = p.Total
		e.task.Speed = p.Speed
		e.task.ETA = p.ETA
	}
}

// setFilePath сохраняет итоговый путь файла, полученный от yt-dlp (after_move).
func (m *Manager) setFilePath(id, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if e := m.tasks[id]; e != nil {
		e.task.FilePath = path
	}
}

// setProcPID запоминает PID процесса текущей попытки загрузки (0 — процесс завершился или
// не запускался) и время его запуска. Вызывается из attempt сразу после старта процесса
// (первый вывод/после возврата m.run).
func (m *Manager) setProcPID(id string, pid int, startedAt time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if e := m.tasks[id]; e != nil {
		e.pid = pid
		e.procStarted = startedAt
	}
}

// clearStalePID снимает учёт PID, если процесс этой попытки уже завершён к моменту возврата
// m.run: иначе проверка живости может ложно принять повторное использование PID системой
// за «загрузка продолжается».
func (m *Manager) clearStalePID(id string, cmd *exec.Cmd) {
	if cmd == nil || cmd.ProcessState == nil {
		return // процесс мог быть убит без состояния — оставляем PID для проверки живости
	}
	m.setProcPID(id, 0, time.Time{})
}

// processAlive сообщает, жив ли ещё процесс yt-dlp этой задачи. Используется, чтобы на Windows
// не принимать ложную «отмену контекста» за реальную остановку: если процесс качает — задача
// остаётся в состоянии running.
func (m *Manager) processAlive(id string) bool {
	m.mu.Lock()
	pid, started := 0, time.Time{}
	if e := m.tasks[id]; e != nil {
		pid, started = e.pid, e.procStarted
	}
	m.mu.Unlock()
	return pid > 0 && procAlive(pid, started)
}

// procPID возвращает PID процесса cmd (0 — процесс не публиковался/завершён без состояния).
func procPID(cmd *exec.Cmd) int {
	if cmd == nil || cmd.Process == nil {
		return 0
	}
	return cmd.Process.Pid
}

// finish фиксирует итоговое состояние задачи и освобождает отмену.
func (m *Manager) finish(id, state, errMsg string) Task {
	m.mu.Lock()
	defer m.mu.Unlock()

	e := m.tasks[id]
	if e == nil {
		return Task{}
	}
	e.task.State = state
	e.task.Error = errMsg
	e.task.Status = ""
	e.cancel = nil
	return e.task
}
