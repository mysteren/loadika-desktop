//go:build windows

package download

import (
	"syscall"
	"time"
	"unsafe"
)

// Проверка живости процесса yt-dlp по PID. Нужна, чтобы отличить ложную «отмену контекста»
// на Windows (Ctrl+C/Ctrl+Break в родительской консоли помечает ctx отменённым, но процесс,
// запущенный с CREATE_NO_WINDOW, сигнал не получает и продолжает качать) от реальной
// остановки загрузки.
//
// Реализация: OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION) + GetExitCodeProcess.
// STILL_ACTIVE (259) означает «процесс ещё не завершился». Дополнительно сверяем время
// создания процесса (GetProcessTimes) со временем запуска попытки: завершённый, но ещё не
// собранный процесс (zombie) или чужой PID, повторно выданный системой, отсекаются по возрасту.

const (
	stillActive                    = 259    // STILL_ACTIVE
	processQueryLimitedInformation = 0x1000 // PROCESS_QUERY_LIMITED_INFORMATION
)

var (
	kernel32Proc           = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess        = kernel32Proc.NewProc("OpenProcess")
	procGetExitCodeProcess = kernel32Proc.NewProc("GetExitCodeProcess")
	procGetProcessTimes    = kernel32Proc.NewProc("GetProcessTimes")
)

// procAlive проверяет, жив ли процесс с указанным PID. startedAt — момент запуска текущей
// попытки загрузки; если процесс создан заметно раньше, он не может быть нашим (нулевой
// startedAt — возраст не учитываем).
func procAlive(pid int, startedAt time.Time) bool {
	if pid <= 0 {
		return false
	}
	handle, _, _ := procOpenProcess.Call(
		uintptr(processQueryLimitedInformation),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		// Отказать в открытии может системе-процессу или уже собранному завершённому процессу.
		return false
	}
	defer syscall.CloseHandle(syscall.Handle(handle))

	var exitCode uint32
	r, _, _ := procGetExitCodeProcess.Call(handle, uintptr(unsafe.Pointer(&exitCode)))
	if r == 0 || exitCode != stillActive {
		return false
	}

	// Формально «активен»: отсекаем zombie завершившегося процесса и чужие PID по времени
	// создания — настоящий yt-dlp нашей попытки создан не раньше старта попытки.
	if !startedAt.IsZero() {
		if ct, ok := processCreationTime(syscall.Handle(handle)); ok && ct.Before(startedAt.Add(-time.Second)) {
			return false
		}
	}
	return true
}

// filetime100ns — FILETIME: такты по 100 нс с 1601-01-01.
type filetime100ns struct {
	low  uint32
	high uint32
}

func (ft filetime100ns) toUnixNano() int64 {
	const epochDiff = 116444736000000000 // тактов между эпохами 1601 и 1970
	v := int64(ft.high)<<32 | int64(ft.low)
	return (v - epochDiff) * 100 // наносекунды
}

// processCreationTime возвращает время создания процесса по хэндлу.
// ok=false — время недоступно (тогда возраст не учитываем).
func processCreationTime(h syscall.Handle) (t time.Time, ok bool) {
	var creation, exit, kernel, user filetime100ns
	r, _, _ := procGetProcessTimes.Call(
		uintptr(h),
		uintptr(unsafe.Pointer(&creation)),
		uintptr(unsafe.Pointer(&exit)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if r == 0 {
		return time.Time{}, false
	}
	ns := creation.toUnixNano()
	if ns <= 0 {
		return time.Time{}, false
	}
	return time.Unix(0, ns), true
}
