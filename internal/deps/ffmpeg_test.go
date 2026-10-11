package deps

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
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

func TestFfmpegStaticPlatform(t *testing.T) {
	cases := []struct {
		goos    string
		goarch  string
		want    string
		wantErr bool
	}{
		{"darwin", "amd64", "darwin-x64", false},
		{"darwin", "arm64", "darwin-arm64", false},
		{"linux", "amd64", "linux-x64", false},
		{"linux", "arm64", "linux-arm64", false},
		{"linux", "arm", "linux-arm", false},
		{"windows", "amd64", "win32-x64", false},
		{"freebsd", "amd64", "", true},
	}

	for _, c := range cases {
		got, err := ffmpegStaticPlatform(c.goos, c.goarch)
		if c.wantErr {
			if err == nil {
				t.Errorf("ffmpegStaticPlatform(%q, %q): ожидалась ошибка", c.goos, c.goarch)
			}
			continue
		}
		if err != nil {
			t.Errorf("ffmpegStaticPlatform(%q, %q): %v", c.goos, c.goarch, err)
			continue
		}
		if got != c.want {
			t.Errorf("ffmpegStaticPlatform(%q, %q) = %q, want %q", c.goos, c.goarch, got, c.want)
		}
	}
}

func TestFfmpegVersion(t *testing.T) {
	raw := "ffmpeg version 6.1.1-full_build Copyright (c) 2000-2023\nbuilt with gcc\n"
	if got := ffmpegVersion(raw); got != "6.1.1-full_build" {
		t.Errorf("ffmpegVersion = %q, want 6.1.1-full_build", got)
	}
}

func TestBinaryVersionArgs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("тест использует shell-скрипт")
	}

	dir := t.TempDir()
	bin := filepath.Join(dir, "ffmpeg")
	// Скрипт печатает версию только для одиночного `-version` (как настоящий ffmpeg).
	script := "#!/bin/sh\nif [ \"$1\" = \"-version\" ]; then echo 'ffmpeg version 6.1.1 x'; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write: %v", err)
	}

	if got := binaryVersion(ffmpegSpec, bin); got != "6.1.1" {
		t.Fatalf("binaryVersion(ffmpeg) = %q, want 6.1.1", got)
	}
}

func TestFfmpegRepo(t *testing.T) {
	if repo, _ := ffmpegRepo("windows", "arm64", ""); repo != "yt-dlp/FFmpeg-Builds" {
		t.Errorf("windows/arm64 repo = %q, want yt-dlp/FFmpeg-Builds", repo)
	}
	if repo, _ := ffmpegRepo("linux", "amd64", ""); repo != "eugeneware/ffmpeg-static" {
		t.Errorf("linux/amd64 repo = %q, want eugeneware/ffmpeg-static", repo)
	}
}

