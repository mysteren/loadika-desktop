# Сборка и релиз — гайд

Практическое руководство: как собрать **Loadika Desktop** локально и как выпустить релиз.
Команды — через Wails v3 (`wails3`) и задачи проекта (`wails3 task …`).

> Библиотеки зависимостей (`yt-dlp`, `deno`, `ffmpeg`) в поставку **не встраиваются** — ставятся
> в рантайме из UI (см. `04-dependencies.md`).

## TL;DR

```sh
wails3 doctor                 # проверка окружения
wails3 dev                    # разработка (hot-reload)
wails3 task build             # бинарник текущей ОС  → bin/
wails3 task package           # пакеты текущей ОС    → bin/

# релиз: поднять версию (см. §3) → коммит → тег → пуш
git commit -am "release: vX.Y.Z" && git tag vX.Y.Z
git push && git push origin vX.Y.Z      # CI соберёт все ОС и опубликует Release
```

## 1. Как это устроено

- **Фронтенд** собирается `npm` (Vite) в `frontend/dist` и встраивается в Go-бинарник (`//go:embed`).
- **Иконка** — `build/appicon.png`; из неё генерируются `windows/icon.ico` и `darwin/icons.icns`
  (задача `common:generate:icons`); иконка также встроена в бинарник.
- **Задачи** — в `Taskfile.yml` (корень) и `build/**/Taskfile.yml`:
  `build` / `package` / `run` в корне диспетчеризуются на ОС (по `GOOS`), а `linux:*` / `windows:*` /
  `darwin:*` — платформенные.

## 2. Предварительные требования

- **Go** ≥ 1.25 (`go version`).
- **Node.js + npm** ≥ 24 (`npm -v`).
- **Wails v3 CLI** (`wails3 version`; проект собран под `v3.0.0-beta.27`).
- **[Task](https://taskfile.dev)** — нужен для вложенных `task:`-вызовов (устанавливается отдельно;
  `wails3 task` из репозитория Wails обычно доступен и без него).
- Проверка окружения: **`wails3 doctor`**.

По платформам:

| ОС | Дополнительно |
|----|---------------|
| Linux (сборка/запуск GUI) | `libgtk-4-dev libwebkitgtk-6.0-dev libsoup-3.0-dev build-essential pkg-config` (+ `squashfs-tools` для AppImage) |
| Windows | WebView2 Runtime (обычно уже есть); NSIS (`makensis`) — для установщика; C-компилятор не нужен |
| macOS | Xcode Command Line Tools (`xcode-select --install`); `lipo`, `codesign`, `hdiutil` — встроены |

## 3. Сборка (билд)

### 3.1 Разработка (hot-reload)

```sh
cd frontend && npm install && cd ..
wails3 dev
```

### 3.2 Production-сборка

```sh
wails3 task build     # → bin/loadika-desktop (Linux/macOS), bin/loadika-desktop.exe (Windows)
wails3 task run       # собрать и запустить
wails3 task package   # собрать и упаковать пакеты текущей ОС (см. §3.3)
```

`build` / `package` / `run` в корне выбирают платформенную задачу по `GOOS` (по умолчанию — ОС хоста).

### 3.3 Пакеты по ОС

```sh
# Linux: deb + rpm + AppImage + AUR (pkg.tar.zst) → bin/
wails3 task linux:package
ARCH=arm64 wails3 task linux:build        # конкретная архитектура

# Windows: установщик NSIS → bin/ (переменная FORMAT=msix — MSIX вместо NSIS)
wails3 task windows:package

# macOS: .app и .dmg (dmg — только на macOS)
wails3 task darwin:package:dmg            # собирает .app и .dmg
wails3 task darwin:build:universal        # универсальный бинарник (amd64 + arm64)
```

- GUI Linux требует **CGO**; если нет компилятора C, сборка идёт не на Linux или архитектура
  отличается — задача сама собирает через Docker (см. §3.4).
- `windows:build` по умолчанию — нативная кросс-сборка (`CGO_ENABLED=0`), работает из Linux/macOS.
- Нативная сборка macOS — **только на macOS** (иначе Docker).

### 3.4 Кросс-компиляция через Docker

```sh
wails3 task setup:docker     # один раз: соберёт образ wails-cross (~800 МБ)
```

После этого `linux:build`, `windows:build` (при `CGO_ENABLED=1`) и `darwin:build` сами используют
образ, если нативная сборка невозможна.

### 3.5 Проверки перед релизом

