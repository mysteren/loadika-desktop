package ytdlp

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"loadika-desktop/internal/procattr"
	"loadika-desktop/internal/procenv"
)

// graceKill — сколько ждать мягкого завершения процесса после отмены контекста,
// прежде чем убить его принудительно.
const graceKill = 5 * time.Second

// maxLineBytes — предел длины одной строки вывода (строка прогресса — небольшой JSON).
const maxLineBytes = 1 << 20 // 1 МиБ

// errTailLimit — сколько символов stderr попадает в текст ошибки, чтобы не раздувать сообщение.
const errTailLimit = 2000

// Handlers — обработчики машинно-читаемого вывода yt-dlp. Все поля необязательны.
type Handlers struct {
	Progress func(Progress)    // строки прогресса
	File     func(path string) // итоговый путь файла (after_move)
	Log      io.Writer         // полный вывод процесса (кроме строк прогресса) для диагностики
	Cmd      **exec.Cmd        // сюда публикуется запущенный процесс сразу после Start — вызывающий
	// может проверить его живость (ytdlp.Alive), чтобы отличить ложную «отмену контекста» на
	// Windows от реальной остановки загрузки.
}

// Run запускает yt-dlp с готовыми аргументами и построчно читает его stdout. Разобранный
// прогресс и итоговый путь файла передаются в h. Возвращает ошибку запуска или ненулевой код
// завершения. Отмена ctx мягко останавливает процесс, а если он не завершился за graceKill —
// убивает его (см. exec.Cmd.WaitDelay).
func Run(ctx context.Context, bin string, args []string, h Handlers) error {
	cmd := newCommand(ctx, bin, args)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("yt-dlp: получение stdout: %w", err)
	}
	var stderr bytes.Buffer
	if h.Log != nil {
		// stderr идёт и в буфер (для текста ошибки), и в лог.
		cmd.Stderr = io.MultiWriter(&stderr, h.Log)
	} else {
		cmd.Stderr = &stderr
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("yt-dlp: запуск %q: %w", bin, err)
	}
	if h.Cmd != nil {
		*h.Cmd = cmd
	}

	scanErr := scanOutput(stdout, h)
	waitErr := cmd.Wait()
	ctxErr := ctx.Err()

	// Процесс остановлен нами (пауза/отмена): контекст отменён и процесс завершён сигналом или
	// кодом STATUS_CONTROL_C_EXIT. Сообщаем context.Canceled — менеджер различает паузу/отмену сам.
	if stopErr := stopError(cmd, ctxErr, waitErr); stopErr != nil {
		return stopErr
	}
	// Контекст «отменён», но процесс завершился сам по обычному коду возврата — отмена на него
	// не повлияла (ложная отмена: Ctrl+C/Ctrl+Break доставляется всем процессам одной консоли,
	// поэтому на Windows ctx может быть помечен отменённым ещё до нашей остановки). Решаем по
	// фактическому результату процесса ниже.
	if ctxErr != nil && errors.Is(ctxErr, context.DeadlineExceeded) {
		return interruptError(stderr.String(), ctxErr)
	}
	if scanErr != nil {
		return fmt.Errorf("yt-dlp: чтение вывода: %w", scanErr)
	}
	if waitErr != nil {
		return fmt.Errorf("yt-dlp: %w%s", waitErr, stderrSuffix(stderr.String()))
	}
	return nil
}

// Output запускает yt-dlp и возвращает полный stdout (для команд вида `-J`, `--print`).
// При ненулевом коде завершения ошибка содержит хвост stderr. Привязан к ctx так же, как Run.
func Output(ctx context.Context, bin string, args []string) ([]byte, error) {
	stdout, _, err := RunInfo(ctx, bin, args)
	return stdout, err
}

// RunInfo запускает yt-dlp и возвращает stdout, stderr и ошибку — чтобы вызывающий мог показать
// точный ответ библиотеки и понять причину сбоя.
func RunInfo(ctx context.Context, bin string, args []string) ([]byte, string, error) {
	cmd := newCommand(ctx, bin, args)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if stopErr := fixStopError(ctx, cmd, err); stopErr != nil {
		return stdout.Bytes(), stderr.String(), stopErr
	}
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return stdout.Bytes(), stderr.String(), interruptError(stderr.String(), ctxErr)
		}
		return stdout.Bytes(), stderr.String(), fmt.Errorf("yt-dlp: %w%s", err, stderrSuffix(stderr.String()))
	}
	return stdout.Bytes(), stderr.String(), nil
}

// interruptError формирует понятное сообщение при отмене/таймауте и добавляет хвост stderr
// (чтобы было видно, на чём именно застряло). ctxErr оборачивается через %w для errors.Is.
func interruptError(stderr string, ctxErr error) error {
	if ctxErr == context.DeadlineExceeded {
		return fmt.Errorf("yt-dlp: превышено время ожидания: %w%s", ctxErr, stderrSuffix(stderr))
	}
	return fmt.Errorf("yt-dlp: прерван: %w%s", ctxErr, stderrSuffix(stderr))
}

// errStoppedByUs — ошибка «процесс остановлен нами» (пауза/отмена), распознаётся менеджером
// задач через errors.Is(context.Canceled). На Windows убитый TerminateProcess/Ctrl+C процесс
// yt-dlp завершается кодом STATUS_CONTROL_C_EXIT; Go может расценить такой выход как штатный
// (nil из Wait) — тогда задача получила бы ложный статус (например, completed сразу после
// паузы). Поэтому такой выход мы явно подменяем этой ошибкой.
var errStoppedByUs = fmt.Errorf("yt-dlp: остановлен (пауза/отмена): %w", context.Canceled)

