// Package browsers обнаруживает установленные браузеры и их профили, из которых yt-dlp может взять
// cookies (`--cookies-from-browser`). Пути и семантика профилей синхронизированы с yt_dlp/cookies.py:
// «профиль» — это имя каталога профиля, которое yt-dlp подставляет к каталогу браузера.
// Доменный пакет, не зависит от Wails.
package browsers

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Profile — профиль браузера. Name — значение для `--cookies-from-browser браузер:профиль`.
type Profile struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}

// Browser — обнаруженный браузер с непустым набором профилей (то есть с сохранёнными cookies).
type Browser struct {
	Name     string    `json:"name"`  // имя для yt-dlp: firefox, chrome, ...
	Label    string    `json:"label"` // человекочитаемое имя
	Profiles []Profile `json:"profiles"`
}

// detector описывает один браузер: где лежат его данные и как искать профили.
type detector struct {
	name    string
	label   string
	roots   func() []string // каталоги, в подкаталогах которых лежат профили
	markers []string        // файлы внутри профиля, подтверждающие наличие cookies
	trim    bool            // укорачивать ли метку профиля (для Firefox — отбросить хеш до первой точки)
}

// chromiumMarkers — файлы cookies у браузеров на Chromium (новые версии хранят в Network/).
var chromiumMarkers = []string{"Cookies", filepath.Join("Network", "Cookies")}

// firefoxMarkers — файл cookies у Firefox.
var firefoxMarkers = []string{"cookies.sqlite"}

// Detect возвращает браузеры, из которых можно взять cookies, вместе с их профилями.
// Порядок браузеров фиксирован; профили — по алфавиту.
func Detect() []Browser {
	var out []Browser
	for _, d := range detectors() {
		profiles := scanProfiles(d.roots(), d.markers, d.trim)
		if len(profiles) == 0 {
			continue
		}
		out = append(out, Browser{Name: d.name, Label: d.label, Profiles: profiles})
	}
	return out
}

// detectors возвращает список обнаруживаемых браузеров (порядок — как в UI).
func detectors() []detector {
	list := []detector{
		{name: "firefox", label: "Firefox", roots: firefoxRoots, markers: firefoxMarkers, trim: true},
		{name: "chrome", label: "Google Chrome", roots: func() []string { return chromiumRoots("chrome") }, markers: chromiumMarkers},
		{name: "chromium", label: "Chromium", roots: func() []string { return chromiumRoots("chromium") }, markers: chromiumMarkers},
		{name: "brave", label: "Brave", roots: func() []string { return chromiumRoots("brave") }, markers: chromiumMarkers},
		{name: "edge", label: "Microsoft Edge", roots: func() []string { return chromiumRoots("edge") }, markers: chromiumMarkers},
		{name: "vivaldi", label: "Vivaldi", roots: func() []string { return chromiumRoots("vivaldi") }, markers: chromiumMarkers},
		{name: "opera", label: "Opera", roots: func() []string { return chromiumRoots("opera") }, markers: chromiumMarkers},
		{name: "whale", label: "Naver Whale", roots: func() []string { return chromiumRoots("whale") }, markers: chromiumMarkers},
	}
	if runtime.GOOS == "darwin" {
		list = append(list, detector{name: "safari", label: "Safari", roots: safariRoots, markers: []string{"Cookies.binarycookies"}})
	}
	return list
}

