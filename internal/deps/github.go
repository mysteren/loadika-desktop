package deps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// userAgent обязателен для запросов к GitHub API.
const userAgent = "loadika-desktop"

// httpClient — общий клиент с таймаутом для сетевых операций.
var httpClient = &http.Client{Timeout: 60 * time.Second}

// githubAPIBase — базовый адрес GitHub API. Вынесен в переменную, чтобы тесты
// могли подменить источник на локальный сервер.
var githubAPIBase = "https://api.github.com"

// release — минимально необходимая часть ответа GitHub Releases API.
type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"` // "sha256:<hex>", GitHub может не заполнять
}

// assetURL возвращает ссылку на файл релиза по его имени.
func (r release) assetURL(name string) (string, bool) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a.URL, true
		}
	}
	return "", false
}

// assetDigest возвращает SHA-256 файла из метаданных релиза (без скачивания суммы).
// Используется, когда у источника нет отдельного файла контрольных сумм.
func (r release) assetDigest(name string) (string, error) {
	for _, a := range r.Assets {
		if a.Name != name {
			continue
		}
		if hex, ok := strings.CutPrefix(a.Digest, "sha256:"); ok && hex != "" {
			return strings.ToLower(hex), nil
		}
		return "", fmt.Errorf("у файла %q нет SHA-256 digest", name)
	}
	return "", fmt.Errorf("в релизе нет файла %q", name)
}

// fetchLatestRelease получает последний релиз указанного GitHub-репозитория.
func fetchLatestRelease(ctx context.Context, repo string) (release, error) {
	body, err := getBytes(ctx, fmt.Sprintf("%s/repos/%s/releases/latest", githubAPIBase, repo))
	if err != nil {
		return release{}, err
	}

	var rel release
	if err := json.Unmarshal(body, &rel); err != nil {
		return release{}, fmt.Errorf("разбор ответа GitHub: %w", err)
	}
	if rel.TagName == "" {
		return release{}, fmt.Errorf("релиз для %q не найден", repo)
	}
	return rel, nil
}

// getBytes выполняет GET и возвращает тело ответа целиком.
func getBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("запрос %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("запрос %s: статус %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
