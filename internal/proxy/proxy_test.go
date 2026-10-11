package proxy

import "testing"

func TestProxyURL(t *testing.T) {
	p, err := ParseLine("socks5://1.2.3.4:1080")
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	// socks5 для yt-dlp явно превращаем в socks5h (remote DNS).
	if got := p.URL(); got != "socks5h://1.2.3.4:1080" {
		t.Errorf("URL без учётных данных = %q", got)
	}

	// Схема socks5h (remote DNS) должна проходить без изменений.
	socks5h, err := ParseLine("socks5h://1.2.3.4:1080")
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	if got := socks5h.URL(); got != "socks5h://1.2.3.4:1080" {
		t.Errorf("URL socks5h = %q", got)
	}

	// Прочие схемы не меняем.
	httpProxy, err := ParseLine("http://1.2.3.4:1080")
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	if got := httpProxy.URL(); got != "http://1.2.3.4:1080" {
		t.Errorf("URL http = %q", got)
	}

	withCreds, err := ParseLine("socks5://bob:p@ss:w0rd@1.2.3.4:1080")
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	// Учётные данные должны быть URL-экранированы.
	want := "socks5h://bob:p%40ss%3Aw0rd@1.2.3.4:1080"
	if got := withCreds.URL(); got != want {
		t.Errorf("URL с учётными данными = %q, want %q", got, want)
	}
}

func TestProxyDisplayHidesPassword(t *testing.T) {
	p, err := ParseLine("socks5://bob:secret@1.2.3.4:1080")
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	got := p.Display()
	if got != "socks5://bob@1.2.3.4:1080" {
		t.Errorf("Display = %q", got)
	}
}

func TestProxyIDIsStable(t *testing.T) {
	a, _ := ParseLine("socks5://1.2.3.4:1080")
	b, _ := ParseLine("socks5://1.2.3.4:1080")
	if a.ID != b.ID {
		t.Errorf("ID нестабилен: %q != %q", a.ID, b.ID)
	}

	c, _ := ParseLine("socks5://1.2.3.4:1081")
	if a.ID == c.ID {
		t.Error("разные прокси должны иметь разные ID")
	}
}
