package proxy

import (
	"math/rand/v2"
	"sort"
	"sync"
	"time"
)

// Состояния работоспособности прокси.
const (
	HealthUnknown = "unknown"
	HealthOK      = "ok"
	HealthFail    = "fail"
)

// Стратегии выбора прокси.
const (
	StrategyRoundRobin    = "round-robin"
	StrategyRandom        = "random"
	StrategyLeastUsed     = "least-used"
	StrategyLowestLatency = "lowest-latency"
	StrategySticky        = "sticky"
)

// Defaults — параметры обработки сбоев по умолчанию.
const (
	DefaultFailThreshold = 3
	DefaultCooldownBase  = 60 * time.Second
	DefaultCooldownMax   = 30 * time.Minute
)

// Strategies — допустимые стратегии в порядке отображения.
var Strategies = []string{StrategyRoundRobin, StrategyRandom, StrategyLeastUsed, StrategyLowestLatency, StrategySticky}

// IsValidStrategy сообщает, допустима ли стратегия.
func IsValidStrategy(s string) bool {
	for _, v := range Strategies {
		if v == s {
			return true
		}
	}
	return false
}

// State — рантайм-состояние прокси (в памяти, не персистится).
type State struct {
	Health        string
	LatencyMs     int64
	SpeedBps      float64 // сглаженная измеренная скорость скачивания, байт/с
	FailCount     int
	CooldownUntil int64 // UnixMilli
	LastUsedAt    int64 // UnixMilli
	UseCount      int
}

// View — прокси вместе с рантайм-состоянием (для UI).
type View struct {
	Proxy
	Health        string  `json:"health"`
	LatencyMs     int64   `json:"latencyMs"`
	SpeedBps      float64 `json:"speedBps"`
	FailCount     int     `json:"failCount"`
	CooldownUntil int64   `json:"cooldownUntil"`
	LastUsedAt    int64   `json:"lastUsedAt"`
	UseCount      int     `json:"useCount"`
}

// Pool управляет выбором прокси из хранилища: доступность, ротация по стратегии и обработка
// сбоев (failCount + cooldown). Безопасен для конкурентного доступа.
type Pool struct {
	store    *Store
	mu       sync.Mutex
	states   map[string]*State
	strategy string
	rrIndex  int
	stickyID string

	failThreshold int
	cooldownBase  time.Duration
	cooldownMax   time.Duration
	now           func() time.Time
}

// NewPool создаёт пул поверх хранилища определений прокси.
func NewPool(store *Store) *Pool {
	return &Pool{
		store:         store,
		states:        make(map[string]*State),
		strategy:      StrategyRoundRobin,
		failThreshold: DefaultFailThreshold,
		cooldownBase:  DefaultCooldownBase,
		cooldownMax:   DefaultCooldownMax,
		now:           time.Now,
	}
}

// SetStrategy задаёт стратегию выбора (если недопустима — round-robin).
func (p *Pool) SetStrategy(s string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if IsValidStrategy(s) {
		p.strategy = s
	}
}

// Strategy возвращает текущую стратегию.
func (p *Pool) Strategy() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.strategy
}

// Forget удаляет рантайм-состояние прокси (например, при удалении из пула).
func (p *Pool) Forget(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.states, id)
}

// ForgetAll сбрасывает всё рантайм-состояние (например, при очистке пула).
func (p *Pool) ForgetAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.states = make(map[string]*State)
	p.stickyID = ""
	p.rrIndex = 0
}

// Proxies возвращает определения всех прокси (без рантайм-состояния).
func (p *Pool) Proxies() []Proxy {
	return p.store.List()
}

// Views возвращает прокси с их рантайм-состоянием в порядке хранения.
func (p *Pool) Views() []View {
	p.mu.Lock()
	defer p.mu.Unlock()

	list := p.store.List()
	out := make([]View, 0, len(list))
	for _, pr := range list {
		pr.Password = "" // пароль не отдаём во фронтенд
		st := p.stateLocked(pr.ID)
		out = append(out, View{
			Proxy:         pr,
			Health:        st.Health,
			LatencyMs:     st.LatencyMs,
			SpeedBps:      st.SpeedBps,
			FailCount:     st.FailCount,
			CooldownUntil: st.CooldownUntil,
			LastUsedAt:    st.LastUsedAt,
			UseCount:      st.UseCount,
		})
	}
	return out
}

