# Сборка проекта — шпаргалка

Справочник-подсказка: как собрать и упаковать **Loadika Desktop** под Linux, Windows, macOS.
Все команды — через Wails v3 (`wails3`) и задачи проекта (`wails3 task …`). Библиотека
зависимостей (`yt-dlp`, `deno`, `ffmpeg`) в инсталлятор **не встраивается** — она ставится
в рантайме (см. `04-dependencies.md`).

> Краткая версия для повседневной работы — в `08-progress.md` («Команды»). Этот документ
> подробнее: предусловия по ОС, кросс-компиляция и упаковка.

## Как это устроено

- **Фронтенд** собирается `npm` (Vite) в `frontend/dist` и встраивается в Go-бинарник (`//go:embed`).
- **Иконка приложения** — `build/appicon.png`; из неё генерируются `build/windows/icon.ico` и
  `build/darwin/icons.icns` (задача `common:generate:icons`). Иконка также встроена в бинарник.
- **Задачи** описаны в `Taskfile.yml` (корень) и `build/**/Taskfile.yml`:
  - `build` / `package` / `run` в корне — диспетчеризуются на ОС (переменная `GOOS`);
  - `linux:*`, `windows:*`, `darwin:*` — платформенные сборка/упаковка.

## Общие предусловия

- **Go** ≥ 1.25 (`go version`).
- **Node.js + npm** (`npm -v`) — сборка фронтенда.
- **Wails v3 CLI** (`wails3 version`).
- **[Task](https://taskfile.dev)** — часть сценариев (`wails3 dev`, вложенные `task:`-вызовы)
  использует исполняемый файл `task`. Установка: https://taskfile.dev/installation/.
- Проверка окружения: `wails3 doctor`.

## Сборка «для своей ОС» (обычный путь)

```sh
wails3 task build     # собрать бинарник текущей ОС
wails3 task package   # собрать и упаковать пакет(ы) текущей ОС
wails3 task run       # собрать и запустить
wails3 dev            # режим разработки с hot-reload (нужны системные зависимости ОС)
```

`build`/`package`/`run` в корне выбирают платформенную задачу по `GOOS` (по умолчанию — ОС хоста).

## Предусловия по платформам

### Linux (для запуска и сборки GUI)

- Пакеты сборки: `libwebkit2gtk-4.1-dev`, `libgtk-3-dev`, `libsoup-3.0-dev`, `pkg-config`,
  `build-essential` (Debian/Ubuntu) — аналогично `webkit2gtk4.1-devel gtk3-devel libsoup3-devel`
  (Fedora) и т.п.
- Для упаковки: `nfpm` используется внутри `wails3 tool package` (deb/rpm); AppImage/AUR —
  соответствующими задачами.

### Windows

- **WebView2 Runtime** (обычно уже стоит в Windows 10/11).
- Компилятор C **не нужен** (по умолчанию `CGO_ENABLED=0`).
- Для установщика — **NSIS** (`makensis` в `PATH`); для MSIX — `wails3 task install:msix:tools`
  (и `makeappx`/`signtool` из Windows SDK).

### macOS

- **Xcode Command Line Tools** (`xcode-select --install`).
- Встроенные инструменты: `lipo`, `codesign`, `hdiutil`.
- Минимальная версия — задана в задачах (`MACOSX_DEPLOYMENT_TARGET=12.0`).

## Сборка под каждую ОС

### Linux

```sh
wails3 task linux:build                 # amd64 хоста; при наличии CC — нативно, иначе в Docker
ARCH=arm64 wails3 task linux:build      # конкретная архитектура
wails3 task linux:package               # AppImage + deb + rpm + AUR (в bin/)
```

- Linux GUI требует **CGO**. Если сборка идёт не на Linux, нет компилятора C или архитектура
  отличается — задача сама собирает через Docker (образ `wails-cross`, см. ниже).

### Windows

```sh
wails3 task windows:build               # кросс-компиляция нативно (CGO_ENABLED=0)
ARCH=arm64 wails3 task windows:build    # arm64
wails3 task windows:package             # установщик NSIS (по умолчанию)
FORMAT=msix wails3 task windows:package # MSIX вместо NSIS
```

- `windows:build` по умолчанию **нативная кросс-сборка** (`CGO_ENABLED=0`) — можно собирать
  из Linux/macOS. При `CGO_ENABLED=1` автоматически используется Docker (Zig).

### macOS

```sh
wails3 task darwin:build                # нативно (только на macOS) или через Docker иначе
wails3 task darwin:build:universal      # универсальный бинарник (amd64 + arm64)
wails3 task darwin:package              # собрать .app-бандл
wails3 task darwin:package:dmg          # собрать .dmg (задача macOS-only)
```

- Нативная сборка macOS возможна **только на macOS**; с других ОС — через Docker (`build:docker`).

## Кросс-компиляция через Docker

Один раз подготовить образ:

```sh
wails3 task setup:docker     # соберёт образ wails-cross (загрузка ~800 МБ)
```

Дальше задачи `linux:build`, `windows:build` (при CGO_ENABLED=1), `darwin:build` сами используют
этот образ, если нативная сборка невозможна.

## Иконки и метаданные

- Изменить иконку: положить новый `build/appicon.png` и выполнить
  `wails3 task common:generate:icons` (обновит `.ico`/`.icns`).
  Иконка из логотипа генерируется инструментом `tools/appicon` (см. `08-progress.md`).
- Изменить данные продукта (`build/config.yml`: `productName`, `productIdentifier`, `version`,
  `copyright`, …) и пересобрать метаданные платформ:

  ```sh
  wails3 task common:update:build-assets
  ```

  Это обновит `Info.plist`, `windows/info.json` и пр. **Версию для релиза** правьте здесь же.

### Иконка на Linux

На Linux по умолчанию собирается стек **GTK4/WebKitGTK-6.0**, где окно **не несёт собственной
иконки** (`gtk_window_set_icon` в этом стеке нет). Рабочий стол подбирает иконку по привязке окна
к `.desktop`-файлу — по **app-id** приложения. App-id задан в коде:
`ru.mysterentc.loadika-desktop` (`internal/app/app.go`, `LinuxOptions.ApplicationID`).
Поэтому имя `.desktop` и иконки в теме должно совпадать с этим id:

- `build/linux/ru.mysterentc.loadika-desktop.desktop` с `Icon=ru.mysterentc.loadika-desktop`;
- иконка в теме как `ru.mysterentc.loadika-desktop.png` (так её ставит `build/linux/nfpm/nfpm.yaml`).

**Установить локально (для dev, без пакета)** — одной командой (работает и на Wayland, и на X11):

```sh
wails3 task linux:install:user:desktop
```

Задача пишет `~/.local/share/applications/ru.mysterentc.loadika-desktop.desktop` (с `Exec`
на `bin/loadika-desktop`) и кладёт иконку в `~/.local/share/icons/hicolor/…/apps/`. Она же
выполняется автоматически после `wails3 task linux:build`.

> Для обычной работы пересоберите приложение (`wails3 task linux:build`). При сборке
> с `-tags gtk3` (устаревший WebKit2GTK-4.1) работал бы и `gtk_window_set_icon` из `Options.Icon`,
> но по умолчанию используется GTK4, поэтому связка идёт через `.desktop`.

## AppImage и внешние инструменты

AppImage (linuxdeploy) выставляет окружение для бандла GTK/WebKit, в т.ч. `PYTHONHOME=$APPDIR/usr`,
`PYTHONPATH` и `LD_LIBRARY_PATH=$APPDIR/usr/lib`. Своему процессу они нужны, но сторонним
инструментам вредят: с чужим `PYTHONHOME` системный Python (которым запускается `yt-dlp`) не находит
стандартную библиотеку — «Fatal Python error: Failed to import encodings module».

Поэтому при запуске `yt-dlp`/`ffmpeg`/`deno` окружение дочернего процесса очищается
(`internal/procenv`: внутри AppImage, т.е. при заданном `APPDIR`, убираются
`PYTHONHOME`/`PYTHONPATH`/`PYTHONSTARTUP`/`LD_LIBRARY_PATH`). Вне AppImage окружение не меняется.

## Артефакты (каталог `bin/`)

- Linux: `bin/loadika-desktop`, а также пакеты из `linux:package` (AppImage/deb/rpm/AUR).
- Windows: `bin/loadika-desktop.exe`, установщик NSIS (или `.msix` при `FORMAT=msix`).
- macOS: `bin/loadika-desktop.app`, `bin/loadika-desktop.dmg`.

Точные имена файлов могут отличаться — смотрите вывод задачи.

## Подпись и нотаризация (когда понадобится)

Готовые задачи (требуют сертификатов/ключей, настраиваются через `wails3 setup`):

- Linux: `linux:sign:deb`, `linux:sign:rpm`, `linux:sign:packages` (PGP).
- Windows: `windows:sign`, `windows:sign:installer` (Authenticode).
- macOS: `darwin:sign`, `darwin:sign:notarize` (Developer ID + нотаризация).

## CI (рекомендация)

- Собирайте на «родных» раннерах: `ubuntu-latest` (Linux), `windows-latest` (Windows/NSIS),
  `macos-latest` (macOS/dmg/нотаризация).
- Для Linux/Windows допустима кросс-сборка из Linux-раннера через `wails3 task setup:docker`.
- Перед сборкой: `wails3 task common:update:build-assets` (если менялся `config.yml`).

## Ограничения текущего окружения (важно для этой машины)

- `wails3 dev`/GUI-сборка **не запускались**: нет `webkit2gtk-4.1` и `task`.
- `go build ./...` спотыкается о `build/ios` (шаблон Wails с build-тегами) — используйте
  `go build . ./internal/...`. Аналогично `gofmt -l internal` (а не `gofmt -l .`).
- Прикладная валидация здесь: `gofmt -l internal`, `go build . ./internal/...`,
  `go vet ./internal/...`, `go test ./internal/...`,
  `cd frontend && npm run typecheck && npm run build`.

## Мини-шпаргалка

```sh
# подготовка (один раз)
wails3 doctor
wails3 task setup:docker        # только если нужна кросс-сборка

# Linux
wails3 task linux:build   && wails3 task linux:package

# Windows (можно с Linux через Docker при CGO_ENABLED=1)
wails3 task windows:build && wails3 task windows:package

# macOS (нативно — только на macOS)
wails3 task darwin:build:universal && wails3 task darwin:package:dmg
```