```sh
gofmt -l internal            # форматирование (должно быть пусто)
go build . ./internal/...    # сборка (НЕ `go build ./...` — см. §7)
go vet ./internal/...
go test ./internal/...
cd frontend && npm run typecheck && npm run build
```

## 4. Как выпустить релиз

Коротко: **версия в файлах → коммит → тег `vX.Y.Z` → `git push` тега → GitHub Actions собирает
все ОС и публикует GitHub Release.**

### 4.1 Поднять версию (единый номер, например `0.2.0`)

- `internal/version/version.go` → `var Version = "0.2.0"`
- `build/config.yml` → `info.version`
- `build/windows/info.json` → `fixed.file_version` и `info.0000.ProductVersion`
- `build/darwin/Info.plist` и `build/darwin/Info.dev.plist` → `CFBundleVersion` и
  `CFBundleShortVersionString`
- `build/linux/nfpm/nfpm.yaml` → `version`
- `build/windows/nsis/project.nsi` → `INFO_PRODUCTVERSION` (данные продукта и путь установки для NSIS)
- `build/windows/msix/template.xml` и `build/windows/msix/app_manifest.xml` → `Version` (для MSIX)

> Версию правим точечно, а не через `wails3 task common:update:build-assets`: эта команда
> пересобирает платформенные ассеты из `config.yml`, но **затирает ручные правки** в `build/**`
> (`nfpm.yaml`, `Info.plist` с `NSAppTransportSecurity`, кастомный `build/linux/Taskfile.yml`).

### 4.2 Тег и пуш

```sh
git commit -am "release: v0.2.0"
git tag v0.2.0
git push && git push origin v0.2.0
```

> ⚠️ Номер в теге (`v0.2.0`) и версия в файлах (`0.2.0`) должны совпадать: в приложении и в
> метаданных пакетов показывается значение из файлов, а имя релиза берётся из тега.

### 4.3 Что делает CI

Workflow [`.github/workflows/release.yml`](../.github/workflows/release.yml) запускается на пуш тега
`v*` (и вручную — «Run workflow»: тогда только собирает артефакты, без публикации):

| ОС | Раннер | Команда | Артефакты |
|----|--------|---------|-----------|
| Linux | `ubuntu-24.04` | `wails3 task linux:package` | `.deb`, `.rpm`, `.AppImage`, `.pkg.tar.zst` (AUR) |
| Windows | `windows-latest` | `windows:build` + zip, затем NSIS | портативный `.zip` и установщик NSIS `loadika-desktop-amd64-installer.exe` |
| macOS | `macos-latest` | `darwin:package:universal` + `create:dmg` | универсальный `.dmg` (Intel + Apple Silicon) |

После сборки job `release` создаёт GitHub Release с автоматическими заметками.

### 4.4 Где взять артефакты

Вкладка **Actions** (прогоны) и раздел **Releases** репозитория. Локально — каталог `bin/` после
`wails3 task package`.

## 5. Требования к системе (какой пакет под какую ОС)

| Формат | Что нужно пользователю |
|--------|------------------------|
| **AppImage** | системные **GTK4 + WebKitGTK 6.0** — Debian 13+, Ubuntu 24.04+, Linux Mint 22+, LMDE 7, Fedora 39+, Arch/Manjaro, openSUSE Tumbleweed. На старых системах (Debian ≤ 12, Ubuntu ≤ 22.04) WebKitGTK 6.0 нет — берите `deb`/`rpm` |
| **deb / rpm** | зависимости объявлены в пакете (`libgtk-4-1`, `libwebkitgtk-6.0-4`) |
| **Windows** | WebView2 (в 10/11 обычно есть); портативный `.zip` не требует установки |
| **macOS** | 12.0+ (universal-сборка) |

AppImage собирается на **системных** библиотеках (без бандлинга GTK) — см. §6.2.

## 6. Иконки, метаданные и AppImage

### 6.1 Иконки и метаданные

- Сменить данные продукта — в `build/config.yml` (`productName`, `productIdentifier`, `version`,
  `copyright`, …).
- **Установщик Windows (NSIS):** данные продукта и путь установки — в
  `build/windows/nsis/project.nsi` (`INFO_COMPANYNAME` / `INFO_PRODUCTNAME` → `C:\Program Files\Mysterentc\Loadika`);
  эти `!define` перебивают шаблонные дефолты в `wails_tools.nsh`.
