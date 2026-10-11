package services

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"loadika-desktop/internal/proxy"
)

// newPreflightService создаёт сервис с пулом из строк и подменённой проверкой прокси.
// checkAll получает список включённых прокси и возвращает заданные результаты.
func newPreflightService(t *testing.T, lines string, checkAll healthChecker) (*DownloadService, *proxy.Store) {
	t.Helper()
	store, err := proxy.Load(filepath.Join(t.TempDir(), "proxies.json"))
	if err != nil {
		t.Fatalf("proxy.Load: %v", err)
	}
	if _, err := store.Add(proxy.ParseList(lines).Proxies); err != nil {
		t.Fatalf("proxy.Add: %v", err)
	}
	pool := proxy.NewPool(store)
	return &DownloadService{pool: pool, checkAll: checkAll}, store
}

func TestPreflightRejectsDeadProxies(t *testing.T) {
	checkAll := func(_ context.Context, proxies []proxy.Proxy, _ string, _ time.Duration) []proxy.HealthResult {
		// Первый прокси живой, второй — мёртвый.
		return []proxy.HealthResult{
			{ID: proxies[0].ID, LatencyMs: 42},
			{ID: proxies[1].ID, Err: errors.New("dead")},
		}
	}
	svc, store := newPreflightService(t, "socks5://1.1.1.1:1080\nsocks5://2.2.2.2:1080", checkAll)

	alive, checked := svc.preflightProxies("www.youtube.com:443")
	if checked != 2 {
		t.Fatalf("checked = %d, want 2", checked)
	}
	if alive != 1 {
		t.Fatalf("alive = %d, want 1", alive)
	}

	// Мёртвый прокси отсеян: Pick выдаёт только живой.
	list := store.List()
	got, ok := svc.pool.Pick(nil)
	if !ok || got.ID != list[0].ID {
		t.Fatalf("выбран %+v, want первый (живой) прокси", got)
	}

	// Состояние в пуле отражает результат: мёртвый — fail.
	views := svc.pool.Views()
	if views[0].Health != proxy.HealthOK {
		t.Errorf("health живого = %q, want %q", views[0].Health, proxy.HealthOK)
	}
	if views[1].Health != proxy.HealthFail {
		t.Errorf("health мёртвого = %q, want %q", views[1].Health, proxy.HealthFail)
	}
}

func TestPreflightNoProxies(t *testing.T) {
	svc, _ := newPreflightService(t, "", func(_ context.Context, _ []proxy.Proxy, _ string, _ time.Duration) []proxy.HealthResult {
		t.Error("проверка не должна вызываться для пустого пула")
		return nil
	})
	if alive, checked := svc.preflightProxies(""); alive != 0 || checked != 0 {
		t.Errorf("preflightProxies = (%d, %d), want (0, 0)", alive, checked)
	}
}

func TestCheckTarget(t *testing.T) {
	if got := checkTarget("https://www.youtube.com/watch?v=x"); got != "www.youtube.com:443" {
		t.Errorf("checkTarget = %q", got)
	}
	if got := checkTarget(""); got != "" {
		t.Errorf("checkTarget(\"\") = %q, want пусто", got)
	}
}
