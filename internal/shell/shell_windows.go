//go:build windows

package shell

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// Реализация через ShellExecuteW — штатный способ Windows открыть путь приложением по
// умолчанию. Корректно работает с файлами, каталогами и URL и не ломается на путях с
// пробелами/юникодом (в отличие от `rundll32 url.dll,FileProtocolHandler` и
// `explorer /select,<path>`, где Go заключал аргумент в кавычки целиком и explorer
// открывал не ту папку).
//
// Длинные пути (> MAX_PATH = 260 символов) ShellExecuteW не принимает и возвращает код 2
// (SE_ERR_FILENOTFOUND), даже если файл существует — имена торрентов часто длинные. Для
// них подбирается короткое 8.3-имя или префикс \\?\ (см. longpath_windows.go); если и это
// не помогло, пользователь получает понятное сообщение вместо «код 2».

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
)

// swShowNormal — SW_SHOWNORMAL: показать окно в обычном размере.
const swShowNormal = 1

// isURL — ShellExecuteW сам разберёт http/https-адрес; проверять его на диске нельзя.
func isURL(p string) bool {
	lower := strings.ToLower(p)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "file://")
}

// shellExecuteOne однократно вызывает ShellExecuteW; возвращает код результата
// (значение больше 32 — успех, иначе код ошибки SE_ERR_*).
func shellExecuteOne(verb, file, params string) (uintptr, error) {
	verbPtr, err := syscall.UTF16PtrFromString(verb)
	if err != nil {
		return 0, err
	}
	filePtr, err := syscall.UTF16PtrFromString(file)
	if err != nil {
		return 0, err
	}
	var paramsPtr *uint16
	if params != "" {
		paramsPtr, err = syscall.UTF16PtrFromString(params)
		if err != nil {
			return 0, err
		}
	}

	ret, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(filePtr)),
		uintptr(unsafe.Pointer(paramsPtr)),
		0,
		swShowNormal,
	)
	lastShellErrCode = ret
	return ret, nil
}

// shellExecute вызывает ShellExecuteW, для длинных путей перебирая допустимые формы пути
// (исходная → 8.3-имя → \\?\). При неудаче возвращает понятную ошибку.
func shellExecute(verb, file, params string) error {
	for _, variant := range pathVariants(file) {
		ret, err := shellExecuteOne(verb, variant, params)
		if err != nil {
			return err
		}
		if ret > 32 {
			return nil
		}
		// Код 2 для несуществующего файла бессмысленно повторять в других формах.
		if ret == seErrFileNotFound && !isURL(file) && !existsOnDisk(file) {
			break
		}
	}
	// Все попытки не удались: сообщаем ошибку по исходному пути.
	return friendlyErrorLast(verb, file)
}

func open(path string) error {
	if !isURL(path) {
		settleBeforeOpen(path)
	}
	// Для файлов — с запасным вариантом через известные плееры (см. openWithFallback);
	// каталоги и URL открываем штатно.
	if !isURL(path) && !isDir(path) {
		return openWithFallback(path)
	}
	return shellExecute("open", path, "")
}

// isDir — каталог ли это (для каталога фолбэк «открыть в плеере» не нужен).
func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func reveal(path string) error {
	// explorer /select,"<path>" — открыть папку и выделить файл; путь в кавычках (пробелы).
	// ВАЖНО: если путь не разобрался, explorer молча открывает «Этот компьютер» (диски) без
	// ошибки — поэтому для путей длиннее 260 символов сначала подбираем разборную форму
	// (8.3-хвост последних компонентов, затем \\?\), а если короткой формы нет — открываем
	// хотя бы содержащий каталог (обычные папки редко бывают длиннее 260 символов).
	if utf16Len(path) <= maxPathLegacy {
		return shellExecute("open", "explorer.exe", `/select,"`+path+`"`)
	}
	sel := pickShortVariant(path)
	if utf16Len(sel) <= maxPathLegacy && existsOnDisk(sel) {
		return shellExecute("open", "explorer.exe", `/select,"`+sel+`"`)
	}
	// Выделение недоступного пути explorer проигнорирует — выделять нечего, открываем папку.
	dir := filepath.Dir(path)
	if d := pickShortVariant(dir); utf16Len(d) < utf16Len(dir) {
		dir = d
	}
	return shellExecute("open", dir, "")
}
