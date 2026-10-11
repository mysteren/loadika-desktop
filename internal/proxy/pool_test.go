package proxy

import (
	"path/filepath"
	"testing"
	"time"
)

// newTestPool создаёт пул с прокси из строк (каждая — socks5://host:port).
func newTestPool(t *testing.T, lines string) (*Pool, *Store) {
	t.Helper()
	store, err := Load(filepath.Join(t.TempDir(), "proxies.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := store.Add(ParseList(lines).Proxies); err != nil {
		t.Fatalf("Add: %v", err)
	}
	return NewPool(store), store
}

const twoProxies = "socks5://1.1.1.1:1080\nsocks5://2.2.2.2:1080"

// TestPoolRejectBenchesImmediately проверяет, что Reject сразу уводит прокси в cooldown
// (без ожидания порога сбоев), и он перестаёт выбираться.
func TestPoolRejectBenchesImmediately(t *testing.T) {
	p, store := newTestPool(t, twoProxies)
	p.now = func() time.Time { return time.Unix(1000, 0) }
	list := store.List()

	p.Reject(list[0].ID)

	st := p.stateLocked(list[0].ID)
	if st.Health != HealthFail || st.CooldownUntil == 0 {
		t.Errorf("состояние после Reject = %+v", st)
	}

	// Доступным остаётся только второй прокси.
	for i := 0; i < 2; i++ {
		pr, ok := p.Pick(nil)
		if !ok || pr.ID != list[1].ID {
			t.Fatalf("выбран %+v, want второй прокси", pr)
		}
	}
}

func TestPoolReportSpeed(t *testing.T) {
	p, store := newTestPool(t, "socks5://1.1.1.1:1080")
	id := store.List()[0].ID

	p.ReportSpeed(id, 1000)
	if got := p.stateLocked(id).SpeedBps; got != 1000 {
		t.Errorf("после первого замера = %v, want 1000", got)
	}
	// EMA: 0.7*1000 + 0.3*2000 = 1300.
	p.ReportSpeed(id, 2000)
	if got := p.stateLocked(id).SpeedBps; got != 1300 {
		t.Errorf("после второго замера = %v, want 1300", got)
	}
	// Неположительные значения игнорируются.
	p.ReportSpeed(id, 0)
	if got := p.stateLocked(id).SpeedBps; got != 1300 {
		t.Errorf("после нулевого замера = %v, want 1300", got)
	}
}

func TestPoolPickFastest(t *testing.T) {
	p, store := newTestPool(t, twoProxies)
	p.now = func() time.Time { return time.Unix(1000, 0) }
	list := store.List()

	p.ReportSpeed(list[0].ID, 100)
	p.ReportSpeed(list[1].ID, 5000)

	pr, ok := p.PickFastest(nil)
	if !ok || pr.ID != list[1].ID {
		t.Fatalf("PickFastest выбрал %+v, want быстрый прокси", pr)
	}

	// Если скорость неизвестна у всех — выбираем хоть какой-то (исследование).
	fresh, _ := newTestPool(t, twoProxies)
	if _, ok := fresh.PickFastest(nil); !ok {
		t.Error("при неизвестных скоростях выбор должен всё равно произойти")
	}
}

func TestPoolPickBest(t *testing.T) {
	p, store := newTestPool(t, "socks5://1.1.1.1:1080\nsocks5://2.2.2.2:1080\nsocks5://3.3.3.3:1080")
	p.now = func() time.Time { return time.Unix(1000, 0) }
	list := store.List()

	// Оба здоровые; второй — с меньшей задержкой; третий — без данных.
	p.ReportSuccess(list[0].ID, 300)
	p.ReportSuccess(list[1].ID, 50)

	pr, ok := p.PickBest(nil)
	if !ok || pr.ID != list[1].ID {
		t.Fatalf("PickBest выбрал %+v, want прокси с наименьшей задержкой", pr)
	}

	// Без данных о прокси выбор всё равно происходит.
	fresh, _ := newTestPool(t, twoProxies)
	if _, ok := fresh.PickBest(nil); !ok {
		t.Error("без данных о прокси выбор всё равно должен быть")
	}
}

func TestPoolHasAvailable(t *testing.T) {
	p, store := newTestPool(t, twoProxies)
	list := store.List()

	if !p.HasAvailable(nil) {
		t.Error("должен быть доступный прокси")
	}
	if p.HasAvailable(map[string]bool{list[0].ID: true, list[1].ID: true}) {
		t.Error("при исключении всех прокси доступных быть не должно")
	}
}

func TestPoolRoundRobin(t *testing.T) {
	p, _ := newTestPool(t, "socks5://1.1.1.1:1080\nsocks5://2.2.2.2:1080\nsocks5://3.3.3.3:1080")
	p.now = func() time.Time { return time.Unix(1000, 0) }

	seen := make(map[string]bool)
	for i := 0; i < 3; i++ {
		pr, ok := p.Pick(nil)
		if !ok {
			t.Fatal("нет доступного прокси")
		}
		seen[pr.Host] = true
	}
	if len(seen) != 3 {
		t.Errorf("round-robin не покрыл все прокси: %v", seen)
	}
}

func TestPoolSkipsDisabledAndExhausted(t *testing.T) {
	p, store := newTestPool(t, twoProxies)
	p.now = func() time.Time { return time.Unix(1000, 0) }
	list := store.List()

	if err := store.SetEnabled(list[0].ID, false); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	pr, ok := p.Pick(nil)
	if !ok || pr.ID != list[1].ID {
		t.Fatalf("выбран %+v, want второй прокси", pr)
	}

	if err := store.SetEnabled(list[1].ID, false); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	if _, ok := p.Pick(nil); ok {
		t.Error("при выключенных прокси выбора быть не должно")
	}
}

func TestPoolCooldownAfterFailures(t *testing.T) {
	p, _ := newTestPool(t, "socks5://1.1.1.1:1080")
	start := time.Unix(1000, 0)
	p.now = func() time.Time { return start }

	pr, ok := p.Pick(nil)
	if !ok {
		t.Fatal("нет прокси")
	}
	for i := 0; i < DefaultFailThreshold; i++ {
		p.ReportFailure(pr.ID)
	}
	if _, ok := p.Pick(nil); ok {
		t.Error("после порога сбоев прокси должен быть в cooldown")
	}

	// Время за cooldown (base*2^(n-1)) — прокси снова доступен.
	p.now = func() time.Time { return start.Add(DefaultCooldownBase * 4) }
	if _, ok := p.Pick(nil); !ok {
		t.Error("после cooldown прокси должен быть доступен снова")
	}
}

func TestPoolLeastUsed(t *testing.T) {
	p, _ := newTestPool(t, twoProxies)
	p.now = func() time.Time { return time.Unix(1000, 0) }
	p.SetStrategy(StrategyLeastUsed)

	first, ok := p.Pick(nil)
	if !ok {
		t.Fatal("нет прокси")
	}
	second, ok := p.Pick(nil)
	if !ok {
		t.Fatal("нет прокси")
	}
	if first.ID == second.ID {
		t.Error("least-used должен выбрать менее использованный прокси")
	}
}

func TestPoolLowestLatency(t *testing.T) {
	p, store := newTestPool(t, twoProxies)
	p.now = func() time.Time { return time.Unix(1000, 0) }
	list := store.List()

	p.ReportSuccess(list[0].ID, 200)
	p.ReportSuccess(list[1].ID, 50)
	p.SetStrategy(StrategyLowestLatency)

	pr, ok := p.Pick(nil)
	if !ok || pr.ID != list[1].ID {
		t.Fatalf("выбран %+v, want прокси с меньшей задержкой (%s)", pr, list[1].ID)
	}
}

func TestPoolSticky(t *testing.T) {
	p, _ := newTestPool(t, twoProxies)
	p.now = func() time.Time { return time.Unix(1000, 0) }
	p.SetStrategy(StrategySticky)

	a, _ := p.Pick(nil)
	b, _ := p.Pick(nil)
	if a.ID != b.ID {
		t.Errorf("sticky должен держать один прокси: %s != %s", a.ID, b.ID)
	}
}

func TestPoolPickExcluding(t *testing.T) {
	p, store := newTestPool(t, twoProxies)
	p.now = func() time.Time { return time.Unix(1000, 0) }
	list := store.List()

	pr, ok := p.Pick(map[string]bool{list[0].ID: true})
	if !ok || pr.ID != list[1].ID {
		t.Fatalf("выбран %+v, want %s (первый исключён)", pr, list[1].ID)
	}
	if _, ok := p.Pick(map[string]bool{list[0].ID: true, list[1].ID: true}); ok {
		t.Error("при исключении всех прокси выбора быть не должно")
	}
}

func TestPoolReportSuccessResetsFailures(t *testing.T) {
	p, _ := newTestPool(t, "socks5://1.1.1.1:1080")
	p.now = func() time.Time { return time.Unix(1000, 0) }

	pr, _ := p.Pick(nil)
	p.ReportFailure(pr.ID)
	p.ReportSuccess(pr.ID, 42)

	st := p.stateLocked(pr.ID)
	if st.Health != HealthOK || st.FailCount != 0 || st.CooldownUntil != 0 || st.LatencyMs != 42 {
		t.Errorf("состояние после успеха = %+v", st)
	}
}
