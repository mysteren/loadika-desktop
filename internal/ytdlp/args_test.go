package ytdlp

import "testing"

// flagValue возвращает значение, идущее сразу за флагом (или "", если флага нет).
func flagValue(args []string, flag string) (string, bool) {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func TestBuildArgsMinimal(t *testing.T) {
	args, err := BuildArgs(Options{URL: "https://example.com/v", OutputDir: "/tmp/dl"})
	if err != nil {
		t.Fatalf("BuildArgs: %v", err)
	}

	if args[len(args)-1] != "https://example.com/v" {
		t.Errorf("последний аргумент = %q, want URL", args[len(args)-1])
	}
	for _, flag := range []string{"--newline", "--no-color", "--progress-template"} {
		if !hasFlag(args, flag) {
			t.Errorf("нет флага %q", flag)
		}
	}
	if got, _ := flagValue(args, "-P"); got != "/tmp/dl" {
		t.Errorf("-P = %q, want /tmp/dl", got)
	}
	if got, _ := flagValue(args, "-o"); got != outputTemplate {
		t.Errorf("-o = %q, want %q", got, outputTemplate)
	}
	if hasFlag(args, "--js-runtimes") || hasFlag(args, "--ffmpeg-location") {
		t.Error("не должно быть флагов зависимостей без путей")
	}
}

func TestBuildArgsWithDependencies(t *testing.T) {
	args, err := BuildArgs(Options{
		URL:       "https://example.com/v",
		OutputDir: "/tmp/dl",
		DenoPath:  "/deps/deno",
		FFmpegDir: "/deps/ffmpeg",
	})
	if err != nil {
		t.Fatalf("BuildArgs: %v", err)
	}

	if got, _ := flagValue(args, "--js-runtimes"); got != "deno:/deps/deno" {
		t.Errorf("--js-runtimes = %q", got)
	}
	if got, _ := flagValue(args, "--ffmpeg-location"); got != "/deps/ffmpeg" {
		t.Errorf("--ffmpeg-location = %q", got)
	}
	if args[len(args)-1] != "https://example.com/v" {
		t.Errorf("URL должен идти последним: %q", args[len(args)-1])
	}
}

func TestBuildArgsErrors(t *testing.T) {
	if _, err := BuildArgs(Options{OutputDir: "/tmp/dl"}); err == nil {
		t.Error("ожидалась ошибка при пустом URL")
	}
	if _, err := BuildArgs(Options{URL: "https://example.com/v"}); err == nil {
		t.Error("ожидалась ошибка при пустом каталоге загрузки")
	}
}

func TestBuildInfoArgs(t *testing.T) {
	args, err := BuildInfoArgs(InfoOptions{URL: "https://example.com/v"})
	if err != nil {
		t.Fatalf("BuildInfoArgs: %v", err)
	}

	for _, flag := range []string{"--dump-single-json", "--no-color", "--no-warnings", "--no-playlist"} {
		if !hasFlag(args, flag) {
			t.Errorf("нет флага %q", flag)
		}
	}
	if args[len(args)-1] != "https://example.com/v" {
		t.Errorf("URL должен идти последним: %q", args[len(args)-1])
	}
	if hasFlag(args, "--js-runtimes") {
		t.Error("не должно быть --js-runtimes без пути к deno")
	}

	withDeno, err := BuildInfoArgs(InfoOptions{URL: "https://example.com/v", DenoPath: "/deps/deno"})
	if err != nil {
		t.Fatalf("BuildInfoArgs: %v", err)
	}
	if got, _ := flagValue(withDeno, "--js-runtimes"); got != "deno:/deps/deno" {
		t.Errorf("--js-runtimes = %q", got)
	}

	withProxy, err := BuildInfoArgs(InfoOptions{URL: "https://example.com/v", Proxy: "socks5://1.2.3.4:1080"})
	if err != nil {
		t.Fatalf("BuildInfoArgs: %v", err)
	}
	if got, _ := flagValue(withProxy, "--proxy"); got != "socks5://1.2.3.4:1080" {
		t.Errorf("--proxy = %q", got)
	}

	if _, err := BuildInfoArgs(InfoOptions{}); err == nil {
		t.Error("ожидалась ошибка при пустом URL")
	}
}

func TestBuildArgsCookies(t *testing.T) {
	// Файл cookies имеет приоритет над браузером.
	both, err := BuildArgs(Options{
		URL:            "https://example.com/v",
		OutputDir:      "/tmp/dl",
		CookiesBrowser: "firefox",
		CookiesFile:    "/tmp/cookies.txt",
	})
	if err != nil {
		t.Fatalf("BuildArgs: %v", err)
	}
	if got, _ := flagValue(both, "--cookies"); got != "/tmp/cookies.txt" {
		t.Errorf("--cookies = %q", got)
	}
	if hasFlag(both, "--cookies-from-browser") {
		t.Error("при заданном файле флаг --cookies-from-browser не нужен")
	}

	// Только браузер.
	browserOnly, err := BuildArgs(Options{URL: "https://example.com/v", OutputDir: "/tmp/dl", CookiesBrowser: "firefox"})
	if err != nil {
		t.Fatalf("BuildArgs: %v", err)
	}
	if got, _ := flagValue(browserOnly, "--cookies-from-browser"); got != "firefox" {
		t.Errorf("--cookies-from-browser = %q", got)
	}

	// Без cookies — ни одного из флагов.
	plain, err := BuildArgs(Options{URL: "https://example.com/v", OutputDir: "/tmp/dl"})
	if err != nil {
		t.Fatalf("BuildArgs: %v", err)
	}
	if hasFlag(plain, "--cookies") || hasFlag(plain, "--cookies-from-browser") {
		t.Error("не должно быть флагов cookies без настроек")
	}

	// И для запроса метаданных тоже.
	info, err := BuildInfoArgs(InfoOptions{URL: "https://example.com/v", CookiesFile: "/tmp/cookies.txt"})
	if err != nil {
		t.Fatalf("BuildInfoArgs: %v", err)
	}
	if got, _ := flagValue(info, "--cookies"); got != "/tmp/cookies.txt" {
		t.Errorf("--cookies (info) = %q", got)
	}
}

func TestBuildArgsFormatSelector(t *testing.T) {
	args, err := BuildArgs(Options{URL: "https://example.com/v", OutputDir: "/tmp/dl", Format: "137+140"})
	if err != nil {
		t.Fatalf("BuildArgs: %v", err)
	}
	if got, ok := flagValue(args, "-f"); !ok || got != "137+140" {
		t.Errorf("-f = %q, ok=%v; want 137+140", got, ok)
	}

	plain, err := BuildArgs(Options{URL: "https://example.com/v", OutputDir: "/tmp/dl"})
	if err != nil {
		t.Fatalf("BuildArgs: %v", err)
	}
	if hasFlag(plain, "-f") {
		t.Error("не должно быть -f без селектора формата")
	}
}

func TestBuildArgsFilePathCapture(t *testing.T) {
	args, err := BuildArgs(Options{URL: "https://example.com/v", OutputDir: "/tmp/dl"})
	if err != nil {
		t.Fatalf("BuildArgs: %v", err)
	}

	// --print должен давать путь после post-processing с нашим префиксом.
	if got, ok := flagValue(args, "--print"); !ok || got != "after_move:loadika-file=%(filepath)s" {
		t.Errorf("--print = %q (ok=%v)", got, ok)
	}
	for _, flag := range []string{"--progress", "--no-simulate"} {
		if !hasFlag(args, flag) {
			t.Errorf("нет флага %q", flag)
		}
	}
}
