package deps

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Manager управляет внешними зависимостями (yt-dlp, deno, ffmpeg).
type Manager struct {
	dir    string
	mu     sync.Mutex
	policy func(name string) string // политика источника зависимости; nil → auto
}

// NewManager создаёт менеджер зависимостей с каталогом управляемых файлов.
func NewManager(dir string) *Manager {
	return &Manager{dir: dir}
}

// SetPolicyResolver задаёт поставщика политик источника зависимости (обычно — настройки).
func (m *Manager) SetPolicyResolver(fn func(name string) string) {
	m.policy = fn
}

// sourcePolicy возвращает политику источника для зависимости (auto по умолчанию).
func (m *Manager) sourcePolicy(name string) string {
	if m.policy == nil {
		return SourceAuto
	}
	if p := m.policy(name); IsValidSource(p) {
		return p
	}
	return SourceAuto
}

func (m *Manager) manifestPath() string {
	return filepath.Join(m.dir, "deps.json")
}

// InspectAll возвращает сведения обо всех зависимостях без обращения к сети.
func (m *Manager) InspectAll() []Dependency {
	out := make([]Dependency, 0, len(specs))
	for _, s := range specs {
		out = append(out, m.inspectSpec(s))
	}
	return out
}

// Inspect возвращает сведения об одной зависимости по имени (без обращения к сети).
func (m *Manager) Inspect(name string) (Dependency, error) {
	s, ok := specByName(name)
	if !ok {
		return Dependency{}, fmt.Errorf("неизвестная зависимость: %q", name)
	}
	return m.inspectSpec(s), nil
}

// Resolve возвращает путь к используемому экземпляру зависимости согласно политике источника,
// БЕЗ вычисления версии — быстро, без запуска процессов. found=false — зависимость недоступна.
// Используется там, где нужен только путь (запуск yt-dlp), чтобы не тратить время на `--version`.
func (m *Manager) Resolve(name string) (string, bool) {
	s, ok := specByName(name)
	if !ok {
		return "", false
	}
	status, path, _, _ := m.resolveSource(s)
	if status == StatusMissing || path == "" {
		return "", false
	}
	return path, true
}

// resolveSource выбирает эффективный источник зависимости по политике (без версии):
// auto — управляемая, если есть, иначе системная; system — только системная; managed — только управляемая.
func (m *Manager) resolveSource(s spec) (status, path string, entry manifestEntry, hasEntry bool) {
	primary := s.primaryBinaryName()
	if primary == "" {
		return StatusMissing, "", entry, false
	}

	bin := filepath.Join(m.dir, primary)
	entry, hasEntry = loadManifest(m.manifestPath())[s.name]
	managedPresent := hasEntry && fileExists(bin)
	systemPath, systemErr := exec.LookPath(primary)
	systemPresent := systemErr == nil

	switch m.sourcePolicy(s.name) {
	case SourceSystem:
		if systemPresent {
			return StatusSystem, systemPath, entry, hasEntry
		}
	case SourceManaged:
		if managedPresent {
			return StatusManaged, bin, entry, hasEntry
		}
	default: // SourceAuto
		if managedPresent {
			return StatusManaged, bin, entry, hasEntry
		}
		if systemPresent {
			return StatusSystem, systemPath, entry, hasEntry
		}
	}
	return StatusMissing, "", entry, hasEntry
}

// inspectSpec определяет статус зависимости с учётом выбранной политики источника и
// дополняет его версией (для системной копии версия вычисляется запуском `--version`).
func (m *Manager) inspectSpec(s spec) Dependency {
	status, path, entry, _ := m.resolveSource(s)
	dep := Dependency{Name: s.name, Source: m.sourcePolicy(s.name), Status: status, Path: path}
	switch status {
	case StatusManaged:
		dep.Version = entry.Version
		dep.UpdatedAt = entry.UpdatedAt
	case StatusSystem:
		dep.Version = binaryVersion(s, path)
	}
	return dep
}

