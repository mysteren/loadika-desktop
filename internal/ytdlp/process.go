package ytdlp

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
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

	scanErr := scanOutput(stdout, h)
	waitErr := cmd.Wait()

	// Отмену/таймаут контекста сообщаем отдельно и понятно; вызывающий различает её через errors.Is.
	if ctxErr := ctx.Err(); ctxErr != nil {
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

	if err := cmd.Run(); err != nil {
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

// newCommand создаёт команду yt-dlp с привязкой к контексту и мягкой остановкой:
// по отмене ctx посылается сигнал прерывания (yt-dlp успевает прибрать .part-файлы), а если
// процесс не завершился за graceKill — убивается через WaitDelay. На Windows сигналы процессам
// не доставляются — там процесс завершается сразу.
func newCommand(ctx context.Context, bin string, args []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, bin, args...)
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
