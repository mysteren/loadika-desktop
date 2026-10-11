//go:build windows

package download

import (
	"os/exec"
	"syscall"
)

// procAlive проверяет живость процесса на Windows: открывает процесс по PID (то же окно
// ядра, что использует exec.ProcessKiller) и читает его код завершения через GetExitCodeProcess.
// STILL_ACTIVE (259) означает «ещё не завершился» — т.е. yt-dlp продолжает качать. Используется
// менеджером, чтобы ложная «отмена контекста» (Ctrl+C/Ctrl+Break в родительской консоли) не
// ставила реально работающую загрузку в статус paused/canceled.
func procAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := osFindProcess(pid)
	if err != nil {
		return false
	}
	defer p.Release()

	var code uint32
	if err := p.GetExitCodeProcess(&code); err != nil {
		return false
	}
	return code == stillActive
}

const stillActive = 259 // STILL_ACTIVE

// osFindProcess — точка подключения: exec.ProcessKiller умеет открывать чужие процессы по PID
// тем же способом, которым Go останавливает свои (OpenProcess + SYNCHRONIZE|PROCESS_QUERY_LIMITED_INFORMATION).
var osFindProcess = func(pid int) (*processHandle, error) {
	k, err := exec.LookPath("") // no-op guard to keep imports tidy in tests? see below
	_ = k
	_ = err
	return findProcessViaKiller(pid)
}