// EnabledCount возвращает число включённых прокси (без учёта cooldown).
func (p *Pool) EnabledCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	n := 0
	for _, pr := range p.store.List() {
		if pr.Enabled {
			n++
		}
	}
	return n
}

// HasAvailable сообщает, есть ли доступный прокси (включён, вне cooldown) кроме exclude.
func (p *Pool) HasAvailable(exclude map[string]bool) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.availableLocked(exclude)) > 0
}

// Pick выбирает доступный прокси по стратегии и помечает его использованным. exclude — уже
// пробованные прокси (для retry после сбоя); может быть nil.
func (p *Pool) Pick(exclude map[string]bool) (Proxy, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	avail := p.availableLocked(exclude)
	if len(avail) == 0 {
		return Proxy{}, false
	}

	var chosen Proxy
	switch p.strategy {
	case StrategyRandom:
		chosen = avail[rand.IntN(len(avail))]
	case StrategyLeastUsed:
		chosen = avail[0]
		for _, pr := range avail[1:] {
			if p.stateLocked(pr.ID).UseCount < p.stateLocked(chosen.ID).UseCount {
				chosen = pr
			}
		}
	case StrategyLowestLatency:
		chosen = avail[0]
		for _, pr := range avail[1:] {
			if latencyRank(p.stateLocked(pr.ID)) < latencyRank(p.stateLocked(chosen.ID)) {
				chosen = pr
			}
		}
	case StrategySticky:
		chosen = p.stickyLocked(avail)
	default: // round-robin
		chosen = p.roundRobinLocked(avail)
	}

	st := p.stateLocked(chosen.ID)
	st.UseCount++
	st.LastUsedAt = p.now().UnixMilli()
	return chosen, true
}

// ReportSuccess отмечает успешное использование прокси и измеренную задержку.
func (p *Pool) ReportSuccess(id string, latencyMs int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	st := p.stateLocked(id)
	st.Health = HealthOK
	st.FailCount = 0
	st.CooldownUntil = 0
	if latencyMs >= 0 {
		st.LatencyMs = latencyMs
	}
}

// ReportSpeed фиксирует измеренную скорость скачивания через прокси (байт/с). Значения
// сглаживаются (EMA), чтобы разовая просадка не портила оценку. Неположительные — игнорируются.
func (p *Pool) ReportSpeed(id string, bps float64) {
	if bps <= 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	st := p.stateLocked(id)
	if st.SpeedBps <= 0 {
		st.SpeedBps = bps
		return
	}
	st.SpeedBps = 0.7*st.SpeedBps + 0.3*bps
}

// ReportFailure фиксирует сбой прокси: считает сбои и, при достижении порога, отправляет прокси
// в cooldown с экспоненциальным ростом.
func (p *Pool) ReportFailure(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	st := p.stateLocked(id)
	st.FailCount++
	if st.FailCount >= p.failThreshold {
		st.Health = HealthFail
		st.CooldownUntil = p.now().Add(p.backoff(st.FailCount)).UnixMilli()
	} else {
		st.Health = HealthUnknown
	}
}

// Reject немедленно помечает прокси нездоровым и отправляет его в cooldown, не дожидаясь порога
// сбоев. Используется предварительной проверкой: мёртвый узел надо отсеять сразу, а не тратить на
// него время в попытках.
func (p *Pool) Reject(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	st := p.stateLocked(id)
	st.FailCount = p.failThreshold
	st.Health = HealthFail
	st.CooldownUntil = p.now().Add(p.backoff(st.FailCount)).UnixMilli()
}

// PickBest выбирает доступный прокси с наилучшими показателями: сначала здоровые (HealthOK),
// затем по наименьшей задержке, затем по наибольшей скорости. Если данных о прокси нет —
// исследует по кругу. exclude — уже пробованные. Это «умный» выбор для операций, где важна
// надёжность/скорость ответа (например, определение качества).
func (p *Pool) PickBest(exclude map[string]bool) (Proxy, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	avail := p.availableLocked(exclude)
	if len(avail) == 0 {
		return Proxy{}, false
	}

	informative := false
	for _, pr := range avail {
		st := p.stateLocked(pr.ID)
		if st.Health == HealthOK || st.LatencyMs > 0 || st.SpeedBps > 0 {
			informative = true
			break
		}
	}

	var chosen Proxy
	if informative {
		sort.SliceStable(avail, func(i, j int) bool { return p.betterLocked(avail[i], avail[j]) })
		chosen = avail[0]
	} else {
		chosen = p.roundRobinLocked(avail)
	}

	st := p.stateLocked(chosen.ID)
	st.UseCount++
	st.LastUsedAt = p.now().UnixMilli()
	return chosen, true
}

