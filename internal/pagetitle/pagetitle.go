// Package pagetitle достаёт «человеческий» заголовок веб-страницы (og:title / twitter:title /
// <h1> / <title>) — запасной вариант, когда yt-dlp не дал внятного названия. Не зависит от Wails.
package pagetitle

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// maxBytes — сколько HTML читаем (хватает для <head> и первого <h1>, защита от гигантских страниц).
const maxBytes = 512 * 1024

// userAgent — часть сайтов отдаёт страницу только «браузеру».
const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"

var (
	metaTagRe  = regexp.MustCompile(`(?is)<meta\s[^>]*>`)
	attrRe     = regexp.MustCompile(`(?is)([a-zA-Z:_-]+)\s*=\s*("([^"]*)"|'([^']*)'|([^\s"'>]+))`)
	titleTagRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	h1TagRe    = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
	tagRe      = regexp.MustCompile(`(?s)<[^>]*>`)
	spaceRe    = regexp.MustCompile(`\s+`)
)

// Fetch скачивает страницу и возвращает её заголовок (пустую строку, если не удалось).
func Fetch(ctx context.Context, client *http.Client, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("статус %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return "", err
	}
	return Title(string(body)), nil
}

// Title извлекает заголовок из HTML. Порядок предпочтений: og:title, twitter:title, <h1>, <title>.
// Функция чистая. Возвращает пустую строку, если ничего подходящего нет.
func Title(page string) string {
	for _, t := range metaTitles(page) {
		if t != "" {
			return t
		}
	}
	if m := h1TagRe.FindStringSubmatch(page); m != nil {
		if t := clean(m[1]); t != "" {
			return t
		}
	}
	if m := titleTagRe.FindStringSubmatch(page); m != nil {
		if t := clean(m[1]); t != "" {
			return t
		}
	}
	return ""
}

// metaTitles возвращает заголовки из <meta> по приоритету: сначала og:title, затем twitter:title.
func metaTitles(page string) []string {
	var og, twitter string
	for _, tag := range metaTagRe.FindAllString(page, -1) {
		attrs := parseAttrs(tag)
		key := strings.ToLower(attrs["property"])
		if key == "" {
			key = strings.ToLower(attrs["name"])
		}
		content := clean(attrs["content"])
		if content == "" {
			continue
		}
		switch key {
		case "og:title":
			if og == "" {
				og = content
			}
		case "twitter:title":
			if twitter == "" {
				twitter = content
			}
		}
	}
	return []string{og, twitter}
}

// parseAttrs разбирает атрибуты одного тега в карту (имена — в нижнем регистре).
func parseAttrs(tag string) map[string]string {
	attrs := make(map[string]string)
	for _, m := range attrRe.FindAllStringSubmatch(tag, -1) {
		name := strings.ToLower(m[1])
		value := m[3]
		if value == "" {
			value = m[4]
		}
		if value == "" {
			value = m[5]
		}
		if _, exists := attrs[name]; !exists {
			attrs[name] = value
		}
	}
	return attrs
}

// clean убирает вложенные теги, раскодирует HTML-сущности и схлопывает пробелы.
func clean(s string) string {
	s = tagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = spaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
