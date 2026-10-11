package proxy

import (
	"fmt"
	"net/url"
	"strings"
)

// ParseResult — результат разбора списка прокси: разобранные записи и число пропущенных строк.
type ParseResult struct {
	Proxies []Proxy
	Skipped int
}

// ParseList разбирает построчный список прокси. Поддерживаемые форматы строк:
//
//	scheme://[user:pass@]host:port
//	[user:pass@]host:port      (схема по умолчанию — http)
//	host:port:user:pass        (схема по умолчанию — http)
//
// Пустые строки и строки, начинающиеся с "#", игнорируются. Дубликаты (по конечной точке)
// схлопываются. Некорректные строки пропускаются и учитываются в Skipped.
func ParseList(text string) ParseResult {
	result := ParseResult{Proxies: []Proxy{}}
	seen := make(map[string]bool)

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		p, err := ParseLine(line)
		if err != nil {
			result.Skipped++
			continue
		}
		if seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		result.Proxies = append(result.Proxies, p)
	}
	return result
}

// ParseLine разбирает одну строку в прокси.
func ParseLine(line string) (Proxy, error) {
	raw := strings.TrimSpace(line)
	if raw == "" {
		return Proxy{}, fmt.Errorf("пустая строка")
	}

	// Формат host:port:user:pass (без схемы и без "@").
	if !strings.Contains(raw, "://") && !strings.Contains(raw, "@") {
		if parts := strings.Split(raw, ":"); len(parts) == 4 {
			return newProxy(defaultScheme, parts[2], parts[3], parts[0], parts[1])
		}
	}

	// Остальные форматы разбираем через net/url, дописав схему, если её нет.
	normalized := raw
	if !strings.Contains(normalized, "://") {
		normalized = defaultScheme + "://" + normalized
	}

	u, err := url.Parse(normalized)
	if err != nil {
		return Proxy{}, fmt.Errorf("некорректный прокси %q: %w", raw, err)
	}

	username, password := "", ""
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}
	return newProxy(u.Scheme, username, password, u.Hostname(), u.Port())
}
