package download

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"loadika-desktop/internal/proxy"
	"loadika-desktop/internal/ytdlp"
)

// testRequest — минимальный запрос на загрузку для тестов.
func testRequest(url, dir string) Request {
	return Request{URL: url, OutputDir: dir, UseProxy: true, Toolchain: Toolchain{YtDlpPath: "/bin/yt-dlp"}}
}

// waitState ждёт, пока onState сообщит нужное состояние (или падает по таймауту).
func waitState(t *testing.T, ch <-chan Task, state string) Task {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(2 * time.Second):
		t.Fatalf("задача не перешла в состояние %q вовремя", state)
		return Task{}
	}
}

func TestManagerQueueSequential(t *testing.T) {
	m := NewManager()
	m.SetConcurrency(1)

	var mu sync.Mutex
	running, maxRunning := 0, 0
	started := make(chan string, 4)
	release1 := make(chan struct{})

	m.run = func(ctx context.Context, _ string, args []string, _ ytdlp.Handlers) error {
		mu.Lock()
		running++
		if running > maxRunning {
			maxRunning = running
		}
		mu.Unlock()
		started <- args[len(args)-1]
		defer func() { mu.Lock(); running--; mu.Unlock() }()
		select {
		case <-release1:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	done := make(chan Task, 2)
	onState := func(ts Task) {
		if ts.State == StateCompleted {
			done <- ts
		}
	}
	if _, err := m.Start(testRequest("https://example.com/one", "/tmp/dl"), nil, onState); err != nil {
		t.Fatalf("Start one: %v", err)
	}
	if _, err := m.Start(testRequest("https://example.com/two", "/tmp/dl"), nil, onState); err != nil {
		t.Fatalf("Start two: %v", err)
	}

	if first := <-started; first != "https://example.com/one" {
		t.Fatalf("первой запустилась %q", first)
	}
	// Вторая задача не должна стартовать параллельно.
	select {
	case s := <-started:
		t.Fatalf("вторая задача стартовала параллельно: %q", s)
	case <-time.After(50 * time.Millisecond):
	}

	close(release1)
	if second := <-started; second != "https://example.com/two" {
		t.Fatalf("второй запустилась %q", second)
	}

	waitState(t, done, StateCompleted)
	waitState(t, done, StateCompleted)

	mu.Lock()
	mr := maxRunning
	mu.Unlock()
	if mr != 1 {
		t.Errorf("максимум одновременных загрузок = %d, want 1", mr)
	}
}

func TestManagerPauseAndResume(t *testing.T) {
	m := NewManager()
	m.SetConcurrency(1)

	started := make(chan struct{}, 4)
	m.run = func(ctx context.Context, _ string, _ []string, _ ytdlp.Handlers) error {
		started <- struct{}{}
		<-ctx.Done()
		return fmt.Errorf("yt-dlp: прерван: %w", ctx.Err())
	}

	done := make(chan Task, 4)
	onState := func(ts Task) {
		switch ts.State {
		case StatePaused, StateCompleted, StateCanceled:
			done <- ts
		}
	}
	task, err := m.Start(testRequest("https://example.com/v", "/tmp/dl"), nil, onState)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	<-started // задача запустилась

	if err := m.Pause(task.ID); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	if st := waitState(t, done, StatePaused); st.State != StatePaused {
		t.Fatalf("после паузы состояние = %q", st.State)
	}

	if err := m.Resume(task.ID); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("задача не возобновилась после Resume")
	}

	if err := m.Cancel(task.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if st := waitState(t, done, StateCanceled); st.State != StateCanceled {
		t.Fatalf("после отмены состояние = %q", st.State)
	}
}

func TestManagerStartCompletes(t *testing.T) {
	m := NewManager()
	m.run = func(_ context.Context, _ string, _ []string, h ytdlp.Handlers) error {
		h.Progress(ytdlp.Progress{Downloaded: 10, Total: 100, Speed: 5})
		h.Progress(ytdlp.Progress{Downloaded: 100, Total: 100})
		h.File("/tmp/dl/video.mp4")
		return nil
	}

	done := make(chan Task, 1)
	var progresses []Progress
	task, err := m.Start(testRequest("https://example.com/v", "/tmp/dl"),
		func(p Progress) { progresses = append(progresses, p) },
		func(ts Task) {
			if ts.State == StateCompleted {
				done <- ts
			}
		})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if task.ID == "" {
		t.Fatalf("начальная задача без ID = %+v", task)
	}
	// Слот свободен — schedule сразу переводит задачу в running, поэтому снимок Start не должен
	// остаться queued (это и есть фикс гонки с событием task:state). К моменту Get фейковый runner
	// мог успеть завершиться, так что допускаем и completed.
	if task.State == StateQueued {
		t.Errorf("снимок Start остался queued при свободном слоте: %+v", task)
	}

	final := waitState(t, done, StateCompleted)
	if final.State != StateCompleted {
		t.Errorf("итоговое состояние = %q", final.State)
	}
	if final.Downloaded != 100 || final.Total != 100 {
		t.Errorf("итоговый прогресс = %+v", final)
	}
	if final.FilePath != "/tmp/dl/video.mp4" {
		t.Errorf("FilePath = %q, want /tmp/dl/video.mp4", final.FilePath)
	}

	// progress приходит из той же горутины, что и финальное состояние, поэтому после
	// получения из канала чтение среза безопасно (установлен happens-before).
	if len(progresses) != 2 {
		t.Fatalf("прогрессов %d, want 2", len(progresses))
	}
	if progresses[0].TaskID != task.ID || progresses[1].Downloaded != 100 {
		t.Errorf("прогрессы = %+v", progresses)
	}

	if got, ok := m.Get(task.ID); !ok || got.State != StateCompleted {
		t.Errorf("Get = %+v, ok=%v", got, ok)
	}
	if list := m.List(); len(list) != 1 {
		t.Errorf("List вернул %d задач, want 1", len(list))
	}
}

// TestManagerStartReturnsLiveSnapshot проверяет, что Start отдаёт актуальный снимок: при свободном
// слоте задача уже running. Иначе UI «застывал» бы на queued, если событие task:state придёт раньше
// ответа Create и будет потеряно.
func TestManagerStartReturnsLiveSnapshot(t *testing.T) {
	m := NewManager()
	m.SetConcurrency(1)
	release := make(chan struct{})
	m.run = func(ctx context.Context, _ string, _ []string, _ ytdlp.Handlers) error {
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	task, err := m.Start(testRequest("https://example.com/v", "/tmp/dl"), nil, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if task.State != StateRunning {
		t.Fatalf("снимок Start = %q, want %q (живой снимок, а не queued)", task.State, StateRunning)
	}
	close(release)
}

func TestManagerStartPassesFormat(t *testing.T) {
	m := NewManager()
	var gotArgs []string
	m.run = func(_ context.Context, _ string, args []string, _ ytdlp.Handlers) error {
		gotArgs = args
		return nil
	}

	req := testRequest("https://example.com/v", "/tmp/dl")
	req.Format = "137+140"
	task, err := m.Start(req, nil, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if task.Format != "137+140" {
		t.Errorf("Task.Format = %q", task.Format)
	}

	// Ждём завершения, чтобы аргументы точно были собраны.
	deadline := time.After(2 * time.Second)
	for {
		if got, ok := m.Get(task.ID); ok && got.State == StateCompleted {
			break
		}
		select {
		case <-deadline:
			t.Fatal("задача не завершилась вовремя")
		case <-time.After(5 * time.Millisecond):
		}
	}
	if got := argValue(gotArgs, "-f"); got != "137+140" {
		t.Fatalf("селектор -f = %q, want 137+140: %v", got, gotArgs)
	}
}

func TestManagerCancel(t *testing.T) {
	m := NewManager()
	// Процесс «виснет» до отмены контекста — имитируем долгую загрузку.
	m.run = func(ctx context.Context, _ string, _ []string, _ ytdlp.Handlers) error {
		<-ctx.Done()
		return ctx.Err()
	}

	done := make(chan Task, 1)
	task, err := m.Start(testRequest("https://example.com/v", "/tmp/dl"), nil, func(ts Task) {
		if ts.State == StateCanceled {
			done <- ts
		}
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := m.Cancel(task.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if final := waitState(t, done, StateCanceled); final.State != StateCanceled {
		t.Errorf("состояние = %q, want canceled", final.State)
	}

	if err := m.Cancel(task.ID); err == nil {
		t.Error("повторная отмена завершённой задачи должна вернуть ошибку")
	}
	if err := m.Cancel("нет-такой"); err == nil {
		t.Error("отмена несуществующей задачи должна вернуть ошибку")
	}
}

func TestManagerStartFails(t *testing.T) {
	m := NewManager()
	m.run = func(_ context.Context, _ string, _ []string, _ ytdlp.Handlers) error {
		return errors.New("сеть недоступна")
	}

	done := make(chan Task, 1)
	if _, err := m.Start(testRequest("https://example.com/v", "/tmp/dl"), nil, func(ts Task) {
		if ts.State == StateFailed {
			done <- ts
		}
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if final := waitState(t, done, StateFailed); final.Error != "сеть недоступна" {
		t.Errorf("ошибка = %q, want %q", final.Error, "сеть недоступна")
	}
}

func TestManagerShutdown(t *testing.T) {
	m := NewManager()
	m.run = func(ctx context.Context, _ string, _ []string, _ ytdlp.Handlers) error {
		<-ctx.Done()
		return ctx.Err()
	}

	done := make(chan Task, 1)
	if _, err := m.Start(testRequest("https://example.com/v", "/tmp/dl"), nil, func(ts Task) {
		if ts.State == StateCanceled {
			done <- ts
		}
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	m.Shutdown()
	waitState(t, done, StateCanceled)
}

func TestManagerStartValidation(t *testing.T) {
	m := NewManager()
	m.run = func(_ context.Context, _ string, _ []string, _ ytdlp.Handlers) error { return nil }

	emptyURL := testRequest("", "/tmp/dl")
	if _, err := m.Start(emptyURL, nil, nil); err == nil {
		t.Error("ожидалась ошибка при пустом URL")
	}

	emptyDir := testRequest("https://example.com/v", "")
	if _, err := m.Start(emptyDir, nil, nil); err == nil {
		t.Error("ожидалась ошибка при пустом каталоге загрузки")
	}

	noTool := Request{URL: "https://example.com/v", OutputDir: "/tmp/dl"}
	if _, err := m.Start(noTool, nil, nil); err == nil {
		t.Error("ожидалась ошибка при отсутствии yt-dlp")
	}

	if got := m.List(); len(got) != 0 {
		t.Errorf("создано %d задач, want 0", len(got))
	}
}

func TestManagerInspect(t *testing.T) {
	m := NewManager()
	var gotArgs []string
	m.query = func(_ context.Context, _ string, args []string) ([]byte, string, error) {
		gotArgs = args
		return []byte(`{"title":"T","formats":[{"format_id":"18","ext":"mp4","vcodec":"avc1","acodec":"mp4a","height":360,"tbr":500}]}`), "WARNING: какое-то предупреждение", nil
	}

	res := m.Inspect(context.Background(), "https://example.com/v", Toolchain{YtDlpPath: "/bin/yt-dlp"}, false)
	if !res.OK {
		t.Fatalf("OK=false, error=%q", res.Error)
	}
	if res.Title != "T" || len(res.Formats) != 1 {
		t.Fatalf("res = %+v", res.Formats)
	}
	if got := res.Formats[0]; got.ID != "18" || got.Kind != ytdlp.KindMuxed || got.Quality != "360p" {
		t.Errorf("формат = %+v", got)
	}
	if !hasArg(gotArgs, "--dump-single-json") {
		t.Errorf("нет --dump-single-json в аргументах: %v", gotArgs)
	}
	if !strings.Contains(res.Output, "предупреждение") {
		t.Errorf("в выводе нет stderr: %q", res.Output)
	}
	if res.Command == "" {
		t.Error("команда должна быть заполнена")
	}
}

func TestManagerInspectFailure(t *testing.T) {
	m := NewManager()
	m.query = func(_ context.Context, _ string, _ []string) ([]byte, string, error) {
		return nil, "ERROR: unable to connect", errors.New("yt-dlp: exit status 1: ERROR: unable to connect")
	}

	res := m.Inspect(context.Background(), "https://example.com/v", Toolchain{YtDlpPath: "/bin/yt-dlp"}, false)
	if res.OK {
		t.Error("OK должен быть false при ошибке")
	}
	if res.Error == "" {
		t.Error("Error должен быть заполнен")
	}
	if !strings.Contains(res.Output, "unable to connect") {
		t.Errorf("в выводе нет stderr: %q", res.Output)
	}
}

func TestManagerInspectAttemptTimeoutIsPerProxy(t *testing.T) {
	m := NewManager()
	m.SetProxyPool(newProxyPool(t, "socks5://1.1.1.1:1\nsocks5://2.2.2.2:2"))
	m.SetMaxAttempts(2)
	// Короткий предел попытки, чтобы тест не ждал реальные 30 с.
	m.attemptTimeout = 30 * time.Millisecond

	calls := 0
	m.query = func(ctx context.Context, _ string, _ []string) ([]byte, string, error) {
		calls++
		if _, ok := ctx.Deadline(); !ok {
			t.Error("ctx попытки должен иметь собственный deadline")
		}
		<-ctx.Done() // имитируем «зависший» прокси
		return nil, "ERROR: Interrupted by user", fmt.Errorf("yt-dlp: превышено время ожидания: %w", ctx.Err())
	}

	res := m.Inspect(context.Background(), "https://example.com/v", Toolchain{YtDlpPath: "/bin/yt-dlp"}, true)
	if res.OK {
		t.Error("OK должен быть false, когда ни один прокси не ответил")
	}
	// Каждому прокси — своя попытка: их две, значит и запросов два.
	if calls != 2 {
		t.Errorf("запросов %d, want 2 (по одному на прокси)", calls)
	}
	if !strings.Contains(res.Output, "таймаут попытки") {
		t.Errorf("в выводе нет пометки о таймауте попытки: %q", res.Output)
	}
}

func TestManagerWritesLog(t *testing.T) {
	m := NewManager()
	m.SetLogDir(t.TempDir())
	m.run = func(_ context.Context, _ string, _ []string, h ytdlp.Handlers) error {
		if h.Log != nil {
			fmt.Fprintln(h.Log, "[download] диагностическая строка")
		}
		return nil
	}

	done := make(chan Task, 1)
	if _, err := m.Start(testRequest("https://example.com/v", "/tmp/dl"), nil, func(ts Task) {
		if ts.State == StateCompleted {
			done <- ts
		}
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	final := waitState(t, done, StateCompleted)
	if final.LogPath == "" {
		t.Fatal("LogPath пуст")
	}
	data, err := os.ReadFile(final.LogPath)
	if err != nil {
		t.Fatalf("чтение лога: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "yt-dlp") || !strings.Contains(content, "диагностическая строка") {
		t.Errorf("содержимое лога = %q", content)
	}
}

// newProxyPool создаёт пул с прокси из строк для тестов ротации.
func newProxyPool(t *testing.T, lines string) *proxy.Pool {
	t.Helper()
	store, err := proxy.Load(filepath.Join(t.TempDir(), "proxies.json"))
	if err != nil {
		t.Fatalf("proxy.Load: %v", err)
	}
	if _, err := store.Add(proxy.ParseList(lines).Proxies); err != nil {
		t.Fatalf("proxy.Add: %v", err)
	}
	return proxy.NewPool(store)
}

func TestManagerRetriesWithAnotherProxy(t *testing.T) {
	m := NewManager()
	m.SetProxyPool(newProxyPool(t, "socks5://1.1.1.1:1\nsocks5://2.2.2.2:2"))

	var used []string
	calls := 0
	m.run = func(_ context.Context, _ string, args []string, _ ytdlp.Handlers) error {
		calls++
		used = append(used, argValue(args, "--proxy"))
		if calls == 1 {
			return errors.New("ERROR: Unable to connect to server")
		}
		return nil
	}

	done := make(chan Task, 1)
	var states []Task
	if _, err := m.Start(testRequest("https://example.com/v", "/tmp/dl"), nil, func(ts Task) {
		states = append(states, ts)
		if ts.State == StateCompleted {
			done <- ts
		}
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	final := waitState(t, done, StateCompleted)
	if calls != 2 {
		t.Fatalf("попыток %d, want 2", calls)
	}
	if used[0] == "" || used[1] == "" || used[0] == used[1] {
		t.Errorf("прокси не сменился: %v", used)
	}
	if final.Proxy == "" {
		t.Error("Task.Proxy должен быть заполнен")
	}
	if final.Status != "" {
		t.Errorf("у завершённой задачи статус должен быть пуст: %q", final.Status)
	}

	// Во время загрузки должны приходить информативные статусы (смена прокси).
	sawRetry := false
	for _, ts := range states {
		if ts.State == StateRunning && strings.Contains(ts.Status, "Сбой") {
			sawRetry = true
		}
	}
	if !sawRetry {
		t.Errorf("не было статуса о сбое: %+v", states)
	}
}

func TestManagerSwitchesOnSlowProxy(t *testing.T) {
	m := NewManager()
	m.SetProxyPool(newProxyPool(t, "socks5://1.1.1.1:1080\nsocks5://2.2.2.2:1080"))
	m.SetMaxAttempts(3)
	// Заведомо «медленный» порог и короткие интервалы — чтобы тест не ждал реальные 20 с.
	m.minSpeedBps = 1 << 30
	m.slowDuration = 60 * time.Millisecond
	m.speedInterval = 15 * time.Millisecond

	calls := 0
	m.run = func(ctx context.Context, _ string, _ []string, h ytdlp.Handlers) error {
		calls++
		if calls > 1 {
			return nil // второй прокси «быстрый»: сразу успех
		}
		// Первый прокси: постоянно сообщаем низкую скорость и ждём отмены сторожем.
		ticker := time.NewTicker(m.speedInterval / 2)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return fmt.Errorf("yt-dlp: прерван: %w", ctx.Err())
			case <-ticker.C:
				if h.Progress != nil {
					h.Progress(ytdlp.Progress{Status: "downloading", Downloaded: 1 << 20, Total: 1 << 30, Speed: 1000})
				}
			}
		}
	}

	done := make(chan Task, 1)
	var states []Task
	if _, err := m.Start(testRequest("https://example.com/v", "/tmp/dl"), nil, func(ts Task) {
		states = append(states, ts)
		if ts.State == StateCompleted {
			done <- ts
		}
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	waitState(t, done, StateCompleted)
	if calls != 2 {
		t.Fatalf("попыток %d, want 2 (смена медленного прокси)", calls)
	}
	sawSlow := false
	for _, ts := range states {
		if strings.Contains(ts.Status, "медленн") {
			sawSlow = true
		}
	}
	if !sawSlow {
		t.Errorf("не было статуса о медленном прокси: %+v", states)
	}
}

func TestManagerSlowSingleProxyDoesNotChurn(t *testing.T) {
	m := NewManager()
	// Единственный прокси — переключаться некуда, поэтому смена по скорости не должна происходить.
	m.SetProxyPool(newProxyPool(t, "socks5://1.1.1.1:1080"))
	m.SetMaxAttempts(3)
	m.minSpeedBps = 1 << 30
	m.slowDuration = 40 * time.Millisecond
	m.speedInterval = 10 * time.Millisecond

	calls := 0
	m.run = func(ctx context.Context, _ string, _ []string, h ytdlp.Handlers) error {
		calls++
		if h.Progress != nil {
			h.Progress(ytdlp.Progress{Status: "downloading", Downloaded: 1 << 20, Total: 1 << 30, Speed: 1000})
		}
		time.Sleep(80 * time.Millisecond)
		if ctx.Err() != nil {
			t.Errorf("ctx не должен быть отменён: нет альтернативного прокси")
		}
		return nil
	}

	done := make(chan Task, 1)
	if _, err := m.Start(testRequest("https://example.com/v", "/tmp/dl"), nil, func(ts Task) {
		if ts.State == StateCompleted {
			done <- ts
		}
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	waitState(t, done, StateCompleted)
	if calls != 1 {
		t.Fatalf("попыток %d, want 1 (альтернативы нет — не переключаемся)", calls)
	}
}

func TestManagerAdaptiveSwitchingDisabled(t *testing.T) {
	m := NewManager()
	m.SetProxyPool(newProxyPool(t, "socks5://1.1.1.1:1080\nsocks5://2.2.2.2:1080"))
	m.SetMaxAttempts(3)
	m.SetAdaptiveSwitching(false)
	m.minSpeedBps = 1 << 30
	m.slowDuration = 40 * time.Millisecond
	m.speedInterval = 10 * time.Millisecond

	calls := 0
	m.run = func(ctx context.Context, _ string, _ []string, h ytdlp.Handlers) error {
		calls++
		if h.Progress != nil {
			h.Progress(ytdlp.Progress{Status: "downloading", Downloaded: 1 << 20, Total: 1 << 30, Speed: 1000})
		}
		time.Sleep(80 * time.Millisecond)
		if ctx.Err() != nil {
			t.Errorf("ctx не должен быть отменён при выключенном авто-переключении")
		}
		return nil
	}

	done := make(chan Task, 1)
	if _, err := m.Start(testRequest("https://example.com/v", "/tmp/dl"), nil, func(ts Task) {
		if ts.State == StateCompleted {
			done <- ts
		}
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	waitState(t, done, StateCompleted)
	if calls != 1 {
		t.Fatalf("попыток %d, want 1 (авто-переключение выключено)", calls)
	}
}

func TestManagerJournalRecordsEvents(t *testing.T) {
	m := NewManager()
	m.SetProxyPool(newProxyPool(t, "socks5://1.1.1.1:1\nsocks5://2.2.2.2:2"))
	calls := 0
	m.run = func(_ context.Context, _ string, _ []string, _ ytdlp.Handlers) error {
		calls++
		if calls == 1 {
			return errors.New("ERROR: Unable to connect to server")
		}
		return nil
	}

	done := make(chan Task, 1)
	task, err := m.Start(testRequest("https://example.com/v", "/tmp/dl"), nil, func(ts Task) {
		if ts.State == StateCompleted {
			done <- ts
		}
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitState(t, done, StateCompleted)

	var b strings.Builder
	for _, l := range m.Journal(task.ID) {
		b.WriteString(l.Message)
		b.WriteByte('\n')
	}
	journal := b.String()
	for _, want := range []string{"Попытка 1/", "использовано прокси", "пробуем другой прокси", "Загрузка завершена"} {
		if !strings.Contains(journal, want) {
			t.Errorf("журнал не содержит %q:\n%s", want, journal)
		}
	}
}

func TestManagerNoRotateOnLogicalError(t *testing.T) {
	m := NewManager()
	m.SetProxyPool(newProxyPool(t, "socks5://1.1.1.1:1\nsocks5://2.2.2.2:2"))

	calls := 0
	m.run = func(_ context.Context, _ string, _ []string, _ ytdlp.Handlers) error {
		calls++
		return errors.New("ERROR: Video unavailable")
	}

	done := make(chan Task, 1)
	if _, err := m.Start(testRequest("https://example.com/v", "/tmp/dl"), nil, func(ts Task) {
		if ts.State == StateFailed {
			done <- ts
		}
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	waitState(t, done, StateFailed)
	if calls != 1 {
		t.Errorf("попыток %d, want 1 (логическая ошибка — прокси не меняем)", calls)
	}
}

func hasArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

// argValue возвращает значение, идущее сразу за флагом ("" если флага нет).
func argValue(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
