# Loadika Desktop

Кроссплатформенное (Linux / Windows / macOS) десктопное приложение — удобная обёртка над
[`yt-dlp`](https://github.com/yt-dlp/yt-dlp): пресеты настроек загрузки, пул прокси
с автоматическим переключением, выбор папки загрузки и управление зависимостями.

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
