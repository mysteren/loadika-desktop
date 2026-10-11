// Package procenv формирует «чистое» окружение для внешних инструментов
// (yt-dlp, ffmpeg, deno), запускаемых приложением.
//
// Приложение может работать из AppImage: его AppRun выставляет переменные окружения,
// нужные бандлу GTK/WebKit (в частности PYTHONHOME=$APPDIR/usr, PYTHONPATH и
// LD_LIBRARY_PATH=$APPDIR/usr/lib). Своему процессу они нужны, но сторонним инструментам
// вредят: с чужим PYTHONHOME системный Python (которым запускается yt-dlp) не находит
// стандартную библиотеку — «Fatal Python error: Failed to import encodings module».
// А LD_LIBRARY_PATH подсовывает инструментам библиотеки из AppImage вместо системных.
//
// Поэтому внутри AppImage (когда задан APPDIR) эти переменные из окружения дочернего
// процесса убираются. Вне AppImage окружение не трогаем.
package procenv

import (
	"os"
	"strings"
)

// appImageVars — переменные, которые выставляет AppRun AppImage и которые нельзя
// пробрасывать внешним инструментам.
var appImageVars = map[string]bool{
	"PYTHONHOME":      true,
	"PYTHONPATH":      true,
	"PYTHONSTARTUP":   true,
	"LD_LIBRARY_PATH": true,
}

// Env возвращает окружение для запуска внешнего инструмента.
func Env() []string {
	return envFrom(os.Environ(), os.Getenv("APPDIR"))
}

// envFrom — тестируемое ядро: при непустом appDir (AppImage) удаляет «вредные» переменные.
func envFrom(environ []string, appDir string) []string {
	if appDir == "" {
		return environ
	}
	out := make([]string, 0, len(environ))
	for _, kv := range environ {
		key, _, ok := strings.Cut(kv, "=")
		if ok && appImageVars[key] {
			continue
		}
		out = append(out, kv)
	}
	return out
}
