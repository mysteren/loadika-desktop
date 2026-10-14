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
// процесса убираются. Вне AppImage окружение не трогаем (кроме переменных Python, см. pythonVars).
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

// pythonVars — переменные Python, задаваемые дочерним инструментам (yt-dlp).
//
// PYTHONUTF8/PYTHONIOENCODING — UTF-8 для stdio. yt-dlp — Python-программа: когда её stdout/stderr
// перенаправлены в канал (а не в консоль), CPython на Windows кодирует вывод кодировкой локали
// (cp1251 для русской Windows), а не UTF-8. Go читает вывод как UTF-8, поэтому кириллица в пути
// файла (`--print after_move`) и в заголовках (`-J`) превращается в «кракозябры».
//
// PYTHONUNBUFFERED — без блочной буферизации stdout/stderr: иначе строки прогресса накапливаются
// в буфере и приходят в Go пачками, и UI обновляет скорость/прогресс рывками. С ним каждая строка
// доходит сразу — обновления ровные.
var pythonVars = [][2]string{
	{"PYTHONUTF8", "1"},           // режим UTF-8 (PEP 540) для stdio и файловой системы
	{"PYTHONIOENCODING", "utf-8"}, // кодировка стандартного ввода-вывода
	{"PYTHONUNBUFFERED", "1"},     // stdout/stderr без блочной буферизации (ровный прогресс без рывков)
}

// Env возвращает окружение для запуска внешнего инструмента.
func Env() []string {
	return withPythonEnv(envFrom(os.Environ(), os.Getenv("APPDIR")))
}

// withPythonEnv задаёт переменные Python (pythonVars), заменяя уже существующие значения — вывод
// наших Python-инструментов всегда должен быть UTF-8 и не буферизованным, как его читает Go.
func withPythonEnv(environ []string) []string {
	out := append([]string(nil), environ...)
	for _, kv := range pythonVars {
		out = setEnv(out, kv[0], kv[1])
	}
	return out
}

// setEnv задаёт переменную key=value, заменяя прежнее значение, если оно было.
func setEnv(environ []string, key, value string) []string {
	prefix := key + "="
	for i, kv := range environ {
		if strings.HasPrefix(kv, prefix) {
			environ[i] = prefix + value
			return environ
		}
	}
	return append(environ, prefix+value)
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
