package deps

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDenoAsset(t *testing.T) {
	cases := []struct {
		goos    string
		goarch  string
		want    string
		wantErr bool
	}{
		{"linux", "amd64", "deno-x86_64-unknown-linux-gnu.zip", false},
		{"linux", "arm64", "deno-aarch64-unknown-linux-gnu.zip", false},
		{"darwin", "amd64", "deno-x86_64-apple-darwin.zip", false},
		{"darwin", "arm64", "deno-aarch64-apple-darwin.zip", false},
		{"windows", "amd64", "deno-x86_64-pc-windows-msvc.zip", false},
		{"windows", "arm64", "deno-aarch64-pc-windows-msvc.zip", false},
		{"freebsd", "amd64", "", true},
		{"linux", "riscv64", "", true},
	}

	for _, c := range cases {
		got, err := denoAsset(c.goos, c.goarch)
		if c.wantErr {
			if err == nil {
				t.Errorf("denoAsset(%q, %q): ожидалась ошибка", c.goos, c.goarch)
			}
			continue
		}
		if err != nil {
			t.Errorf("denoAsset(%q, %q): %v", c.goos, c.goarch, err)
			continue
		}
		if got != c.want {
			t.Errorf("denoAsset(%q, %q) = %q, want %q", c.goos, c.goarch, got, c.want)
		}
	}
}

func TestDenoVersion(t *testing.T) {
	raw := "deno 2.9.7 (stable, release, x86_64-unknown-linux-gnu)\nv8 13.0.0\ntypescript 5.9.2\n"
	if got := denoVersion(raw); got != "2.9.7" {
		t.Errorf("denoVersion = %q, want 2.9.7", got)
	}
}

func TestParseSHA256(t *testing.T) {
	hash := strings.Repeat("a", 64)

	if got, err := parseSHA256(hash + "\n"); err != nil || got != hash {
		t.Errorf("bare: got %q, err %v", got, err)
	}
	if got, err := parseSHA256(hash + "  deno.zip\n"); err != nil || got != hash {
		t.Errorf("GNU: got %q, err %v", got, err)
	}
	if _, err := parseSHA256("не хэш\n"); err == nil {
		t.Error("ожидалась ошибка для текста без SHA-256")
	}
}

// makeZip собирает zip-архив с единственным файлом — для проверки распаковки.
func makeZip(t *testing.T, name string, content []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func TestDenoInstallAndRemove(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("тест проверяет бит исполнения на unix")
	}
	t.Setenv("PATH", "") // системный deno не должен вмешиваться

	assetFile, err := denoAsset(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("платформа не поддерживается: %v", err)
	}
	sumsFile := assetFile + ".sha256sum"

	payload := []byte("fake deno binary")
	archive := makeZip(t, "deno", payload)
	sum := sha256.Sum256(archive)
	goodSum := hex.EncodeToString(sum[:])

	mux := http.NewServeMux()
	var srv *httptest.Server

	mux.HandleFunc("/repos/denoland/deno/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		rel := release{
			TagName: "v2.9.7",
			Assets: []asset{
				{Name: assetFile, URL: srv.URL + "/" + assetFile},
				{Name: sumsFile, URL: srv.URL + "/sums"},
			},
		}
		_ = json.NewEncoder(w).Encode(rel)
	})
	mux.HandleFunc("/sums", func(w http.ResponseWriter, _ *http.Request) {
		// per-asset файл: «хэш  имя».
		_, _ = w.Write([]byte(goodSum + "  " + assetFile + "\n"))
	})
	mux.HandleFunc("/"+assetFile, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	})

	srv = httptest.NewServer(mux)
	defer srv.Close()

	oldBase := githubAPIBase
	githubAPIBase = srv.URL
	t.Cleanup(func() { githubAPIBase = oldBase })

	dir := t.TempDir()
	m := NewManager(dir)

	if got, err := m.Inspect(NameDeno); err != nil || got.Status != StatusMissing {
		t.Fatalf("до установки: dep=%+v, err=%v", got, err)
	}

	dep, err := m.Install(context.Background(), NameDeno, "", nil)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if dep.Status != StatusManaged {
		t.Fatalf("после установки: status = %q, want %q", dep.Status, StatusManaged)
	}
	if dep.Version != "v2.9.7" {
		t.Fatalf("version = %q, want v2.9.7", dep.Version)
	}

	bin := filepath.Join(dir, denoBinary(runtime.GOOS))
	info, err := os.Stat(bin)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Fatal("файл не помечен исполняемым")
	}
	got, err := os.ReadFile(bin)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("содержимое распакованного файла = %q, want %q", got, payload)
	}
	if entry, ok := loadManifest(filepath.Join(dir, "deps.json"))[NameDeno]; !ok || entry.Version != "v2.9.7" {
		t.Fatalf("манифест: %+v, ok=%v", entry, ok)
	}

	if err := m.Remove(NameDeno); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if got, err := m.Inspect(NameDeno); err != nil || got.Status != StatusMissing {
		t.Fatalf("после удаления: dep=%+v, err=%v", got, err)
	}
}
