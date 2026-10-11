# Loadika Desktop

[![CI](https://github.com/mysteren/loadika-desktop/actions/workflows/ci.yml/badge.svg)](https://github.com/mysteren/loadika-desktop/actions/workflows/ci.yml)
[![Release](https://github.com/mysteren/loadika-desktop/actions/workflows/release.yml/badge.svg)](https://github.com/mysteren/loadika-desktop/actions/workflows/release.yml)

Кроссплатформенное (Linux / Windows / macOS) десктопное приложение — удобная обёртка над
[`yt-dlp`](https://github.com/yt-dlp/yt-dlp): очередь загрузок с паузой/докачкой, пул прокси
с автоматическим переключением, выбор качества по фактическим форматам, библиотека «Видео»,
авторизация YouTube через cookies браузера и управление зависимостями.

## Скачать

Готовые сборки — в [Releases](https://github.com/mysteren/loadika-desktop/releases):

- **Linux** — `.deb`, `.rpm`, `.pkg.tar.zst` (Arch) и `.AppImage` (AppImage использует системные
  GTK4 + WebKitGTK 6.0 — Debian 13+, Ubuntu 24.04+, Fedora 39+, Arch/Manjaro, openSUSE Tumbleweed);
- **Windows** — портативный `.zip` и установщик NSIS (`.exe`);
- **macOS** — `.dmg` (universal: Intel + Apple Silicon).

Библиотеки `yt-dlp`, `deno` и `ffmpeg` в поставку не входят — приложение устанавливает их при
первом запуске («Настройки → Зависимости»). Подробности — в справке внутри приложения.

## Стек

- **Wails v3** (нативный webview) + **Go** — оболочка и бэкенд (один лёгкий бинарник).
- **Vue 3 + TypeScript (strict)** по FSD, **Vite**, **Pinia**, собственный UI на CSS-токенах
  + headless-примитивы `reka-ui`.

## Требования

- Go 1.25+
- Node.js 24+
- `wails3` (v3.0.0-beta.27) и системные зависимости Wails для вашей ОС

## Разработка

```sh
# фронтенд-зависимости (один раз)
cd frontend && npm install && cd ..

# запуск в режиме разработки (hot-reload фронтенда и бэкенда)
wails3 dev

# production-сборка
wails3 build
```

Проверки:

```sh
cd frontend && npm run typecheck   # типы TS/Vue
go build . ./internal/...          # сборка бэкенда
```

## Структура

```
main.go              # точка входа: встраивание frontend/dist, запуск приложения
internal/            # Go: app, services, version (далее — deps, ytdlp, proxy, download, config, ...)
frontend/            # Vue 3 + FSD (src/app, pages, widgets, features, entities, shared)
  bindings/          # сгенерированные TS-биндинги Go → frontend (не редактировать вручную)
docs/                # архитектура, стек, соглашения, план разработки
.agents/skills/      # скилы для агентов (конвенции проекта)
```

Биндинги перегенерируются командой:

```sh
wails3 generate bindings -clean=true -ts -i
```

## Документация

Начните с [`docs/README.md`](./docs/README.md).

## Соглашения

Комментарии и документация — на русском; идентификаторы — английские.
Подробности — в [`docs/06-conventions.md`](./docs/06-conventions.md).

## Релиз

Порядок сборки и выпуска версий — в [`docs/BUILD-AND-RELEASE.md`](./docs/BUILD-AND-RELEASE.md).

## Лицензия

[MIT](./LICENSE).
