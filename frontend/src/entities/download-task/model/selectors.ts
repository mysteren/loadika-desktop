import type { Task } from "@/shared/api";

// Состояния задачи (синхронизировано с internal/download).
export const TASK_STATES = {
  queued: "queued",
  running: "running",
  paused: "paused",
  completed: "completed",
  failed: "failed",
  canceled: "canceled",
} as const;

// Задача выполняется прямо сейчас (можно поставить на паузу/отменить).
export function isRunning(task: Task): boolean {
  return task.state === TASK_STATES.running;
}

// Задача завершена (успех/ошибка/отмена) — финальное состояние.
export function isFinished(task: Task): boolean {
  return (
    task.state === TASK_STATES.completed ||
    task.state === TASK_STATES.failed ||
    task.state === TASK_STATES.canceled
  );
}

// Задача ещё не завершена (в очереди/выполняется/на паузе).
export function isPending(task: Task): boolean {
  return !isFinished(task);
}

// Доля выполнения (0..1) для индикатора; 0, если общий размер неизвестен.
export function progressRatio(task: Task): number {
  if (task.total <= 0) {
    return 0;
  }
  return Math.min(task.downloaded / task.total, 1);
}

// Человекочитаемая подпись состояния для интерфейса.
export function taskStateLabel(state: string): string {
  switch (state) {
    case TASK_STATES.queued:
      return "В очереди";
    case TASK_STATES.running:
      return "Загрузка";
    case TASK_STATES.paused:
      return "Пауза";
    case TASK_STATES.completed:
      return "Готово";
    case TASK_STATES.failed:
      return "Ошибка";
    case TASK_STATES.canceled:
      return "Отменено";
    default:
      return state;
  }
}