// controlCExitStatus — код выхода STATUS_CONTROL_C_EXIT (0xC000013A): консольное приложение
// Windows, завершённое по Ctrl+C/Ctrl+Break или TerminateProcess.
const controlCExitStatus = 3221225786

// stopError возвращает errStoppedByUs, если ненулевой выход процесса вызван нашей остановкой:
// контекст отменён И процесс либо завершён сигналом (SIGINT/SIGTERM/Kill), либо вышел с кодом
// STATUS_CONTROL_C_EXIT. Иначе — nil (решать по обычной ошибке выхода).
func stopError(cmd *exec.Cmd, ctxErr error, waitErr error) error {
	if ctxErr == nil || waitErr == nil {
		return nil
	}
	ps := cmd.ProcessState
	if ps == nil {
		return errStoppedByUs // процесс убит, состояния выхода нет
	}
	if !ps.Exited() || ps.ExitCode() == controlCExitStatus {
		return errStoppedByUs
	}
	return nil
}

// fixStopError приводит ошибку завершения cmd к errStoppedByUs, если это была наша остановка,
// а exec вернул nil (Windows: убитый процесс с кодом STATUS_CONTROL_C_EXIT). Используется в
// RunInfo, где нельзя просто проигнорировать «успешный» выход отменённого процесса.
func fixStopError(ctx context.Context, cmd *exec.Cmd, err error) error {
	if err != nil {
		return err
	}
	if ctx.Err() == nil {
		return nil
	}
	ps := cmd.ProcessState
	if ps == nil || !ps.Exited() || ps.ExitCode() == controlCExitStatus {
		return errStoppedByUs
	}
	return nil
}

// StoppedByUs сообщает, что процесс завершился из-за нашей остановки (пауза/отмена), а не сам:
// контекст отменён И процесс завершён сигналом (SIGINT/SIGTERM/Kill) либо кодом
// STATUS_CONTROL_C_EXIT (Windows). Используется менеджером задач, чтобы не принимать ложную
// «отмену» (например, Ctrl+C в родительской консоли) за реальную остановку загрузки.
func StoppedByUs(ctx context.Context, cmd *exec.Cmd) bool {
	if ctx.Err() == nil {
		return false
	}
	ps := cmd.ProcessState
	return ps == nil || !ps.Exited() || ps.ExitCode() == controlCExitStatus
}

// Alive проверяет, жив ли ещё запущенный процесс cmd (не завершился ли). Нужна менеджеру задач,
// чтобы отличить ложную «отмену контекста» на Windows (сигнал консоли пометил ctx отменённым, но
// процесс yt-dlp продолжает качать) от реальной остановки: если процесс жив — задача не паузится.
// Требует уже выполненного cmd.Start(). Реализация кроссплатформенная: signal 0 ничего не делает,
// но возвращает ошибку для завершённого процесса.
func Alive(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return false
	}
	return cmd.Process.Signal(syscall.Signal(0)) == nil
}

// newCommand создаёт команду yt-dlp с привязкой к контексту и мягкой остановкой:
// по отмене ctx посылается сигнал прерывания (yt-dlp успевает прибрать .part-файлы), а если
// процесс не завершился за graceKill — убивается через WaitDelay. На Windows сигналы процессам
// не доставляются — там процесс завершается сразу.
func newCommand(ctx context.Context, bin string, args []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, bin, args...)
	// Чистое окружение: внутри AppImage AppRun задаёт PYTHONHOME и др., что ломает yt-dlp (Python).
	cmd.Env = procenv.Env()
	// На Windows — без окна терминала.
	cmd.SysProcAttr = procattr.HideConsole()
	cmd.Cancel = func() error {
		if runtime.GOOS == "windows" {
			return cmd.Process.Kill()
		}
		return cmd.Process.Signal(os.Interrupt)
	}
	cmd.WaitDelay = graceKill
	return cmd
}

// scanOutput читает stdout построчно, передавая разобранный прогресс и итоговые пути файлов.
// Остальные строки (не прогресс) при наличии h.Log пишутся в лог для диагностики.
func scanOutput(r io.Reader, h Handlers) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	for scanner.Scan() {
		line := scanner.Text()
		if p, ok := parseProgressLine(line); ok {
			if h.Progress != nil {
				h.Progress(p)
			}
			continue
		}
		if h.Log != nil {
			fmt.Fprintln(h.Log, line)
		}
		if path, ok := parseFilePathLine(line); ok && h.File != nil {
			h.File(path)
		}
	}
	return scanner.Err()
}

// parseFilePathLine извлекает итоговый путь файла из строки `--print after_move:...`.
func parseFilePathLine(line string) (string, bool) {
	path, ok := strings.CutPrefix(line, filePrefix)
	if !ok || path == "" {
		return "", false
	}
	return path, true
}

// stderrSuffix добавляет к тексту ошибки хвост stderr (если он непустой), ограничивая длину.
func stderrSuffix(stderr string) string {
	s := strings.TrimSpace(stderr)
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) > errTailLimit {
		s = "…" + string(runes[len(runes)-errTailLimit:])
	}
	return ": " + s
}
