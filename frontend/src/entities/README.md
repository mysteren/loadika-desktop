# entities

Бизнес-сущности: `download-task`, `proxy`, `dependency`, `download-preset`.
Каждая сущность — слайс с публичным API через `index.ts` (`model/`, `ui/`, `api/` по необходимости).

Реализовано: `dependency` (Pinia-стор + селекторы). Остальные слайсы наполняются по мере
развития (см. `docs/03-frontend-fsd.md`).
