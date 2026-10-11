// Имена событий Go → Frontend. Держать в синхроне с пакетом internal/events (Go).
// Типы payload'ов берём из сгенерированных биндингов, имена объявляем здесь явно.
import type {
  DepProgress,
  DepState,
  ProxyHealth,
  TaskLog,
  TaskProgress,
  TaskState,
} from "@bindings/loadika-desktop/internal/events";

export type { DepProgress, DepState, ProxyHealth, TaskLog, TaskProgress, TaskState };

export const EVENTS = {
  depProgress: "dep:progress",
  depState: "dep:state",
  taskProgress: "task:progress",
  taskState: "task:state",
  taskLog: "task:log",
  proxyHealth: "proxy:health",
} as const;
