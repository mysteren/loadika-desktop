package services

import (
	"sync"
	"time"
)

// throttle ограничивает частоту однотипных событий: пропускает первое событие фазы
// и далее не чаще, чем раз в interval. Так webview не заваливается прогрессом,
// приходящим на каждый прочитанный блок.
type throttle struct {
	mu        sync.Mutex
	interval  time.Duration
	last      time.Time
	lastPhase string
}

// newThrottle создаёт троттлер с заданным минимальным интервалом между событиями.
func newThrottle(interval time.Duration) *throttle {
	return &throttle{interval: interval}
}

// allow сообщает, нужно ли отправить событие сейчас: при смене фазы — всегда,
// иначе — только если прошёл интервал.
func (t *throttle) allow(phase string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	if phase != t.lastPhase || now.Sub(t.last) >= t.interval {
		t.lastPhase = phase
		t.last = now
		return true
	}
	return false
}
