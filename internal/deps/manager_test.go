package deps

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// inspectName возвращает состояние зависимости, провалив тест при ошибке имени.
func inspectName(t *testing.T, m *Manager, name string) Dependency {
	t.Helper()
	dep, err := m.Inspect(name)
	if err != nil {
		t.Fatalf("Inspect(%q): %v", name, err)
	}
	return dep
}

func TestInspectMissing(t *testing.T) {
	t.Setenv("PATH", "") // гарантируем, что yt-dlp не найдётся в системе

	m := NewManager(t.TempDir())
	if got := inspectName(t, m, NameYtDlp).Status; got != StatusMissing {
		t.Fatalf("status = %q, want %q", got, StatusMissing)
	}
}

func TestInspectSystem(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("тест использует shell-скрипт")
	}

	dir := t.TempDir()
	bin := filepath.Join(dir, "yt-dlp")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho 1.2.3\n"), 0o755); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv("PATH", dir)

	dep := inspectName(t, NewManager(t.TempDir()), NameYtDlp)
	if dep.Status != StatusSystem {
		t.Fatalf("status = %q, want %q", dep.Status, StatusSystem)
	}
	if dep.Version != "1.2.3" {
		t.Fatalf("version = %q, want 1.2.3", dep.Version)
	}
}

func TestInstallAndRemoveCycle(t *testing.T) {
	t.Setenv("PATH", "") // системная копия не должна вмешиваться

	assetFile, err := ytdlpAsset(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("платформа не поддерживается: %v", err)
	}

	payload := []byte("fake yt-dlp binary")
	sum := sha256.Sum256(payload)
	goodSum := hex.EncodeToString(sum[:])

	mux := http.NewServeMux()
	var srv *httptest.Server

	mux.HandleFunc("/repos/yt-dlp/yt-dlp/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		rel := release{
			TagName: "2026.08.19",
			Assets: []asset{
				{Name: assetFile, URL: srv.URL + "/" + assetFile},
				{Name: checksumsAsset, URL: srv.URL + "/sums"},
			},
		}
		_ = json.NewEncoder(w).Encode(rel)
	})
	mux.HandleFunc("/sums", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(goodSum + "  " + assetFile + "\n"))
	})
	mux.HandleFunc("/"+assetFile, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(payload)
	})

	srv = httptest.NewServer(mux)
	defer srv.Close()

	oldBase := githubAPIBase
	githubAPIBase = srv.URL
	t.Cleanup(func() { githubAPIBase = oldBase })

	dir := t.TempDir()
	m := NewManager(dir)

	if got := inspectName(t, m, NameYtDlp).Status; got != StatusMissing {
		t.Fatalf("до установки: status = %q, want %q", got, StatusMissing)
	}

	dep, err := m.Install(context.Background(), NameYtDlp, "stable", nil)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if dep.Status != StatusManaged {
		t.Fatalf("после установки: status = %q, want %q", dep.Status, StatusManaged)
	}
	if dep.Version != "2026.08.19" {
		t.Fatalf("version = %q, want 2026.08.19", dep.Version)
	}

	info, err := os.Stat(filepath.Join(dir, ytdlpBinary(runtime.GOOS)))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o100 == 0 {
		t.Fatal("файл не помечен исполняемым")
	}

	if entry, ok := loadManifest(filepath.Join(dir, "deps.json"))[NameYtDlp]; !ok || entry.Version != "2026.08.19" {
		t.Fatalf("манифест: %+v, ok=%v", entry, ok)
	}

	if err := m.Remove(NameYtDlp); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if got := inspectName(t, m, NameYtDlp).Status; got != StatusMissing {
		t.Fatalf("после удаления: status = %q, want %q", got, StatusMissing)
	}
}

