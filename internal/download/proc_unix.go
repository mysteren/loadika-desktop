//go:build !windows

package download

import "syscall"

// procAlive проверяет живость процесса по PID. На Unix signal(0) ничего не делает, но
// возвращает ошибку, если процесс завершён (или это чужой процесс без прав — тогда считаем
// живым и не трогаем; ложная пауза хуже, чем лишняя строка в логе).
func procAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	if err == nil || err == syscall.EPERM {
		return true
	}
	return false // ESRCH — процесс завершился
}