// CheckUpdates возвращает зависимости с заполненной свежей версией (обращается к сети).
// Для зависимостей с роллинг-релизом (ffmpeg) свежая версия не определяется. Неудача по
// одной зависимости не обнуляет результаты остальных; ошибка возвращается, только если
// не удалось узнать версию ни для одной из зависимостей.
func (m *Manager) CheckUpdates(ctx context.Context, channel string) ([]Dependency, error) {
	list := m.InspectAll()

	checked := false
	var firstErr error
	for i := range list {
		s, ok := specByName(list[i].Name)
		if !ok || !s.updates {
			continue
		}
		latest, err := m.latestVersion(ctx, s, channel)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", s.name, err)
			}
			continue
		}
		list[i].LatestVersion = latest
		list[i].UpdateAvailable = updateAvailable(s, list[i], latest)
		checked = true
	}
	if !checked && firstErr != nil {
		return nil, firstErr
	}
	return list, nil
}

// latestVersion возвращает версию последнего релиза зависимости (сеть).
func (m *Manager) latestVersion(ctx context.Context, s spec, channel string) (string, error) {
	repo, err := s.repo(runtime.GOOS, runtime.GOARCH, channel)
	if err != nil {
		return "", err
	}
	rel, err := fetchLatestRelease(ctx, repo)
	if err != nil {
		return "", err
	}
	return rel.TagName, nil
}

// Install скачивает зависимость по имени, проверяет SHA-256 и устанавливает.
// Возвращает итоговое состояние зависимости.
func (m *Manager) Install(ctx context.Context, name, channel string, onProgress ProgressFunc) (Dependency, error) {
	s, ok := specByName(name)
	if !ok {
		return Dependency{}, fmt.Errorf("неизвестная зависимость: %q", name)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.installSpec(ctx, s, channel, onProgress); err != nil {
		return Dependency{}, err
	}
	return m.inspectSpec(s), nil
}

// Remove удаляет управляемую копию зависимости по имени (системную не трогает).
func (m *Manager) Remove(name string) error {
	s, ok := specByName(name)
	if !ok {
		return fmt.Errorf("неизвестная зависимость: %q", name)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	return m.removeSpec(s)
}

// RemoveAll удаляет все управляемые (скачанные приложением) копии зависимостей,
// не затрагивая системные.
func (m *Manager) RemoveAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mf := loadManifest(m.manifestPath())
	for _, s := range specs {
		if err := m.removeOutputs(s); err != nil {
			return err
		}
		delete(mf, s.name)
	}
	return saveManifest(m.manifestPath(), mf)
}

// removeSpec удаляет управляемую копию зависимости и её запись в манифесте.
// Вызывается при удержанном m.mu.
func (m *Manager) removeSpec(s spec) error {
	if err := m.removeOutputs(s); err != nil {
		return err
	}
	mf := loadManifest(m.manifestPath())
	delete(mf, s.name)
	return saveManifest(m.manifestPath(), mf)
}

// removeOutputs удаляет все итоговые файлы зависимости из каталога.
func (m *Manager) removeOutputs(s spec) error {
	names, err := s.outputsFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := os.Remove(filepath.Join(m.dir, name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("удаление %s: %w", s.name, err)
		}
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// versionKey приводит версию к сравнимому виду через spec.versionKey (или оставляет как есть).
func versionKey(s spec, v string) string {
	if s.versionKey == nil {
		return v
	}
	return s.versionKey(v)
}

// updateAvailable сообщает, отличается ли свежая версия от установленной. Для отсутствующей
// зависимости это «установка», а не обновление → false. Работает и для системных копий.
func updateAvailable(s spec, dep Dependency, latest string) bool {
	if dep.Status == StatusMissing || dep.Version == "" || latest == "" {
		return false
	}
	return versionKey(s, dep.Version) != versionKey(s, latest)
}

// binaryVersion запускает `<bin> versionArgs` и нормализует версию через spec.version.
// Разные инструменты используют разные флаги (`--version`/`-version`), поэтому их задаёт spec.
func binaryVersion(s spec, path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	args := s.versionArgs
	if len(args) == 0 {
		args = []string{"--version"}
	}
	out, err := exec.CommandContext(ctx, path, args...).Output()
	if err != nil {
		return ""
	}
	return s.version(string(out))
}

func report(fn ProgressFunc, p Progress) {
	if fn != nil {
		fn(p)
	}
}
