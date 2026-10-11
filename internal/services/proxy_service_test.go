package services

import (
	"path/filepath"
	"testing"

	"loadika-desktop/internal/config"
	"loadika-desktop/internal/proxy"
)

func newProxyServiceTest(t *testing.T) *ProxyService {
	t.Helper()
	dir := t.TempDir()

	store, err := proxy.Load(filepath.Join(dir, "proxies.json"))
	if err != nil {
		t.Fatalf("proxy.Load: %v", err)
	}
	pool := proxy.NewPool(store)
	cfg, err := config.Load(filepath.Join(dir, "config.json"), config.Config{
		SchemaVersion: config.CurrentSchemaVersion,
		DownloadsDir:  dir,
		YtDlpChannel:  config.DefaultYtDlpChannel,
		ProxyStrategy: config.DefaultProxyStrategy,
	})
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	return NewProxyService(pool, store, cfg)
}

func TestProxyServiceImportMaskingAndRemove(t *testing.T) {
	s := newProxyServiceTest(t)

	res, err := s.Import("socks5://user:secret@1.2.3.4:1080\nsocks5://5.6.7.8:1080\nбитая строка")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if res.Added != 2 || res.Total != 2 {
		t.Fatalf("Import = %+v, want added=2 total=2", res)
	}

	views := s.GetProxies()
	if len(views) != 2 {
		t.Fatalf("прокси %d, want 2", len(views))
	}
	for _, v := range views {
		if v.Password != "" {
			t.Errorf("пароль не должен отдаваться во фронтенд: %q", v.Password)
		}
	}

	if err := s.SetEnabled(views[0].ID, false); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	if s.GetProxies()[0].Enabled {
		t.Error("прокси должен быть выключен")
	}

	if err := s.Remove(views[0].ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if got := len(s.GetProxies()); got != 1 {
		t.Errorf("после удаления %d, want 1", got)
	}

	if err := s.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if got := len(s.GetProxies()); got != 0 {
		t.Errorf("после очистки %d, want 0", got)
	}
}

func TestProxyServiceStrategy(t *testing.T) {
	s := newProxyServiceTest(t)

	if err := s.SetStrategy("random"); err != nil {
		t.Fatalf("SetStrategy: %v", err)
	}
	if err := s.SetStrategy("bogus"); err == nil {
		t.Error("ожидалась ошибка для недопустимой стратегии")
	}
	if len(s.GetStrategies()) == 0 {
		t.Error("список стратегий пуст")
	}
}

func TestProxyServiceCheckEmpty(t *testing.T) {
	s := newProxyServiceTest(t)
	if views := s.Check(); len(views) != 0 {
		t.Errorf("пустой пул: проверка вернула %d, want 0", len(views))
	}
}
