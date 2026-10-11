import { onWailsEvent } from "@/shared/api";
import { EVENTS } from "@/shared/config/events";
import type { DepProgress, DepState } from "@/shared/config/events";

import { useDependencyStore } from "./store";

// Подписка стора на события зависимостей. Вызывать один раз в корне приложения
// (App.vue): подписки живут вместе со скоупом и снимаются при его уничтожении (без утечек).
export function useDependencyEvents(): void {
  const store = useDependencyStore();
  onWailsEvent<DepProgress>(EVENTS.depProgress, (payload) => store.setProgress(payload));
  onWailsEvent<DepState>(EVENTS.depState, (payload) => store.applyState(payload));
}
