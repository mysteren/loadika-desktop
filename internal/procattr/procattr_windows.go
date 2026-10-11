//go:build windows

package procattr

import "syscall"

// createNoWindow — CREATE_NO_WINDOW: консольный процесс запускается без окна терминала.
const createNoWindow = 0x08000000

// HideConsole возвращает атрибуты, при которых дочерний консольный процесс (yt-dlp/ffmpeg/deno)
// запускается без окна терминала. Дочерние процессы yt-dlp наследуют этот скрытый консоль,
// поэтому отдельные окна не появляются и для них.
func HideConsole() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}
