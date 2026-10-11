package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

// DefaultHealthTarget — хост, до которого проверяем доступ через прокси.
const DefaultHealthTarget = "1.1.1.1:443"

// DefaultHealthTimeout — таймаут одной проверки.
const DefaultHealthTimeout = 5 * time.Second

// HealthResult — результат проверки одного прокси.
type HealthResult struct {
	ID        string
	LatencyMs int64
	Err       error
}

// Check проверяет прокси: устанавливает соединение через него до target и возвращает задержку.
// Для socks5/socks5h выполняется SOCKS5-рукопожатие (CONNECT); для остальных схем — TCP-коннект к
// самому прокси (грубая проверка достижимости). Задержка — полное время проверки.
func Check(ctx context.Context, p Proxy, target string) (int64, error) {
	if target == "" {
		target = DefaultHealthTarget
	}
	start := time.Now()

	dialer := net.Dialer{Timeout: DefaultHealthTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(p.Host, strconv.Itoa(p.Port)))
	if err != nil {
		return 0, fmt.Errorf("подключение к прокси: %w", err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(DefaultHealthTimeout))
	}

	if isSOCKS5(p.Scheme) {
		if err := socks5Connect(conn, target, p.Username, p.Password); err != nil {
			return 0, err
		}
	}
	return time.Since(start).Milliseconds(), nil
}

// CheckAll проверяет несколько прокси параллельно (каждый — с таймаутом timeout) и возвращает
// результаты в порядке входного списка.
func CheckAll(ctx context.Context, proxies []Proxy, target string, timeout time.Duration) []HealthResult {
	results := make([]HealthResult, len(proxies))

	var wg sync.WaitGroup
	for i, p := range proxies {
		wg.Add(1)
		go func(i int, p Proxy) {
			defer wg.Done()
			cctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			latency, err := Check(cctx, p, target)
			results[i] = HealthResult{ID: p.ID, LatencyMs: latency, Err: err}
		}(i, p)
	}
	wg.Wait()
	return results
}

// socks5Connect выполняет SOCKS5-рукопожатие и запрос CONNECT до target (host:port).
func socks5Connect(conn net.Conn, target, username, password string) error {
	methods := []byte{0x00} // без аутентификации
	if username != "" {
		methods = []byte{0x00, 0x02} // + имя/пароль
	}
	if _, err := conn.Write(append([]byte{0x05, byte(len(methods))}, methods...)); err != nil {
		return fmt.Errorf("socks5: приветствие: %w", err)
	}

	var greeting [2]byte
	if _, err := io.ReadFull(conn, greeting[:]); err != nil {
		return fmt.Errorf("socks5: ответ на приветствие: %w", err)
	}
	if greeting[0] != 0x05 {
		return fmt.Errorf("socks5: неожиданная версия 0x%02x", greeting[0])
	}
	switch greeting[1] {
	case 0x00:
	case 0x02:
		if err := socks5Auth(conn, username, password); err != nil {
			return err
		}
	default:
		return fmt.Errorf("socks5: сервер не принял способ аутентификации 0x%02x", greeting[1])
	}

	if err := socks5Request(conn, target); err != nil {
		return err
	}
	return nil
}

// socks5Auth выполняет аутентификацию по имени/паролю (RFC 1929).
func socks5Auth(conn net.Conn, username, password string) error {
	if len(username) > 255 || len(password) > 255 {
		return errors.New("socks5: слишком длинные учётные данные")
	}
	buf := []byte{0x01, byte(len(username))}
	buf = append(buf, username...)
	buf = append(buf, byte(len(password)))
	buf = append(buf, password...)
	if _, err := conn.Write(buf); err != nil {
		return fmt.Errorf("socks5: аутентификация: %w", err)
	}

	var resp [2]byte
	if _, err := io.ReadFull(conn, resp[:]); err != nil {
		return fmt.Errorf("socks5: ответ аутентификации: %w", err)
	}
	if resp[1] != 0x00 {
		return errors.New("socks5: аутентификация отклонена")
	}
	return nil
}

// socks5Request отправляет CONNECT до target и проверяет успех.
func socks5Request(conn net.Conn, target string) error {
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		return fmt.Errorf("socks5: некорректная цель %q: %w", target, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("socks5: некорректный порт цели: %w", err)
	}

	req := []byte{0x05, 0x01, 0x00} // VER, CONNECT, RSV
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			req = append(req, 0x01)
			req = append(req, ip4...)
		} else {
			req = append(req, 0x04)
			req = append(req, ip.To16()...)
		}
	} else {
		if len(host) > 255 {
			return errors.New("socks5: слишком длинное имя хоста")
		}
		req = append(req, 0x03, byte(len(host)))
		req = append(req, host...)
	}
	req = append(req, byte(port>>8), byte(port))

	if _, err := conn.Write(req); err != nil {
		return fmt.Errorf("socks5: запрос CONNECT: %w", err)
	}

	var reply [4]byte
	if _, err := io.ReadFull(conn, reply[:]); err != nil {
		return fmt.Errorf("socks5: ответ CONNECT: %w", err)
	}
	if reply[1] != 0x00 {
		return fmt.Errorf("socks5: соединение отклонено (код 0x%02x)", reply[1])
	}
	// Дочитываем адрес, привязанный сервером (иначе остаются лишние байты в потоке).
	return discardSocks5Addr(conn, reply[3])
}

// discardSocks5Addr дочитывает адрес из ответа SOCKS5 по типу адреса (ATYP).
func discardSocks5Addr(conn net.Conn, atyp byte) error {
	switch atyp {
	case 0x01: // IPv4 + порт
		_, err := io.ReadFull(conn, make([]byte, 4+2))
		return err
	case 0x04: // IPv6 + порт
		_, err := io.ReadFull(conn, make([]byte, 16+2))
		return err
	case 0x03: // домен: длина + имя + порт
		var length [1]byte
		if _, err := io.ReadFull(conn, length[:]); err != nil {
			return err
		}
		_, err := io.ReadFull(conn, make([]byte, int(length[0])+2))
		return err
	default:
		return fmt.Errorf("socks5: неизвестный тип адреса 0x%02x", atyp)
	}
}
