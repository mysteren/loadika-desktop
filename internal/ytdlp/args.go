// Package ytdlp формирует аргументы и запускает процесс yt-dlp, разбирая его
// машинно-читаемый вывод (прогресс, метаданные). Доменный пакет, не зависит от Wails.
package ytdlp

import (
	"errors"
)

// args.go намеренно не содержит «свидетелей работы» процесса (маркеров вывода): статус задачи
// менеджер определяет по факту — коду выхода, наличию .part-файла и живости процесса (см.
// internal/download/manager.go). Маркеры stdout ненадёжны: их текст зависит от версии yt-dlp и
// набора аргументов.

// outputTemplate — шаблон имени файла по умолчанию. Каталог задаётся отдельно через -P.
// Разрешение в имени нужно, чтобы разные качества одного видео не конфликтовали по имени: иначе
// yt-dlp находит уже скачанный файл и пропускает загрузку (а не качает новое качество).
const outputTemplate = "%(title)s [%(id)s] [%(resolution)s].%(ext)s"

// progressPrefix — префикс строки прогресса в машинно-читаемом выводе.
const progressPrefix = "loadika-progress:"

// progressTemplate — шаблон прогресса: JSON-объект прогресса yt-dlp (%(progress)j).
// Позволяет разбирать поток без парсинга «человеческого» текста.
const progressTemplate = progressPrefix + "%(progress)j"

// filePrefix — префикс строки с итоговым путём файла (--print after_move). Без двоеточия,
// чтобы не конфликтовать с разделителем WHEN в --print.
const filePrefix = "loadika-file="

// filePrintTemplate — шаблон --print: после перемещения файла печатаем его путь с префиксом.
// Именно after_move даёт путь ПОСЛЕ склейки/post-processing (см. документацию yt-dlp).
const filePrintTemplate = "after_move:" + filePrefix + "%(filepath)s"

// socketTimeout — таймаут сокета yt-dlp (секунды): зависшее соединение прерывается раньше и даёт
// понятную ошибку (а не бесконечное ожидание).
const socketTimeout = "30"

// restrictFilenames — аргумент yt-dlp: в имени файла остаются только ASCII-символы, пробелы
// заменяются подчёркиваниями. Кириллические/иные не-ASCII имена торрентов легко выводят путь за
// MAX_PATH (260 символов) — Windows тогда отказывает открыть файл (ShellExecuteW код 2), даже
// если он скачан. С этим флагом имена получаются короткими и латиницей; метаданные (заголовок
// страницы) в интерфейсе при этом остаются оригинальными — они читаются из JSON отдельно.
const restrictFilenames = "--restrict-filenames"

// Options — параметры одиночной загрузки.
type Options struct {
	URL            string // ссылка
	OutputDir      string // каталог загрузки (-P)
	Format         string // селектор формата (-f); пусто — yt-dlp выберет лучший
	Proxy          string // URL прокси (--proxy); пусто — не передаём
	DenoPath       string // путь к deno для --js-runtimes; пусто — не передаём
	FFmpegDir      string // каталог ffmpeg/ffprobe для --ffmpeg-location; пусто — не передаём
	CookiesBrowser string // браузер для --cookies-from-browser; пусто — не передаём
	CookiesFile    string // файл cookies для --cookies; пусто — не передаём
}

// InfoOptions — параметры запроса метаданных (`-J`): заголовок и список форматов.
type InfoOptions struct {
	URL            string // ссылка
	Proxy          string // URL прокси (--proxy); пусто — не передаём
	DenoPath       string // путь к deno для --js-runtimes; пусто — не передаём
	CookiesBrowser string // браузер для --cookies-from-browser; пусто — не передаём
	CookiesFile    string // файл cookies для --cookies; пусто — не передаём
}

// withCookies добавляет аргументы авторизации cookies. Файл имеет приоритет над браузером
// (если заданы оба). Cookies помогают обойти проверку YouTube «Sign in to confirm you're not a bot».
func withCookies(args []string, browser, file string) []string {
	if file != "" {
		return append(args, "--cookies", file)
	}
	if browser != "" {
		return append(args, "--cookies-from-browser", browser)
	}
	return args
}

// BuildArgs собирает аргументы командной строки yt-dlp. Функция чистая: результат
// зависит только от Options. Пустой URL или каталог загрузки — ошибка.
func BuildArgs(o Options) ([]string, error) {
	if o.URL == "" {
		return nil, errors.New("не задана ссылка для загрузки")
	}
	if o.OutputDir == "" {
		return nil, errors.New("не задан каталог загрузки")
	}

	args := []string{
		"--newline",                       // прогресс — отдельной строкой
		"--no-color",                      // без ANSI-цветов
		"--socket-timeout", socketTimeout, // зависшее соединение прерывается раньше
		restrictFilenames,                 // короткие ASCII-имена файлов (см. выше)
		// --print включает --quiet, который глушит прогресс; --progress возвращает его.
		"--progress",
		"--progress-template", progressTemplate,
		"--no-simulate", // --print иначе включил бы симуляцию
		"--print", filePrintTemplate,
		"-P", o.OutputDir,
		"-o", outputTemplate,
	}

	if o.Format != "" {
		args = append(args, "-f", o.Format)
	}
	if o.Proxy != "" {
		args = append(args, "--proxy", o.Proxy)
	}
	if o.DenoPath != "" {
		args = append(args, "--js-runtimes", "deno:"+o.DenoPath)
	}
	if o.FFmpegDir != "" {
		args = append(args, "--ffmpeg-location", o.FFmpegDir)
	}
	args = withCookies(args, o.CookiesBrowser, o.CookiesFile)

	return append(args, o.URL), nil
}

// BuildInfoArgs собирает аргументы yt-dlp для получения метаданных в машинно-читаемом виде
// (--dump-single-json). Функция чистая. Пустой URL — ошибка.
func BuildInfoArgs(o InfoOptions) ([]string, error) {
	if o.URL == "" {
		return nil, errors.New("не задана ссылка для получения метаданных")
	}

	args := []string{
		"--dump-single-json", // один JSON-объект со всеми полями, включая форматы
		"--no-color",
		"--no-warnings",
		"--no-playlist", // ссылку обрабатываем как одно видео
		"--socket-timeout", socketTimeout,
	}
	if o.Proxy != "" {
		args = append(args, "--proxy", o.Proxy)
	}
	if o.DenoPath != "" {
		args = append(args, "--js-runtimes", "deno:"+o.DenoPath)
	}
	args = withCookies(args, o.CookiesBrowser, o.CookiesFile)

	return append(args, o.URL), nil
}
