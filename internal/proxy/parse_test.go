package proxy

import "testing"

func TestParseListSample(t *testing.T) {
	text := `
 socks5://5.75.133.113:10808
 socks5://147.93.172.241:5555
 socks5://101.32.60.93:1080
 socks5://148.251.68.232:8888
 socks5://45.74.31.42:12690
 socks5://45.74.31.42:13261
`
	res := ParseList(text)
	if len(res.Proxies) != 6 {
		t.Fatalf("разобрано %d прокси, want 6 (пропущено %d)", len(res.Proxies), res.Skipped)
	}
	if res.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", res.Skipped)
	}

	first := res.Proxies[0]
	if first.Scheme != SchemeSOCKS5 || first.Host != "5.75.133.113" || first.Port != 10808 {
		t.Errorf("первый прокси = %+v", first)
	}
	if !first.Enabled {
		t.Error("импортированный прокси должен быть включён")
	}

	ids := make(map[string]bool)
	for _, p := range res.Proxies {
		if p.ID == "" {
			t.Errorf("пустой ID у %+v", p)
		}
		if ids[p.ID] {
			t.Errorf("дубликат ID: %q", p.ID)
		}
		ids[p.ID] = true
	}
}

func TestParseListDeduplicates(t *testing.T) {
	res := ParseList("socks5://1.2.3.4:1080\nsocks5://1.2.3.4:1080\n")
	if len(res.Proxies) != 1 {
		t.Fatalf("разобрано %d, want 1 (дубликат схлопнут)", len(res.Proxies))
	}
}

func TestParseListSkipsInvalid(t *testing.T) {
	res := ParseList("# комментарий\n\nftp://1.2.3.4:21\nsocks5://1.2.3.4:1080\n:bad\n")
	if len(res.Proxies) != 1 {
		t.Fatalf("разобрано %d, want 1", len(res.Proxies))
	}
	if res.Skipped != 2 {
		t.Errorf("Skipped = %d, want 2 (неверная схема и битая строка)", res.Skipped)
	}
}

func TestParseLineFormats(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		scheme   string
		host     string
		port     int
		user     string
		password string
	}{
		{"со схемой", "socks5://1.2.3.4:1080", SchemeSOCKS5, "1.2.3.4", 1080, "", ""},
		{"socks5h", "socks5h://1.2.3.4:1080", SchemeSOCKS5H, "1.2.3.4", 1080, "", ""},
		{"socks4a", "socks4a://1.2.3.4:1080", SchemeSOCKS4A, "1.2.3.4", 1080, "", ""},
		{"без схемы", "1.2.3.4:8080", SchemeHTTP, "1.2.3.4", 8080, "", ""},
		{"host:port:user:pass", "1.2.3.4:8080:bob:secret", SchemeHTTP, "1.2.3.4", 8080, "bob", "secret"},
		{"user:pass@host:port", "bob:secret@1.2.3.4:8080", SchemeHTTP, "1.2.3.4", 8080, "bob", "secret"},
		{"http", "http://proxy.local:3128", SchemeHTTP, "proxy.local", 3128, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ParseLine(tt.line)
			if err != nil {
				t.Fatalf("ParseLine(%q): %v", tt.line, err)
			}
			if p.Scheme != tt.scheme || p.Host != tt.host || p.Port != tt.port ||
				p.Username != tt.user || p.Password != tt.password {
				t.Errorf("получено %+v", p)
			}
		})
	}
}

func TestParseLineErrors(t *testing.T) {
	for _, line := range []string{"", "ftp://1.2.3.4:21", "socks5://1.2.3.4", "socks5://1.2.3.4:abc"} {
		if _, err := ParseLine(line); err == nil {
			t.Errorf("ParseLine(%q): ожидалась ошибка", line)
		}
	}
}
