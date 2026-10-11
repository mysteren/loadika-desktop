// Package proxy описывает пул прокси: модель, импорт списком и персистенцию. Доменный пакет,
// не зависит от Wails. Рантайм-состояние (health/латентность/сбои) здесь не хранится —
// им управляет пул при работе (см. след. шаги).
package proxy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// Допустимые схемы прокси.
const (
	SchemeHTTP    = "http"
	SchemeHTTPS   = "https"
	SchemeSOCKS4  = "socks4"
	SchemeSOCKS4A = "socks4a" // как socks4, но DNS разрешает прокси
	SchemeSOCKS5  = "socks5"
	SchemeSOCKS5H = "socks5h" // как socks5, но DNS разрешает прокси (remote DNS)
)

// defaultScheme — схема для записей без явной схемы (например, `host:port`).
const defaultScheme = SchemeHTTP

// allowedSchemes — поддерживаемые схемы.
var allowedSchemes = map[string]bool{
	SchemeHTTP:    true,
	SchemeHTTPS:   true,
	SchemeSOCKS4:  true,
	SchemeSOCKS4A: true,
	SchemeSOCKS5:  true,
	SchemeSOCKS5H: true,
}

// isSOCKS5 сообщает, что схема использует протокол SOCKS5 (обычный или с remote DNS).
func isSOCKS5(scheme string) bool {
	return scheme == SchemeSOCKS5 || scheme == SchemeSOCKS5H
}

// Proxy — определение прокси (персистится). Пароль не показываем в UI и не логируем.
type Proxy struct {
	ID       string `json:"id"`
	Scheme   string `json:"scheme"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Label    string `json:"label"`
	Enabled  bool   `json:"enabled"`
}

// URL собирает строку прокси для yt-dlp (`socks5h://user:pass@host:port`) с корректным
// URL-экранированием учётных данных. Для `socks5` явно выдаём `socks5h` (DNS разрешает прокси,
// remote DNS): yt-dlp и сам так делает (`yt_dlp/utils/networking.py`, `clean_proxies`), но явная
// схема не зависит от версии/бэкенда. Пароль в открытом виде — только для запуска yt-dlp, не для
// логов/UI.
func (p Proxy) URL() string {
	scheme := p.Scheme
	if scheme == SchemeSOCKS5 {
		scheme = SchemeSOCKS5H
	}
	u := url.URL{Scheme: scheme, Host: net.JoinHostPort(p.Host, strconv.Itoa(p.Port))}
	if p.Username != "" {
		u.User = url.UserPassword(p.Username, p.Password)
	}
	return u.String()
}

// Display возвращает представление прокси для UI/логов БЕЗ пароля (только схема, хост, порт и,
// если есть, имя пользователя).
func (p Proxy) Display() string {
	endpoint := net.JoinHostPort(p.Host, strconv.Itoa(p.Port))
	if p.Username != "" {
		return fmt.Sprintf("%s://%s@%s", p.Scheme, p.Username, endpoint)
	}
	return fmt.Sprintf("%s://%s", p.Scheme, endpoint)
}

// DialContext подключается к addr через прокси. Для SOCKS5 выполняется рукопожатие и CONNECT
// (удалённое разрешение DNS). Для остальных схем прокси поддержка не реализована — ошибка.
// Возвращённое соединение уже туннелировано до addr и пригодно для http.Transport.DialContext.
func (p Proxy) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if !isSOCKS5(p.Scheme) {
		return nil, fmt.Errorf("прокси-схема %q не поддерживает туннелирование", p.Scheme)
	}
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(p.Host, strconv.Itoa(p.Port)))
	if err != nil {
		return nil, err
	}
	if err := socks5Connect(conn, addr, p.Username, p.Password); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

// newProxy создаёт прокси с проверкой полей и вычислением стабильного идентификатора.
func newProxy(scheme, username, password, host, portStr string) (Proxy, error) {
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	if !allowedSchemes[scheme] {
		return Proxy{}, fmt.Errorf("неизвестная схема прокси: %q", scheme)
	}

	host = strings.TrimSpace(host)
	if host == "" {
		return Proxy{}, fmt.Errorf("не задан хост прокси")
	}

	port, err := strconv.Atoi(strings.TrimSpace(portStr))
	if err != nil || port < 1 || port > 65535 {
		return Proxy{}, fmt.Errorf("некорректный порт прокси: %q", portStr)
	}

	p := Proxy{
		Scheme:   scheme,
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		Enabled:  true,
	}
	p.ID = idFor(p)
	return p, nil
}

// idFor вычисляет стабильный идентификатор по конечной точке (схема+пользователь+хост+порт).
// Благодаря этому повторный импорт того же прокси не создаёт дубликат.
func idFor(p Proxy) string {
	key := strings.ToLower(p.Scheme) + "://" + p.Username + "@" + net.JoinHostPort(p.Host, strconv.Itoa(p.Port))
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:12]
}
