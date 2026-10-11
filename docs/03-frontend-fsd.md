# 03. Архитектура фронтенда (FSD)

Используем **Feature-Sliced Design** как ориентир, но без догматизма: слои и правила импортов
соблюдаем, а внутренние сегменты добавляем по необходимости (не создаём пустые папки «на будущее»).

## Слои

Снизу вверх (импорт разрешён только «вниз», т.е. слой знает о слоях ниже, но не выше):

```
app → pages → widgets → features → entities → shared
```

| Слой | Назначение | Примеры у нас |
|------|-----------|---------------|
| `app` | Инициализация приложения, провайдеры, глобальные стили, корневой layout | `providers/`, `styles/`, `App.vue`, `main.ts` |
| `pages` | Экраны, собирают widgets/features | `downloads`, `proxies`, `settings`, `history` |
| `widgets` | Крупные составные блоки | `sidebar`, `task-list`, `add-download-panel`, `deps-status` |
| `features` | Пользовательские сценарии (действия) | `add-download`, `manage-proxy-pool`, `select-download-dir`, `update-dependencies`, `toggle-theme` |
| `entities` | Бизнес-сущности | `download-task`, `proxy`, `dependency`, `download-preset` |
| `shared` | Переиспользуемое, не знает о бизнесе | `ui/`, `lib/`, `api/`, `config/`, `assets/` |

## Структура каталогов

```
frontend/src/
  app/
    App.vue
    main.ts
    providers/          # pinia, theme, wails-bindings init
    styles/             # tokens.css, reset.css, base.css
  pages/
    downloads/          # ui/ + index.ts
    proxies/
    settings/
    history/
  widgets/
    sidebar/
    task-list/
    add-download-panel/
    deps-status/
  features/
    add-download/
    manage-proxy-pool/
    select-download-dir/
    update-dependencies/
  entities/
    download-task/
      model/            # типы, стор, селекторы
      ui/               # карточка/строка задачи
      api/              # вызовы биндингов, привязанные к сущности
      index.ts
    proxy/
    dependency/
    download-preset/
  shared/
    ui/                 # Button, Input, Select, Modal, Tooltip, ProgressBar...
    lib/                # утилиты: форматирование размера/скорости/времени, debounce
    api/                # обёртки над generated bindings + подписки на события
    config/             # константы, имена событий
    assets/             # иконки (инлайн SVG), шрифты
```

## Правила

1. **Публичный API слайса** — только через `index.ts`. Импорт «внутрь» слайса из другого слайса
   запрещён (`entities/proxy/model` нельзя импортировать напрямую — только `entities/proxy`).
2. **Направление импортов** — строго вниз по слоям. Ссылки вверх — запрещены.
3. **`shared` не знает о бизнесе** — никаких импортов из `entities` и выше.
4. **Никаких «свалок» в `shared/lib`.** Утилита попадает туда, только если реально
   переиспользуется в нескольких слайсах.
5. **Компонент ≠ фича.** Сценарий с действием и побочными эффектами — `feature`;
   чисто презентационный блок — `shared/ui`.

## Доступ к бэкенду

- Сгенерированные биндинги (`frontend/bindings/...`) напрямую в компонентах **не используем**.
- Всё оборачивается в `shared/api` (вызовы Go-сервисов) и в `entities/*/api`/`features/*`
  (подписки на события).
- Обёртки обеспечивают: единый стиль ошибок, типизацию, централизованную отписку от событий.

## Состояние

- **Локальное** (внутри компонента/фичи) — `ref`/`computed`/`shallowRef`.
- **Разделяемое** — Pinia-стор, размещённый по FSD: стор сущности — в `entities/*/model`,
  стор-координатор сценария — в `features/*/model`.
- Сторы не должны дублировать состояние друг друга; источник истины — один.

## Производительность (обязательно)

- Большие списки (задачи, прокси) — `shallowRef`/`markRaw`; при необходимости виртуализация.
- События прогресса приходят уже throttled с бэкенда; в UI дополнительно избегаем тяжёлых
  пересчётов на каждое обновление (вычисляем производные через `computed`).
- Подписки на события Wails — обязательно с отпиской в `onScopeDispose`/`onUnmounted`.

## Стили

- CSS-переменные (токены) в `app/styles/tokens.css`: цвета (светлая/тёмная тема), отступы,
  радиусы, тени, типографика.
- Компоненты — scoped CSS, опирающийся на токены. Никаких «магических» значений в разметке.
- Плотный, предсказуемый desktop-layout (высота строк, размеры контролов рассчитаны на мышь).
