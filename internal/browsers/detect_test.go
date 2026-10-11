package browsers

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// makeFile создаёт файл (и родительские каталоги) для имитации cookie-базы браузера.
func makeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// findBrowser находит браузер по имени среди обнаруженных.
func findBrowser(list []Browser, name string) (Browser, bool) {
	for _, b := range list {
		if b.Name == name {
			return b, true
		}
	}
	return Browser{}, false
}

func TestDetectLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("тест проверяет Linux-пути")
	}

	home := t.TempDir()
	config := filepath.Join(home, ".config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", config)

	// Firefox (XDG-раскладка FF147+): профиль с cookies.sqlite.
	makeFile(t, filepath.Join(config, "mozilla", "firefox", "abc123.default-release", "cookies.sqlite"))
	// Firefox (старая раскладка): второй профиль.
	makeFile(t, filepath.Join(home, ".mozilla", "firefox", "xyz789.dev", "cookies.sqlite"))

	// Chrome: профиль Default с Cookies; служебный каталог без cookies — игнорируем.
	makeFile(t, filepath.Join(config, "google-chrome", "Default", "Cookies"))
	_ = os.MkdirAll(filepath.Join(config, "google-chrome", "Crashpad"), 0o755)

	// Chromium: новый формат — cookies в Network/.
	makeFile(t, filepath.Join(config, "chromium", "Profile 1", "Network", "Cookies"))

	got := Detect()

	firefox, ok := findBrowser(got, "firefox")
	if !ok {
		t.Fatalf("firefox не обнаружен: %+v", got)
	}
	if len(firefox.Profiles) != 2 {
		t.Fatalf("профилей firefox %d, want 2: %+v", len(firefox.Profiles), firefox.Profiles)
	}
	// Оба профиля — по алфавиту; метка укорачивает хеш до первой точки.
	if firefox.Profiles[0].Name != "abc123.default-release" || firefox.Profiles[0].Label != "default-release" {
		t.Errorf("профиль firefox[0] = %+v", firefox.Profiles[0])
	}
	if firefox.Profiles[1].Name != "xyz789.dev" || firefox.Profiles[1].Label != "dev" {
		t.Errorf("профиль firefox[1] = %+v", firefox.Profiles[1])
	}

	chrome, ok := findBrowser(got, "chrome")
	if !ok || len(chrome.Profiles) != 1 || chrome.Profiles[0].Name != "Default" {
		t.Fatalf("chrome = %+v, want один профиль Default", chrome)
	}

	chromium, ok := findBrowser(got, "chromium")
	if !ok || len(chromium.Profiles) != 1 || chromium.Profiles[0].Name != "Profile 1" {
		t.Fatalf("chromium = %+v, want один профиль Profile 1", chromium)
	}

	// Браузеры без cookies не попадают в список.
	for _, name := range []string{"brave", "edge", "vivaldi", "opera", "whale"} {
		if _, ok := findBrowser(got, name); ok {
			t.Errorf("браузер %q не должен быть обнаружен (нет cookies)", name)
		}
	}
}

func TestDetectEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	if got := Detect(); len(got) != 0 {
		t.Errorf("Detect на пустом окружении = %+v, want пусто", got)
	}
}