// scanProfiles ищет в подкаталогах roots профили, содержащие любой из marker-файлов.
func scanProfiles(roots []string, markers []string, trim bool) []Profile {
	seen := make(map[string]bool)
	var out []Profile
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			if seen[name] {
				continue
			}
			if !hasAnyMarker(filepath.Join(root, name), markers) {
				continue
			}
			seen[name] = true
			out = append(out, Profile{Name: name, Label: profileLabel(name, trim)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// hasAnyMarker сообщает, есть ли в каталоге dir хотя бы один из файлов rels.
func hasAnyMarker(dir string, rels []string) bool {
	for _, rel := range rels {
		if fi, err := os.Stat(filepath.Join(dir, rel)); err == nil && !fi.IsDir() {
			return true
		}
	}
	return false
}

// profileLabel укорачивает метку профиля Firefox: `xxxx.default-release` → `default-release`.
func profileLabel(name string, trim bool) string {
	if trim {
		if i := strings.IndexByte(name, '.'); i >= 0 && i+1 < len(name) {
			return name[i+1:]
		}
	}
	return name
}

// home возвращает домашний каталог пользователя.
func home() string {
	h, _ := os.UserHomeDir()
	return h
}

// configHome возвращает XDG-каталог конфигурации (по умолчанию ~/.config).
func configHome() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	return filepath.Join(home(), ".config")
}

// firefoxRoots возвращает каталоги, в подкаталогах которых лежат профили Firefox.
func firefoxRoots() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			filepath.Join(os.Getenv("APPDATA"), "Mozilla", "Firefox", "Profiles"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Packages", "Mozilla.Firefox_n80bbvh6b1yt2",
				"LocalCache", "Roaming", "Mozilla", "Firefox", "Profiles"),
		}
	case "darwin":
		return []string{filepath.Join(home(), "Library", "Application Support", "Firefox", "Profiles")}
	default:
		return []string{
			filepath.Join(configHome(), "mozilla", "firefox"),                                           // FF147+ (XDG)
			filepath.Join(home(), ".mozilla", "firefox"),                                                // FF<=146
			filepath.Join(home(), ".var", "app", "org.mozilla.firefox", "config", "mozilla", "firefox"), // flatpak
			filepath.Join(home(), ".var", "app", "org.mozilla.firefox", ".mozilla", "firefox"),
			filepath.Join(home(), "snap", "firefox", "common", ".mozilla", "firefox"), // snap
		}
	}
}

// chromiumRoots возвращает каталог данных указанного Chromium-браузера.
func chromiumRoots(name string) []string {
	switch runtime.GOOS {
	case "windows":
		local := os.Getenv("LOCALAPPDATA")
		roaming := os.Getenv("APPDATA")
		dirs := map[string]string{
			"brave":    filepath.Join(local, "BraveSoftware", "Brave-Browser", "User Data"),
			"chrome":   filepath.Join(local, "Google", "Chrome", "User Data"),
			"chromium": filepath.Join(local, "Chromium", "User Data"),
			"edge":     filepath.Join(local, "Microsoft", "Edge", "User Data"),
			"opera":    filepath.Join(roaming, "Opera Software", "Opera Stable"),
			"vivaldi":  filepath.Join(local, "Vivaldi", "User Data"),
			"whale":    filepath.Join(local, "Naver", "Naver Whale", "User Data"),
		}
		return single(dirs[name])
	case "darwin":
		app := filepath.Join(home(), "Library", "Application Support")
		dirs := map[string]string{
			"brave":    filepath.Join(app, "BraveSoftware", "Brave-Browser"),
			"chrome":   filepath.Join(app, "Google", "Chrome"),
			"chromium": filepath.Join(app, "Chromium"),
			"edge":     filepath.Join(app, "Microsoft Edge"),
			"opera":    filepath.Join(app, "com.operasoftware.Opera"),
			"vivaldi":  filepath.Join(app, "Vivaldi"),
			"whale":    filepath.Join(app, "Naver", "Whale"),
		}
		return single(dirs[name])
	default:
		c := configHome()
		dirs := map[string]string{
			"brave":    filepath.Join(c, "BraveSoftware", "Brave-Browser"),
			"chrome":   filepath.Join(c, "google-chrome"),
			"chromium": filepath.Join(c, "chromium"),
			"edge":     filepath.Join(c, "microsoft-edge"),
			"opera":    filepath.Join(c, "opera"),
			"vivaldi":  filepath.Join(c, "vivaldi"),
			"whale":    filepath.Join(c, "naver-whale"),
		}
		return single(dirs[name])
	}
}

// safariRoots возвращает каталог с базой cookies Safari (только macOS).
func safariRoots() []string {
	return []string{filepath.Join(home(), "Library", "Cookies")}
}

// single возвращает пустой срез для пустой строки (чтобы не читать каталог "").
func single(dir string) []string {
	if dir == "" {
		return nil
	}
	return []string{dir}
}
