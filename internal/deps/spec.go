package deps

import (
	"fmt"
	"runtime"
)

// archiveKind задаёт, как подготовить скачанный файл к установке.
type archiveKind int

const (
	archiveNone archiveKind = iota // скачанный файл сам является исполняемым
	archiveGzip                    // скачанный файл — gzip-сжатый исполняемый файл
	archiveZip                     // скачанный файл — zip-архив
)

// output — исполняемый файл, который даёт компонент. member — имя файла внутри архива
// (сравнивается по базовому имени, важно для zip); dest — итоговое имя в каталоге зависимостей.
type output struct {
	member string
	dest   string
}

// component — один скачиваемый файл зависимости и бинарники, которые он даёт.
type component struct {
	asset     string      // имя файла в релизе
	checksums string      // имя файла контрольных сумм; "" — взять SHA-256 из digest релиза
	archive   archiveKind // как извлекать
	outputs   []output    // извлекаемые файлы (обычно один; для zip может быть два)
}

// spec описывает, откуда и как получить зависимость. Благодаря этому установка/обновление/
// удаление работают единообразно для всех зависимостей.
type spec struct {
	name        string
	repo        func(goos, goarch, channel string) (string, error)
	components  func(goos, goarch string) ([]component, error)
	hash        func(text, asset string) (string, error) // разбор файла контрольных сумм
	version     func(raw string) string                  // нормализация версии из вывода `--version`
	versionArgs []string                                 // аргументы для запроса версии; пусто → `--version`
	versionKey  func(string) string                      // приведение версии к сравнимому виду; nil → как есть
	updates     bool                                     // определяется ли обновление по тегу релиза
}

// specs — реестр управляемых зависимостей в порядке отображения в интерфейсе.
var specs = []spec{ytdlpSpec, denoSpec, ffmpegSpec}

// specByName возвращает описание зависимости по имени.
func specByName(name string) (spec, bool) {
	for _, s := range specs {
		if s.name == name {
			return s, true
		}
	}
	return spec{}, false
}

// outputsFor возвращает имена всех итоговых файлов зависимости для платформы.
func (s spec) outputsFor(goos, goarch string) ([]string, error) {
	comps, err := s.components(goos, goarch)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, c := range comps {
		for _, o := range c.outputs {
			names = append(names, o.dest)
		}
	}
	return names, nil
}

// primaryBinary возвращает имя основного исполняемого файла (для поиска в PATH и версии).
func (s spec) primaryBinary(goos, goarch string) (string, error) {
	names, err := s.outputsFor(goos, goarch)
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", fmt.Errorf("зависимость %q: не заданы файлы", s.name)
	}
	return names[0], nil
}

// primaryBinaryName возвращает основной файл зависимости для текущей ОС (или "").
func (s spec) primaryBinaryName() string {
	name, err := s.primaryBinary(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return ""
	}
	return name
}
