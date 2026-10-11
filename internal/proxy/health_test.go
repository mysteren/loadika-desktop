package proxy

import (
	"context"
	"io"
	"net"
	"testing"
	"time"
)

// startFakeSocks5 поднимает минимальный SOCKS5-сервер (без аутентификации). reject=true — отвечает
// на CONNECT отказом.
func startFakeSocks5(t *testing.T, reject bool) (Proxy, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleFakeSocks5(conn, reject)
		}
	}()

	addr := ln.Addr().(*net.TCPAddr)
	p := Proxy{Scheme: SchemeSOCKS5, Host: addr.IP.String(), Port: addr.Port}
	return p, func() { _ = ln.Close(); <-done }
}

func handleFakeSocks5(conn net.Conn, reject bool) {
	defer conn.Close()

	head := make([]byte, 2)
	if _, err := io.ReadFull(conn, head); err != nil {
		return
	}
	if _, err := io.ReadFull(conn, make([]byte, int(head[1]))); err != nil {
		return
	}
	_, _ = conn.Write([]byte{0x05, 0x00}) // выбран способ «без аутентификации»

	req := make([]byte, 4)
	if _, err := io.ReadFull(conn, req); err != nil {
		return
	}
	switch req[3] {
	case 0x01:
		_, _ = io.ReadFull(conn, make([]byte, 4))
	case 0x03:
		var l [1]byte
		if _, err := io.ReadFull(conn, l[:]); err != nil {
			return
		}
		_, _ = io.ReadFull(conn, make([]byte, int(l[0])))
	case 0x04:
		_, _ = io.ReadFull(conn, make([]byte, 16))
	}
	_, _ = io.ReadFull(conn, make([]byte, 2)) // порт

	if reject {
		_, _ = conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // connection refused
		return
	}
	_, _ = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
}

func TestCheckSOCKS5Success(t *testing.T) {
	p, stop := startFakeSocks5(t, false)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	latency, err := Check(ctx, p, "1.1.1.1:443")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if latency < 0 {
		t.Errorf("latency = %d, want >= 0", latency)
	}
}

func TestCheckSOCKS5Rejected(t *testing.T) {
	p, stop := startFakeSocks5(t, true)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := Check(ctx, p, "1.1.1.1:443"); err == nil {
		t.Error("ожидалась ошибка при отказе CONNECT")
	}
}

func TestCheckUnreachable(t *testing.T) {
	p := Proxy{Scheme: SchemeSOCKS5, Host: "127.0.0.1", Port: 1}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := Check(ctx, p, "1.1.1.1:443"); err == nil {
		t.Error("ожидалась ошибка для недоступного прокси")
	}
}

func TestCheckAll(t *testing.T) {
	good, stop := startFakeSocks5(t, false)
	defer stop()
	bad := Proxy{Scheme: SchemeSOCKS5, Host: "127.0.0.1", Port: 1}

	results := CheckAll(context.Background(), []Proxy{good, bad}, "1.1.1.1:443", time.Second)
	if len(results) != 2 {
		t.Fatalf("результатов %d, want 2", len(results))
	}
	if results[0].Err != nil {
		t.Errorf("хороший прокси: %v", results[0].Err)
	}
	if results[1].Err == nil {
		t.Error("недоступный прокси должен вернуть ошибку")
	}
}
