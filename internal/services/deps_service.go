package services

import (
	"context"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"loadika-desktop/internal/config"
	"loadika-desktop/internal/deps"
	"loadika-desktop/internal/events"
)

// DepsService управляет внешними зависимостями (yt-dlp, deno, ffmpeg) из интерфейса.
type DepsService struct {
	manager     *deps.Manager
	configStore *config.Store
}

// NewDepsService создаёт сервис зависимостей.
func NewDepsService(manager *deps.Manager, cfg *config.Store) *DepsService {
	return &DepsService{manager: manager, configStore: cfg}
}

// GetDependencies возвращает список зависимостей с локальным статусом (без сети).
func (s *DepsService) GetDependencies() []deps.Dependency {
	return s.manager.InspectAll()
}

// CheckUpdates проверяет свежие версии зависимостей (обращается к сети).
func (s *DepsService) CheckUpdates() ([]deps.Dependency, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.manager.CheckUpdates(ctx, s.configStore.Get().YtDlpChannel)
}

// Install устанавливает или обновляет зависимость по имени, используя настроенный канал.
// Прогресс установки стримится событием `dep:progress`, завершение — `dep:state`.
func (s *DepsService) Install(name string) (deps.Dependency, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	th := newThrottle(100 * time.Millisecond)
	dep, err := s.manager.Install(ctx, name, s.configStore.Get().YtDlpChannel, func(p deps.Progress) {
		if !th.allow(p.Phase) {
			return
		}
		emit(events.DepProgressName, events.DepProgress{
			Name:     name,
			Phase:    p.Phase,
			Received: p.Received,
			Total:    p.Total,
		})
	})
	if err != nil {
		return deps.Dependency{}, err
	}

	emit(events.DepStateName, events.DepState{Name: dep.Name, Status: dep.Status, Version: dep.Version})
	return dep, nil
}

// Remove удаляет управляемую копию зависимости по имени.
func (s *DepsService) Remove(name string) error {
	return s.manager.Remove(name)
}

// RemoveAllManaged удаляет все управляемые (скачанные приложением) копии зависимостей,
// не затрагивая системные.
func (s *DepsService) RemoveAllManaged() error {
	return s.manager.RemoveAll()
}

// SetSourcePolicy сохраняет политику использования источника для зависимости.
func (s *DepsService) SetSourcePolicy(name, policy string) error {
	if !deps.IsValidSource(policy) {
		return fmt.Errorf("неизвестный источник зависимости: %q", policy)
	}
	if _, err := s.manager.Inspect(name); err != nil {
		return err
	}
	return s.configStore.Update(func(c *config.Config) {
		if c.DepSources == nil {
			c.DepSources = map[string]string{}
		}
		c.DepSources[name] = policy
	})
}

// emit отправляет событие во фронтенд, если приложение уже создано.
func emit(name string, data any) {
	if app := application.Get(); app != nil {
		app.Event.Emit(name, data)
	}
}