func TestRemoveAllRemovesManagedAndKeepsSystem(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("тест использует shell-скрипт")
	}

	// Системная копия в PATH: RemoveAll не должен её удалять.
	sysDir := t.TempDir()
	sysBin := filepath.Join(sysDir, ytdlpBinary(runtime.GOOS))
	if err := os.WriteFile(sysBin, []byte("#!/bin/sh\necho 1.0.0\n"), 0o755); err != nil {
		t.Fatalf("write system: %v", err)
	}
	t.Setenv("PATH", sysDir)

	// Управляемая копия + запись манифеста (устанавливаем вручную, без сети).
	dir := t.TempDir()
	managedBin := filepath.Join(dir, ytdlpBinary(runtime.GOOS))
	if err := os.WriteFile(managedBin, []byte("fake yt-dlp"), 0o755); err != nil {
		t.Fatalf("write managed: %v", err)
	}
	if err := saveManifest(filepath.Join(dir, "deps.json"), manifest{
		NameYtDlp: manifestEntry{Version: "1.2.3", Channel: "stable"},
	}); err != nil {
		t.Fatalf("saveManifest: %v", err)
	}

	m := NewManager(dir)
	if got := inspectName(t, m, NameYtDlp).Status; got != StatusManaged {
		t.Fatalf("до удаления: status = %q, want %q", got, StatusManaged)
	}

	if err := m.RemoveAll(); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}

	if _, err := os.Stat(managedBin); !os.IsNotExist(err) {
		t.Fatalf("управляемый бинарь не удалён: err=%v", err)
	}
	if _, ok := loadManifest(filepath.Join(dir, "deps.json"))[NameYtDlp]; ok {
		t.Fatal("запись манифеста не удалена")
	}
	if _, err := os.Stat(sysBin); err != nil {
		t.Fatalf("системная копия удалена: %v", err)
	}
	if got := inspectName(t, m, NameYtDlp).Status; got != StatusSystem {
		t.Fatalf("после удаления: status = %q, want %q", got, StatusSystem)
	}
}

func TestResolve(t *testing.T) {
	t.Setenv("PATH", "") // системная копия не должна вмешиваться

	dir := t.TempDir()
	managedBin := filepath.Join(dir, ytdlpBinary(runtime.GOOS))
	if err := os.WriteFile(managedBin, []byte("fake yt-dlp"), 0o755); err != nil {
		t.Fatalf("write managed: %v", err)
	}
	if err := saveManifest(filepath.Join(dir, "deps.json"), manifest{
		NameYtDlp: manifestEntry{Version: "1.2.3"},
	}); err != nil {
		t.Fatalf("saveManifest: %v", err)
	}

	m := NewManager(dir)

	// Управляемая копия разрешается, версия при этом не запускается.
	if path, ok := m.Resolve(NameYtDlp); !ok || path != managedBin {
		t.Fatalf("Resolve(yt-dlp) = %q, %v; want %q, true", path, ok, managedBin)
	}
	if _, ok := m.Resolve("нет-такой"); ok {
		t.Error("неизвестное имя не должно разрешаться")
	}
	if _, ok := m.Resolve(NameFfmpeg); ok {
		t.Error("отсутствующий ffmpeg не должен разрешаться")
	}
}

func TestUpdateAvailable(t *testing.T) {
	// deno: тег «v2.9.7» и системная версия «2.9.7» — одинаковы.
	sysDeno := Dependency{Status: StatusSystem, Version: "2.9.7"}
	if updateAvailable(denoSpec, sysDeno, "v2.9.7") {
		t.Error("deno: 2.9.7 и v2.9.7 должны считаться одинаковыми")
	}
	if !updateAvailable(denoSpec, sysDeno, "v2.10.0") {
		t.Error("deno: v2.10.0 — обновление")
	}

	// yt-dlp: сравнение как есть, работает и для системной копии.
	sysYt := Dependency{Status: StatusSystem, Version: "2026.08.19"}
	if updateAvailable(ytdlpSpec, sysYt, "2026.08.19") {
		t.Error("yt-dlp: одинаковые версии")
	}
	if !updateAvailable(ytdlpSpec, sysYt, "2026.09.01") {
		t.Error("yt-dlp: 2026.09.01 — обновление")
	}

	// Отсутствующая зависимость — это установка, а не обновление.
	if updateAvailable(ytdlpSpec, Dependency{Status: StatusMissing}, "2026.09.01") {
		t.Error("missing: обновления нет")
	}
}
