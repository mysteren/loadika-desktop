package deps

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAssetURL(t *testing.T) {
	rel := release{
		TagName: "2026.08.19",
		Assets: []asset{
			{Name: "yt-dlp.exe", URL: "https://example/yt-dlp.exe"},
			{Name: "SHA2-256SUMS", URL: "https://example/sums"},
		},
	}

	if url, ok := rel.assetURL("yt-dlp.exe"); !ok || url != "https://example/yt-dlp.exe" {
		t.Fatalf("assetURL = %q, %v", url, ok)
	}
	if _, ok := rel.assetURL("missing"); ok {
		t.Fatal("assetURL: ожидалось отсутствие файла")
	}
}

func TestDownloadFileVerifiesChecksum(t *testing.T) {
	payload := []byte("hello yt-dlp binary")
	sum := sha256.Sum256(payload)
	goodSum := hex.EncodeToString(sum[:])

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	dir := t.TempDir()

	// Успешное скачивание и проверка.
	dst := filepath.Join(dir, "ok.bin")
	var lastReceived int64
	err := downloadFile(context.Background(), srv.URL, dst, goodSum, func(received, _ int64) {
		lastReceived = received
	})
	if err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
	if lastReceived != int64(len(payload)) {
		t.Fatalf("прогресс received = %d, want %d", lastReceived, len(payload))
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatal("содержимое файла не совпало")
	}

	// Неверная контрольная сумма.
	badDst := filepath.Join(dir, "bad.bin")
	if err := downloadFile(context.Background(), srv.URL, badDst, "deadbeef", nil); err == nil {
		t.Fatal("downloadFile: ожидалась ошибка контрольной суммы")
	}
}

func TestReplaceFileOverwrites(t *testing.T) {
	dir := t.TempDir()

	dst := filepath.Join(dir, "bin")
	if err := os.WriteFile(dst, []byte("old"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	src := filepath.Join(dir, "bin.tmp")
	if err := os.WriteFile(src, []byte("new"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := replaceFile(src, dst); err != nil {
		t.Fatalf("replaceFile: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "new" {
		t.Fatalf("dst = %q, want new", string(got))
	}
}
