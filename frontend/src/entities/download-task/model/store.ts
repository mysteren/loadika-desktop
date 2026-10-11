// Стор сущности `download-task` — единый источник истины о задачах загрузки.
// UI не обращается к бэкенду напрямую: только через этот стор поверх `shared/api`.
// Список храним «свежие сверху»; прогресс и состояние приходят событиями task:*.
import { defineStore } from "pinia";
import { computed, ref, shallowRef } from "vue";

import {
  cancelDownload as apiCancel,
  createDownload as apiCreate,
  getDownloadTasks as apiGetTasks,
  getTaskJournal as apiGetJournal,
  pauseDownload as apiPause,
  resumeDownload as apiResume,
} from "@/shared/api";
import type { Task, TaskLogLine } from "@/shared/api";
import type { DownloadMeta } from "@/shared/api";
import type { TaskLog, TaskProgress, TaskState } from "@/shared/config/events";

import { TASK_STATES, isRunning } from "./selectors";

export const useDownloadTaskStore = defineStore("download-task", () => {
  // Список задач (свежие сверху). shallowRef — список лёгкий, но обновляем его целиком.
  const tasks = shallowRef<Task[]>([]);
  // Журналы задач (taskId → строки). Растут по мере поступления событий task:log.
  const journals = ref<Record<string, TaskLogLine[]>>({});
  const isLoading = ref(false);

  // Число задач по состояниям (для шапки списка загрузок).
  const runningCount = computed(() => tasks.value.filter(isRunning).length);
  const queuedCount = computed(
    () => tasks.value.filter((t) => t.state === TASK_STATES.queued).length,
  );
  const pausedCount = computed(
    () => tasks.value.filter((t) => t.state === TASK_STATES.paused).length,
  );

  // Вставляет задачу или заменяет существующую по id.
  function upsert(task: Task): void {
    const index = tasks.value.findIndex((t) => t.id === task.id);
    if (index === -1) {
      tasks.value = [task, ...tasks.value];
      return;
    }
    const next = tasks.value.slice();
    next[index] = task;
    tasks.value = next;
  }

  // Перечитывает список задач с бэкенда (свежие сверху).
  async function load(): Promise<void> {
    isLoading.value = true;
    try {
      const loaded = await apiGetTasks();
      tasks.value = [...loaded].reverse();
    } finally {
      isLoading.value = false;
    }
  }

  // Создаёт и запускает задачу. Ошибку пробрасываем — её показывает вызывающий компонент.
  async function create(
    url: string,
    format: string,
    useProxy: boolean,
    meta: DownloadMeta,
  ): Promise<void> {
    upsert(await apiCreate(url, format, useProxy, meta));
  }

  // Отменяет задачу; итоговое состояние придёт событием task:state.
  async function cancel(id: string): Promise<void> {
    await apiCancel(id);
  }

  // Ставит задачу на паузу (недокачанный файл сохраняется).
  async function pause(id: string): Promise<void> {
    await apiPause(id);
  }

  // Возвращает задачу с паузы (докачка).
  async function resume(id: string): Promise<void> {
    await apiResume(id);
  }

  // Применяет событие прогресса task:progress.
  function applyProgress(p: TaskProgress): void {
    const index = tasks.value.findIndex((t) => t.id === p.taskId);
    const current = tasks.value[index];
    if (current === undefined) {
      return;
    }
    const next = tasks.value.slice();
    next[index] = { ...current, downloaded: p.downloaded, total: p.total, speed: p.speed, eta: p.eta };
    tasks.value = next;
  }

  // Применяет событие смены состояния task:state.
  function applyState(s: TaskState): void {
    const index = tasks.value.findIndex((t) => t.id === s.taskId);
    const current = tasks.value[index];
    if (current === undefined) {
      return;
    }
    const next = tasks.value.slice();
    next[index] = { ...current, state: s.state, error: s.error, status: s.status };
    tasks.value = next;
  }

  // Применяет событие журнала task:log.
  function applyLog(p: TaskLog): void {
    const line: TaskLogLine = { time: p.time, level: p.level, message: p.message };
    journals.value = {
      ...journals.value,
      [p.taskId]: [...(journals.value[p.taskId] ?? []), line],
    };
  }

  // Возвращает журнал задачи (из памяти).
  function journalFor(id: string): TaskLogLine[] {
    return journals.value[id] ?? [];
  }

  // Загружает журнал задачи с бэкенда (если в памяти пусто — например, после перезапуска).
  async function loadJournal(id: string): Promise<void> {
    const loaded = await apiGetJournal(id);
    journals.value = { ...journals.value, [id]: loaded };
  }

  return {
    tasks,
    journals,
    isLoading,
    runningCount,
    queuedCount,
    pausedCount,
    load,
    create,
    cancel,
    pause,
    resume,
    applyProgress,
    applyState,
    applyLog,
    journalFor,
    loadJournal,
  };
});