- Сменить иконку: положить новый логотип в `frontend/public/logo.svg` и перегенерировать иконку
  приложения:
  ```sh
  cd tools/appicon && go run . -in ../../frontend/public/logo.svg -out ../../build/appicon.png
  wails3 task common:generate:icons    # обновит windows/icon.ico и darwin/icons.icns
  ```
  `tools/appicon` кладёт рядом `appicon-256.png` — её требует linuxdeploy для AppImage.
- **Иконка на Linux:** по умолчанию стек GTK4/WebKitGTK-6.0, где окно не несёт своей иконки —
  рабочий стол сопоставляет окно с `.desktop` по **app-id** (`ru.mysterentc.loadika-desktop`,
  `internal/app/app.go`). Имя `.desktop` и иконки в теме должно совпадать с этим id. Установить
  ярлык и иконку для разработки без пакета:
  ```sh
  wails3 task linux:install:user:desktop
  ```

### 6.2 AppImage (системные библиотеки)

AppImage собирается через `linuxdeploy` **без плагина GTK** и с `--exclude-library='*'` — GTK/WebKit
не пакуются, используются системные (образ ~5 МБ вместо ~80 МБ).

Причина: при бандлинге GTK собранный на Ubuntu 24.04 GTK 4.14 конфликтовал с системными модулями
на более новых системах (GTK 4.18) — при запуске сыпались `Failed to load module … undefined symbol …`
(не фатальные, но шумные). Требования к системе — в §5.

Смежно: при запуске `yt-dlp`/`ffmpeg`/`deno` окружение дочернего процесса очищается
(`internal/procenv`): внутри AppImage убираются `PYTHONHOME`/`PYTHONPATH`/`LD_LIBRARY_PATH`, которые
ломали Python-`yt-dlp` («Failed to import encodings module»).

## 7. Частые проблемы

- **`go build ./...` падает на `build/ios`** — используйте `go build . ./internal/...`
  (аналогично `gofmt -l internal`, а не `gofmt -l .`).
- **CI Linux: `Package gtk4/webkitgtk-6.0 not found`** — `go install wails3` компилируется с cgo и
  требует GTK4/WebKitGTK; dev-пакеты в CI ставятся **до** установки CLI.
- **CI Windows: установщик не собрался** — `makensis` не в `PATH`; NSIS добавляется в `GITHUB_PATH`.
  Портативный `.zip` собирается всегда (не зависит от NSIS).
- **CI macOS: нет `.dmg`** — команда `darwin:package` даёт только `.app`; нужен
  `darwin:package:universal` + `create:dmg`.
- **`go build` в CI: `pattern all:frontend/dist: no matching files found`** — в git нет `frontend/dist`
  (встраивается `//go:embed`); в job-проверках создаётся заглушка, реальную сборку делает job фронтенда.
- **AppImage: `Failed to import encodings module`** — следствие `PYTHONHOME` из AppRun; устранено
  очисткой окружения (`internal/procenv`). Обновляйтесь до версии с фиксом.
- **AppImage: `Failed to load module … undefined symbol …`** — конфликт бандленного GTK с системными
  модулями; устранено переходом на системные библиотеки (без бандлинга).

## 8. Подпись и нотаризация (когда понадобится)

По умолчанию артефакты **не подписаны**: macOS покажет Gatekeeper, Windows — SmartScreen.
Задачи готовы (нужны сертификаты, настраиваются через `wails3 setup`):

- Linux: `linux:sign:deb`, `linux:sign:rpm`, `linux:sign:packages` (PGP).
- Windows: `windows:sign`, `windows:sign:installer` (Authenticode).
- macOS: `darwin:sign`, `darwin:sign:notarize` (Developer ID + нотаризация).

## 9. Мини-шпаргалка

```sh
# подготовка (один раз)
wails3 doctor
wails3 task setup:docker            # только если нужна кросс-сборка

# разработка
wails3 dev

# сборка и пакеты текущей ОС
wails3 task build && wails3 task package

# Linux
wails3 task linux:package

# Windows (установщик + портативный zip — в CI; локально — NSIS в PATH)
wails3 task windows:package

# macOS (нативно — только на macOS)
wails3 task darwin:package:dmg

# релиз
# 1) поднять версию в файлах (§4.1)
# 2) git commit -am "release: vX.Y.Z"
# 3) git tag vX.Y.Z && git push && git push origin vX.Y.Z
```

## Ссылки

- Архитектура и статус: [`08-progress.md`](./08-progress.md), [`02-backend.md`](./02-backend.md).
- Зависимости: [`04-dependencies.md`](./04-dependencies.md).
- Лицензия: [`LICENSE`](../LICENSE) (MIT; та же указана в `build/linux/nfpm/nfpm.yaml`).