func TestFfmpegComponents(t *testing.T) {
	// Обычные платформы: два gzip-файла (ffmpeg + ffprobe).
	comps, err := ffmpegComponents("linux", "amd64")
	if err != nil {
		t.Fatalf("ffmpegComponents(linux, amd64): %v", err)
	}
	if len(comps) != 2 {
		t.Fatalf("компонентов = %d, want 2", len(comps))
	}
	if comps[0].asset != "ffmpeg-linux-x64.gz" || comps[0].archive != archiveGzip {
		t.Errorf("компонент ffmpeg = %+v", comps[0])
	}
	if comps[1].asset != "ffprobe-linux-x64.gz" {
		t.Errorf("компонент ffprobe = %+v", comps[1])
	}

	// Windows ARM64: единый zip из FFmpeg-Builds с двумя бинарниками.
	comps, err = ffmpegComponents("windows", "arm64")
	if err != nil {
		t.Fatalf("ffmpegComponents(windows, arm64): %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("компонентов = %d, want 1", len(comps))
	}
	if comps[0].asset != "ffmpeg-master-latest-winarm64-gpl.zip" || comps[0].checksums != "checksums.sha256" {
		t.Errorf("компонент win-arm64 = %+v", comps[0])
	}
	if len(comps[0].outputs) != 2 {
		t.Errorf("outputs = %d, want 2", len(comps[0].outputs))
	}
}

// gzipBytes сжимает данные в gzip.
func gzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

// makeZipFiles собирает zip-архив с несколькими файлами (пути сохраняются).
func makeZipFiles(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip create %q: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("zip write %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func TestFfmpegInstallStatic(t *testing.T) {
	if runtime.GOOS == "windows" && runtime.GOARCH == "arm64" {
		t.Skip("для windows/arm64 используется другой источник")
	}
	if runtime.GOOS == "windows" {
		t.Skip("тест проверяет бит исполнения на unix")
	}
	t.Setenv("PATH", "") // системный ffmpeg не должен вмешиваться

	platform, err := ffmpegStaticPlatform(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("платформа не поддерживается: %v", err)
	}
	ffAsset := "ffmpeg-" + platform + ".gz"
	fpAsset := "ffprobe-" + platform + ".gz"

	payloadFF := []byte("fake ffmpeg")
	payloadFP := []byte("fake ffprobe")
	gzFF := gzipBytes(t, payloadFF)
	gzFP := gzipBytes(t, payloadFP)
	sumFF := sha256.Sum256(gzFF)
	sumFP := sha256.Sum256(gzFP)

	mux := http.NewServeMux()
	var srv *httptest.Server

	mux.HandleFunc("/repos/eugeneware/ffmpeg-static/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		rel := release{
			TagName: "b6.1.1",
			Assets: []asset{
				{Name: ffAsset, URL: srv.URL + "/ff", Digest: "sha256:" + hex.EncodeToString(sumFF[:])},
				{Name: fpAsset, URL: srv.URL + "/fp", Digest: "sha256:" + hex.EncodeToString(sumFP[:])},
			},
		}
		_ = json.NewEncoder(w).Encode(rel)
	})
	mux.HandleFunc("/ff", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(gzFF) })
	mux.HandleFunc("/fp", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(gzFP) })

	srv = httptest.NewServer(mux)
	defer srv.Close()

	oldBase := githubAPIBase
	githubAPIBase = srv.URL
	t.Cleanup(func() { githubAPIBase = oldBase })

	dir := t.TempDir()
	m := NewManager(dir)

	if got := inspectName(t, m, NameFfmpeg).Status; got != StatusMissing {
		t.Fatalf("до установки: status = %q, want %q", got, StatusMissing)
	}

	dep, err := m.Install(context.Background(), NameFfmpeg, "", nil)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if dep.Status != StatusManaged {
		t.Fatalf("после установки: status = %q, want %q", dep.Status, StatusManaged)
	}
	// Бинарник не запускается — версия берётся из тега релиза.
	if dep.Version != "b6.1.1" {
		t.Fatalf("version = %q, want b6.1.1", dep.Version)
	}

	for name, want := range map[string][]byte{
		ffmpegBinary(runtime.GOOS):  payloadFF,
		ffprobeBinary(runtime.GOOS): payloadFP,
	} {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %q: %v", name, err)
		}
		if info.Mode().Perm()&0o100 == 0 {
			t.Fatalf("%q не помечен исполняемым", name)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %q: %v", name, err)
		}
		if string(got) != string(want) {
			t.Fatalf("%q = %q, want %q", name, got, want)
		}
	}

	if err := m.Remove(NameFfmpeg); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if got := inspectName(t, m, NameFfmpeg).Status; got != StatusMissing {
		t.Fatalf("после удаления: status = %q, want %q", got, StatusMissing)
	}
}

func TestActivateComponentZip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("тест проверяет бит исполнения на unix")
	}

	dir := t.TempDir()
	archive := makeZipFiles(t, map[string]string{
		"ffmpeg-master-latest-winarm64-gpl/bin/ffmpeg.exe":  "FF",
		"ffmpeg-master-latest-winarm64-gpl/bin/ffprobe.exe": "FP",
	})
	src := filepath.Join(dir, "a.zip")
	if err := os.WriteFile(src, archive, 0o644); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	m := NewManager(dir)
	comp := component{
		archive: archiveZip,
		outputs: []output{
			{member: "ffmpeg.exe", dest: "ffmpeg.exe"},
			{member: "ffprobe.exe", dest: "ffprobe.exe"},
		},
	}
	if err := m.activateComponent(comp, src); err != nil {
		t.Fatalf("activateComponent: %v", err)
	}

	for name, want := range map[string]string{"ffmpeg.exe": "FF", "ffprobe.exe": "FP"} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %q: %v", name, err)
		}
		if string(got) != want {
			t.Fatalf("%q = %q, want %q", name, got, want)
		}
	}
}
