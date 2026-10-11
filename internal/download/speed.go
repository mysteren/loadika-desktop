package download

import (
	"context"
	"sync"
	"time"
)

// speedGuard наблюдает за скоростью скачивания одной попытки: хранит последнюю и максимальную
// скорость и время последнего замера. Используется сторожем скорости (watchSpeed).
type speedGuard struct {
	mu        sync.Mutex
	bps       float64
	maxBps    float64
	lastAt    time.Time
	slowSince time.Time
}

// observe фиксирует очередной замер скорости (байт/с).
func (g *speedGuard) observe(bps float64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.bps = bps
	g.lastAt = time.Now()
	if bps > g.maxBps {
		g.maxBps = bps
	}
}

// peak возвращает максимальную наблюдённую скорость — для телеметрии пула.
func (g *speedGuard) peak() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.maxBps
}

// watchSpeed запускает сторож: если скорость держится ниже порога дольше slowDuration, попытка
// отменяется через cancel с причиной errTooSlow — чтобы переключиться на другой прокси. Возвращает
// функцию остановки сторожа. Учитываются только свежие замеры (фаза скачивания), поэтому сторож не
// срабатывает на паузах post-processing (когда прогресс не идёт).
func (m *Manager) watchSpeed(ctx context.Context, cancel context.CancelCauseFunc, g *speedGuard) func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(m.speedInterval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				if tooSlow(g, m.minSpeedBps, m.speedInterval, m.slowDuration) {
					cancel(errTooSlow)
					return
				}
			}
		}
	}()
	return func() { close(done) }
}

// tooSlow сообщает, держится ли скорость ниже порога дольше slowFor. Устанавливает/сбрасывает
// отметку начала «медленного» периода.
func tooSlow(g *speedGuard, minBps float64, interval, slowFor time.Duration) bool {
	now := time.Now()

	g.mu.Lock()
	defer g.mu.Unlock()

	// «Медленно» считаем, только если замеры свежие (идёт скачивание) и скорость положительна.
	recent := !g.lastAt.IsZero() && now.Sub(g.lastAt) < 3*interval
	if !recent || g.bps <= 0 || g.bps >= minBps {
		g.slowSince = time.Time{}
		return false
	}
	if g.slowSince.IsZero() {
		g.slowSince = now
		return false
	}
	return now.Sub(g.slowSince) >= slowFor
}
