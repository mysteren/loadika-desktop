package services

import (
	"context"
	"fmt"
	"time"

	"loadika-desktop/internal/config"
	"loadika-desktop/internal/events"
	"loadika-desktop/internal/proxy"
)

// healthCheckTimeout — таймаут одной проверки прокси.
const healthCheckTimeout = 5 * time.Second

// ImportResult — результат импорта списка прокси.
type ImportResult struct {
	Added   int `json:"added"`
	Total   int `json:"total"`
	Skipped int `json:"skipped"`
}

// ProxyService управляет пулом прокси из интерфейса.
type ProxyService struct {
	pool        *proxy.Pool
	store       *proxy.Store
	configStore *config.Store
}

// NewProxyService создаёт сервис прокси.
func NewProxyService(pool *proxy.Pool, store *proxy.Store, cfg *config.Store) *ProxyService {
	return &ProxyService{pool: pool, store: store, configStore: cfg}
}

// GetProxies возвращает прокси с рантайм-состоянием (health, задержка, сбои, cooldown).
func (s *ProxyService) GetProxies() []proxy.View {
	return s.pool.Views()
}

// GetStrategies возвращает допустимые стратегии выбора прокси.
func (s *ProxyService) GetStrategies() []string {
	return proxy.Strategies
}

// Import разбирает список прокси (по строке) и добавляет новые. Возвращает число добавленных,
// пропущенных (битых строк) и всего в пуле.
func (s *ProxyService) Import(text string) (ImportResult, error) {
	parsed := proxy.ParseList(text)
	added, err := s.store.Add(parsed.Proxies)
	if err != nil {
		return ImportResult{}, err
	}
	return ImportResult{Added: added, Total: len(s.store.List()), Skipped: parsed.Skipped}, nil
}

// Remove удаляет прокси по идентификатору.
func (s *ProxyService) Remove(id string) error {
	if err := s.store.Remove(id); err != nil {
		return err
	}
	s.pool.Forget(id)
	return nil
}

// SetEnabled включает/выключает прокси.
func (s *ProxyService) SetEnabled(id string, enabled bool) error {
	return s.store.SetEnabled(id, enabled)
}

// Clear очищает пул.
func (s *ProxyService) Clear() error {
	if err := s.store.Clear(); err != nil {
		return err
	}
	s.pool.ForgetAll()
	return nil
}

// SetStrategy задаёт стратегию выбора прокси и сохраняет её в настройках.
func (s *ProxyService) SetStrategy(strategy string) error {
	if !proxy.IsValidStrategy(strategy) {
		return fmt.Errorf("неизвестная стратегия выбора прокси: %q", strategy)
	}
	s.pool.SetStrategy(strategy)
	return s.configStore.Update(func(c *config.Config) { c.ProxyStrategy = strategy })
}

// Check проверяет доступность включённых прокси, обновляет их состояние и возвращает список.
// Результаты по каждому прокси стримятся событием proxy:health.
func (s *ProxyService) Check() []proxy.View {
	list := s.pool.Proxies()
	enabled := make([]proxy.Proxy, 0, len(list))
	for _, p := range list {
		if p.Enabled {
			enabled = append(enabled, p)
		}
	}
	if len(enabled) == 0 {
		return s.pool.Views()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*healthCheckTimeout)
	defer cancel()

	for _, r := range proxy.CheckAll(ctx, enabled, "", healthCheckTimeout) {
		if r.Err != nil {
			s.pool.ReportFailure(r.ID)
			emit(events.ProxyHealthName, events.ProxyHealth{ProxyID: r.ID, OK: false})
			continue
		}
		s.pool.ReportSuccess(r.ID, r.LatencyMs)
		emit(events.ProxyHealthName, events.ProxyHealth{ProxyID: r.ID, OK: true, LatencyMs: r.LatencyMs})
	}
	return s.pool.Views()
}
