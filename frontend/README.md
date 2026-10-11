# frontend

Фронтенд Loadika Desktop: **Vue 3 (Composition API) + TypeScript (strict)**, организован по FSD.

- Точка входа — `src/main.ts`; корневой компонент — `src/app/App.vue`.
- Слои: `src/app` → `pages` → `widgets` → `features` → `entities` → `shared`.
- Биндинги Go в `bindings/` **генерируются** — не редактировать вручную:
  `wails3 generate bindings -clean=true -ts -i`.
- Доступ к бэкенду — только через `src/shared/api` (прямой импорт биндингов в компонентах запрещён).

Команды:

```sh
npm install
npm run typecheck   # проверка типов
npm run build       # production-сборка (обновляет dist/, который встраивает Go)
```

Подробнее — [`../docs/03-frontend-fsd.md`](../docs/03-frontend-fsd.md).