// betterLocked сообщает, лучше ли прокси a, чем b (меньше — лучше): здоровье, задержка, скорость.
func (p *Pool) betterLocked(a, b Proxy) bool {
	sa, sb := p.stateLocked(a.ID), p.stateLocked(b.ID)
	if hr := healthRank(sa.Health) - healthRank(sb.Health); hr != 0 {
		return hr < 0
	}
	if la, lb := latencyRank(sa), latencyRank(sb); la != lb {
		return la < lb
	}
	return sa.SpeedBps > sb.SpeedBps
}

// healthRank задаёт порядок здоровья: ok < unknown < fail (меньше — лучше).
func healthRank(health string) int {
	switch health {
	case HealthOK:
		return 0
	case HealthUnknown:
		return 1
	case HealthFail:
		return 2
	default:
		return 1
	}
}

// PickFastest выбирает доступный прокси с наибольшей измеренной скоростью. Если скорость не
// известна ни у одного — ведёт себя как обычный выбор (исследование): по кругу. exclude — уже
// пробованные прокси; может быть nil.
func (p *Pool) PickFastest(exclude map[string]bool) (Proxy, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	avail := p.availableLocked(exclude)
	if len(avail) == 0 {
		return Proxy{}, false
	}

	known := false
	chosen := avail[0]
	for _, pr := range avail {
		if st := p.stateLocked(pr.ID); st.SpeedBps > 0 {
			known = true
			if st.SpeedBps > p.stateLocked(chosen.ID).SpeedBps {
				chosen = pr
			}
		}
	}
	if !known {
		chosen = p.roundRobinLocked(avail)
	}

	st := p.stateLocked(chosen.ID)
	st.UseCount++
	st.LastUsedAt = p.now().UnixMilli()
	return chosen, true
}

// availableLocked возвращает включённые прокси вне cooldown, кроме тех, что в exclude.
func (p *Pool) availableLocked(exclude map[string]bool) []Proxy {
	now := p.now().UnixMilli()
	var out []Proxy
	for _, pr := range p.store.List() {
		if !pr.Enabled || exclude[pr.ID] {
			continue
		}
		if st := p.stateLocked(pr.ID); st.CooldownUntil > now {
			continue
		}
		out = append(out, pr)
	}
	return out
}

// roundRobinLocked выдаёт доступные прокси по кругу.
func (p *Pool) roundRobinLocked(avail []Proxy) Proxy {
	chosen := avail[p.rrIndex%len(avail)]
	p.rrIndex++
	return chosen
}

// stickyLocked держит один прокси, пока он доступен; иначе выбирает новый по кругу.
func (p *Pool) stickyLocked(avail []Proxy) Proxy {
	for _, pr := range avail {
		if pr.ID == p.stickyID {
			return pr
		}
	}
	chosen := p.roundRobinLocked(avail)
	p.stickyID = chosen.ID
	return chosen
}

// stateLocked возвращает состояние прокси, создавая его при необходимости (вызывается при mu).
func (p *Pool) stateLocked(id string) *State {
	st := p.states[id]
	if st == nil {
		st = &State{Health: HealthUnknown}
		p.states[id] = st
	}
	return st
}

// backoff вычисляет cooldown по числу сбоев: base * 2^(n-1), не выше cooldownMax.
func (p *Pool) backoff(failCount int) time.Duration {
	d := p.cooldownBase
	for i := 1; i < failCount && d < p.cooldownMax; i++ {
		d *= 2
	}
	if d > p.cooldownMax || d <= 0 {
		d = p.cooldownMax
	}
	return d
}

// latencyRank — ключ сортировки по задержке: неизвестная задержка считается худшей.
func latencyRank(st *State) int64 {
	if st.LatencyMs <= 0 {
		return 1 << 62
	}
	return st.LatencyMs
}
