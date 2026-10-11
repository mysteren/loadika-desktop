//go:build windows

package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// Поддержка длинных путей Windows (> MAX_PATH = 260 символов). ShellExecuteW возвращает
// SE_ERR_FILENOTFOUND (код 2), если путь слишком длинный и не использует префикс \\?\ —
// файл при этом существует. Имена торрентов часто длинные, поэтому путь в папке загрузок
// легко превышает 260 символов. Решения по порядку:
//   - 8.3-имя (GetShortPathNameW) — работает даже без включения поддержки длинных путей;
//   - префикс \\?\ с нормализованным абсолютным путём;
//   - если ничего не помогло — понятная ошибка вместо голого «кода 2».

const (
	// maxPathLegacy — лимит WinAPI без префикса \\?\.
	maxPathLegacy = 260
	// maxLongPath — лимит Windows с префиксом \\?\.
	maxLongPath = 32767

	seErrFileNotFound = uintptr(2)  // файл/ресурс не найден (в т.ч. из-за длины пути)
	seErrDdeFailed    = uintptr(11) // DDE-обмен с приложением не удался
	seErrNoAssoc      = uintptr(31) // нет ассоциации файла с приложением

	// createDetached — CREATE_NEW_PROCESS_GROUP: запуск без нового окна консоли.
	createDetached = 0x00000200
)

var procGetShortPathNameW = kernel32.NewProc("GetShortPathNameW")

// lastShellErrCode — последний код ошибки ShellExecuteW (для сообщения пользователю).
var lastShellErrCode uintptr

