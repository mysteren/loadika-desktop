package deps

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// downloadFile скачивает url в dst, вычисляя SHA-256 по ходу и сообщая о прогрессе.
// Проверка хэша выполняется после записи всего файла.
func downloadFile(ctx context.Context, url, dst, wantSHA string, onProgress func(received, total int64)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("загрузка %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("загрузка %s: статус %d", url, resp.StatusCode)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("создание %q: %w", dst, err)
	}

	hasher := sha256.New()
	writer := io.MultiWriter(out, hasher)
	total := resp.ContentLength
	var received int64
	buf := make([]byte, 64*1024)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := writer.Write(buf[:n]); writeErr != nil {
				_ = out.Close()
				return fmt.Errorf("запись %q: %w", dst, writeErr)
			}
			received += int64(n)
			if onProgress != nil {
				onProgress(received, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = out.Close()
			return fmt.Errorf("чтение ответа: %w", readErr)
		}
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("закрытие %q: %w", dst, err)
	}

	got := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(got, wantSHA) {
		return fmt.Errorf("контрольная сумма не совпала: получено %s, ожидалось %s", got, wantSHA)
	}
	return nil
}

// replaceFile атомарно заменяет dst файлом src.
func replaceFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// Windows не перезаписывает существующий файл — удаляем и пробуем снова.
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(src, dst)
}
