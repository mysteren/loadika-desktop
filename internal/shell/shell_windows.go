//go:build windows

package shell

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Реализация через ShellExecuteW — штатный способ Windows открыть путь приложением по
// умолчанию. Корректно работает с файлами, каталогами и URL и не ломается на путях с
// пробелами/юникодом (в отличие от `rundll32 url.dll,FileProtocolHandler` и
// `explorer /select,<path>`, где Go заключал аргумент в кавычки целиком и explorer
// открывал не ту папку).

var (
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
)

// swShowNormal — SW_SHOWNORMAL: показать окно в обычном размере.
const swShowNormal = 1

// shellExecute вызывает ShellExecuteW; значение больше 32 — успех.
func shellExecute(verb, file, params string) error {
	verbPtr, err := syscall.UTF16PtrFromString(verb)
	if err != nil {
		return err
	}
	filePtr, err := syscall.UTF16PtrFromString(file)
	if err != nil {
		return err
	}
	var paramsPtr *uint16
	if params != "" {
		paramsPtr, err = syscall.UTF16PtrFromString(params)
		if err != nil {
			return err
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
	if ret <= 32 {
		return fmt.Errorf("ShellExecuteW(%q): код %d", file, ret)
	}
	return nil
}

func open(path string) error {
	return shellExecute("open", path, "")
}

func reveal(path string) error {
	// explorer /select,"<path>" — открыть папку и выделить файл; путь в кавычках (пробелы).
	return shellExecute("open", "explorer.exe", `/select,"`+path+`"`)
}