// isUncPath — путь вида \\server\share (для него префикс — \\?\UNC\).
func isUncPath(p string) bool {
	return strings.HasPrefix(p, `\\`) && !strings.HasPrefix(p, `\\?\`) && !strings.HasPrefix(p, `\\.\`)
}

// toExtendedPath переводит путь в форму с префиксом \\?\ (обходит лимит MAX_PATH).
// Требует абсолютного пути с буквой тома (иначе Windows вернёт ERROR_INVALID_NAME).
func toExtendedPath(p string) string {
	if strings.HasPrefix(p, `\\?\`) {
		return p
	}
	if isUncPath(p) {
		return `\\?\UNC\` + p[2:]
	}
	return `\\?\` + filepath.Clean(p)
}

// shortPath возвращает 8.3-имя пути (короткая форма гарантированно короче 260 символов).
// Пустой результат — путь не найден или том без поддержки 8.3 имён.
func shortPath(p string) string {
	src, err := syscall.UTF16PtrFromString(p)
	if err != nil {
		return ""
	}
	for size := uint32(4); size <= maxLongPath; size *= 2 {
		buf := make([]uint16, size)
		ret, _, _ := procGetShortPathNameW.Call(
			uintptr(unsafe.Pointer(src)),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(size),
		)
		switch {
		case ret == 0: // ошибка: нет 8.3 имён или путь недоступен
			return ""
		case ret < uintptr(size): // строка уместилась в буфер
			return syscall.UTF16ToString(buf[:ret])
		}
		// ret >= size — буфер мал, увеличиваем и повторяем
	}
	return ""
}

// existsOnDisk проверяет доступность пути (учитывает формы с префиксом \\?\).
func existsOnDisk(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// pathVariants перечисляет формы пути, которые стоит попробовать в ShellExecuteW:
// исходную, 8.3-имя и расширенную \\?\. Для коротких путей — только исходную.
func pathVariants(p string) []string {
	variants := []string{p}
	if utf16Len(p) <= maxPathLegacy {
		return variants
	}
	if s := shortPath(p); s != "" && s != p {
		variants = append(variants, s)
	}
	if e := toExtendedPath(p); e != p {
		variants = append(variants, e)
	}
	return variants
}

// utf16Len — длина строки в UTF-16 (именно её ограничивает MAX_PATH).
func utf16Len(s string) int {
	u, err := syscall.UTF16FromString(s)
	if err != nil {
		return len(s)
	}
	return len(u) - 1
}

// friendlyErrorLast превращает последний код возврата ShellExecuteW в понятное сообщение.
func friendlyErrorLast(verb, file string) error {
	ret := lastShellErrCode
	if ret == 0 {
		ret = seErrFileNotFound
	}
	switch ret {
	case seErrFileNotFound:
		if existsOnDisk(file) {
			return fmt.Errorf("не удалось открыть %q: файл существует, но Windows отказала "+
				"(код %d). Частые причины: путь длиннее 260 символов, для расширения не "+
				"назначено приложение по умолчанию либо оно установлено из Microsoft Store "+
				"(у таких приложений нет exe-файла для прямого запуска)", file, ret)
		}
		return fmt.Errorf("файл не найден: %q (код %d)", file, ret)
	case seErrDdeFailed:
		return fmt.Errorf("приложение по умолчанию для %q не ответило (код %d): возможно, "+
			"оно повреждено или завершено с ошибкой", file, ret)
	case seErrNoAssoc:
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(file)), ".")
		return fmt.Errorf("для расширения .%s не назначено приложение по умолчанию (код %d): "+
			"«Параметры → Приложения → Приложения по умолчанию»", ext, ret)
	default:
		return fmt.Errorf("%s(%q): код %d", verb, file, ret)
	}
}

// openWithFallback открывает файл: сначала приложением по умолчанию (ShellExecuteW),
// а если оно не найдено/не ответило (коды 2/11/31 — типично для «Кино и телефонов» из
// Microsoft Store и других UWP-приложений, а также для .webm без ассоциации) — напрямую
// через известные плееры.
func openWithFallback(path string) error {
	err := shellExecute("open", path, "")
	if err == nil {
		return nil
	}
	// Файла реально нет (или это URL) — прямой запуск плеера не поможет.
	if isURL(path) || !existsOnDisk(path) {
		return err
	}
	switch lastShellErrCode {
	case seErrFileNotFound, seErrDdeFailed, seErrNoAssoc:
		for _, player := range knownPlayers() {
			// Длинный путь плееру передаём в короткой форме, если она нашлась.
			arg := pickShortVariant(path)
			if ok, _ := tryExec(player, arg); ok {
				return nil
			}
		}
	}
	return err
}

// pickShortVariant выбирает форму пути, которую примет приложение без поддержки
// длинных путей: 8.3-имя либо \\?\; при неудаче — исходный путь.
func pickShortVariant(p string) string {
	if utf16Len(p) <= maxPathLegacy {
		return p
	}
	if s := shortPath(p); s != "" {
		return s
	}
	return toExtendedPath(p)
}

// knownPlayers — видеоплееры, которые умеем запускать напрямую, если ассоциация Windows
// не сработала. Порядок: VLC → MPV → MPC-HC/PotPlayer в типовых каталогах установки.
func knownPlayers() []string {
	var dirs []string
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		dirs = append(dirs, pf)
	}
	if pfx86 := os.Getenv("ProgramFiles(x86)"); pfx86 != "" {
		dirs = append(dirs, pfx86)
	}
	local := os.Getenv("LOCALAPPDATA")
	if local != "" {
		dirs = append(dirs, filepath.Join(local, "Programs"))
	}
	candidates := []string{}
	exes := [][]string{
		{"VideoLAN", "VLC", "vlc.exe"},
		{"mpv", "mpv.exe"},
		{"ClangChaos", "mpc-hc", "mpc-hc64.exe"},
		{"Caffeine", "potplayer", "PotPlayerMini64.exe"},
	}
	for _, d := range dirs {
		for _, e := range exes {
			p := filepath.Join(append([]string{d}, e...)...)
			if existsOnDisk(p) {
				candidates = append(candidates, p)
			}
		}
	}
	// PATH как последний шанс (portable-установки).
	for _, name := range []string{"vlc.exe", "mpv.exe"} {
		if p, err := exec.LookPath(name); err == nil {
			candidates = append(candidates, p)
		}
	}
	return candidates
}

// tryExec запускает exe с аргументом файла DETACHED (без окна консоли); true — процесс стартовал.
func tryExec(exe, file string) (bool, error) {
	cmd := exec.Command(exe, file)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createDetached}
	if err := cmd.Start(); err != nil {
		return false, err
	}
	go func() { _ = cmd.Wait() }() // не ждём: плеер работает своё время
	return true, nil
}

// settleBeforeOpen кратко ждёт появления файла — защита от гонки, когда «Открыть»
// нажимают сразу после завершения загрузки, а файловая система ещё дописывает данные.
func settleBeforeOpen(path string) {
	if existsOnDisk(path) {
		return
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		if existsOnDisk(path) {
			return
		}
	}
}
