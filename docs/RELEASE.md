# Релиз — как собрать и опубликовать

Коротко: **версия правится в файлах → коммит → тег `vX.Y.Z` → `git push` тега → GitHub Actions
собирает пакеты под все ОС и публикует GitHub Release.**

Возможна и **локальная сборка** (Linux — на этой машине проверено; Windows/macOS — через CI или Docker).

## Как устроена публикация (CI)

Workflow [`.github/workflows/release.yml`](../.github/workflows/release.yml) запускается на пуш
тега вида `v*` (и вручную через «Run workflow» — тогда только собирает артефакты, без публикации).

Матрица — на «родных» раннерах:

| ОС | Раннер | Команда | Артефакты |
|----|--------|---------|-----------|
| Linux | `ubuntu-24.04` | `wails3 task linux:package` | `.deb`, `.rpm`, `.AppImage`, `.pkg.tar.zst` (AUR) |
| Windows | `windows-latest` | `windows:build` + zip, затем NSIS | портативный `.zip` и установщик NSIS `loadika-desktop-amd64-installer.exe` |
| macOS | `macos-latest` | `darwin:package:universal` + `create:dmg` | универсальный `.dmg` (Intel + Apple Silicon) |

После успешной сборки job `release` скачивает артефакты и создаёт GitHub Release с заметками
(`softprops/action-gh-release`, `generate_release_notes: true`).

## Требования к системе

- **AppImage** (Linux) собирается на **системных** библиотеках: нужны **GTK4 + WebKitGTK 6.0** —
  Debian 13+, Ubuntu 24.04+, Linux Mint 22+, LMDE 7, Fedora 39+, Arch/Manjaro, openSUSE Tumbleweed.
  На более старых системах (Debian ≤ 12, Ubuntu ≤ 22.04) берите `deb`/`rpm` (или ставьте системные
  библиотеки). Бандлинга GTK в AppImage нет — так убраны ошибки загрузки системных модулей из-за
  рассинхрона версий.
- **deb / rpm** — зависимости объявлены в пакете (`libgtk-4-1`, `libwebkitgtk-6.0-4`).
- **Windows** — WebView2 (в Windows 10/11 обычно уже есть); портативный `.zip` не требует установки.
- **macOS** — 12.0+ (universal-сборка: Intel и Apple Silicon).
- Библиотеки `yt-dlp` / `deno` / `ffmpeg` в поставку не входят — устанавливаются в рантайме.

## Шаги релиза

1. **Поднять версию** во всех местах (единый номер, например `0.2.0`):
   - `internal/version/version.go` → `var Version = "0.2.0"`
   - `build/config.yml` → `info.version`
   - `build/windows/info.json` → `fixed.file_version` и `info.0000.ProductVersion`
   - `build/darwin/Info.plist` и `build/darwin/Info.dev.plist` → `CFBundleVersion` и
     `CFBundleShortVersionString`
   - `build/linux/nfpm/nfpm.yaml` → `version`
2. **Проверить, что собирается**: `wails3 task linux:package` (Linux) — или локальные проверки
   (см. `08-progress.md`).
3. **Закоммитить и протегировать**:
   ```sh
   git commit -am "release: v0.2.0"
   git tag v0.2.0
   git push && git push origin v0.2.0
   ```
4. **Дождаться CI** → во вкладке Actions и в разделе Releases появится релиз с файлами.

> ⚠️ Номер в теге (`v0.2.0`) и версия в файлах (`0.2.0`) должны совпадать — в приложении и в
> метаданных пакетов показывается значение из файлов, а имя релиза берётся из тега.

## Почему версия правится вручную

Штатная команда `wails3 task common:update:build-assets` пересобирает платформенные ассеты из
`build/config.yml`, но **затирает ручные правки** в `build/**` (например, `nfpm.yaml`, `Info.plist`
с `NSAppTransportSecurity`, кастомный `build/linux/Taskfile.yml`). Поэтому метаданные правятся точечно,
а `update:build-assets` не используется.

## Локальная сборка (для проверки)

```sh
# Linux (стек GTK4 + WebKitGTK 6.0; нужны libgtk-4-dev, libwebkitgtk-6.0-dev, libsoup-3.0-dev)
wails3 task linux:build
wails3 task linux:package      # deb/rpm/AppImage/AUR → bin/

# Windows — можно кросс-собрать (CGO_ENABLED=0); для установщика нужен makensis (NSIS)
wails3 task windows:build
wails3 task windows:package

# macOS — только на macOS (или через Docker: wails3 task setup:docker)
wails3 task darwin:build:universal
wails3 task darwin:package:dmg
```

Подробнее про предусловия по ОС — в [`BUILD.md`](./BUILD.md).

## Подпись (когда понадобится)

По умолчанию артефакты **не подписаны**: macOS покажет предупреждение Gatekeeper, Windows —
SmartScreen. Задачи для подписи уже есть (нужны сертификаты, настраиваются через `wails3 setup`):

- Linux: `linux:sign:packages` (PGP).
- Windows: `windows:sign:installer` (Authenticode).
- macOS: `darwin:sign:notarize` (Developer ID + нотаризация).

## Зависимости в инсталляторе

`yt-dlp`, `deno`, `ffmpeg`/`ffprobe` в пакеты **не встраиваются** — ставятся в рантайме из UI
(«Настройки → Зависимости»). См. `04-dependencies.md`.

## Лицензия

Проект распространяется под MIT (файл [`LICENSE`](../LICENSE)); та же лицензия указана в
`build/linux/nfpm/nfpm.yaml`. При смене лицензии — обновите оба места.
