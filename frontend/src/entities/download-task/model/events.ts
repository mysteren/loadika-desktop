import { onWailsEvent } from "@/shared/api";
import { EVENTS } from "@/shared/config/events";
import type { TaskLog, TaskProgress, TaskState } from "@/shared/config/events";

import { useDownloadTaskStore } from "./store";

// Подписка стора на события задач загрузки. Вызывать один раз в корне приложения
// (App.vue): подписки живут со скоупом и снимаются при его уничтожении (без утечек).
export function useDownloadTaskEvents(): void {
  const store = useDownloadTaskStore();
  onWailsEvent<TaskProgress>(EVENTS.taskProgress, (payload) => store.applyProgress(payload));
  onWailsEvent<TaskState>(EVENTS.taskState, (payload) => store.applyState(payload));
  onWailsEvent<TaskLog>(EVENTS.taskLog, (payload) => store.applyLog(payload));
}
