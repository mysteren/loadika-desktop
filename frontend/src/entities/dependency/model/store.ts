// Стор сущности `dependency` — единый источник истины о внешних зависимостях.
// UI-слои (widgets/features/pages) не обращаются к бэкенду напрямую: только через этот стор,
// который, в свою очередь, опирается на обёртки `shared/api`.
import { defineStore } from "pinia";
import { computed, ref } from "vue";

import {
  checkUpdates as apiCheckUpdates,
  getDependencies as apiGetDependencies,
  installDependency as apiInstall,
  removeAllManagedDependencies as apiRemoveAllManaged,
  removeDependency as apiRemove,
  setDependencySource as apiSetSource,
} from "@/shared/api";
import type { Dependency } from "@/shared/api";
import type { DepProgress, DepState } from "@/shared/config/events";

import { isUpdateAvailable } from "./selectors";

// Приводит неизвестную ошибку бэкенда к тексту, пригодному для показа пользователю.
function toMessage(err: unknown): string {
  if (err instanceof Error) {
    return err.message;
  }
  return String(err);
}

export const useDependencyStore = defineStore("dependency", () => {
  // Список зависимостей — единый источник для всех виджетов.
  const items = ref<Dependency[]>([]);

  // Идёт загрузка списка или проверка обновлений — блокирует кнопки в UI.
  const isLoading = ref(false);

  // Имя зависимости, для которой сейчас выполняется установка/удаление.
  const busyName = ref<string | null>(null);

  // Идёт полная очистка управляемых зависимостей.
  const isClearing = ref(false);

  // Приходящий с бэкенда прогресс установки (для текущей операции).
  const progress = ref<DepProgress | null>(null);

  // Последняя ошибка операции (для показа пользователю), иначе null.
  const error = ref<string | null>(null);

  // Есть ли хотя бы одна зависимость с доступным обновлением.
  const hasUpdates = computed(() => items.value.some(isUpdateAvailable));

  // Заменяет запись зависимости по имени, добавляя её, если её ещё не было в списке.
  function replaceItem(updated: Dependency): void {
    const index = items.value.findIndex((dep) => dep.name === updated.name);
    if (index === -1) {
      items.value = [...items.value, updated];
      return;
    }
    const next = items.value.slice();
    next[index] = updated;
    items.value = next;
  }

  // Перечитывает список с бэкенда, сохраняя ранее полученный результат проверки обновлений,
  // если установленная версия не изменилась. Иначе итог проверки терялся при навигации,
  // смене источника и т.п. — и казалось, что проверка «не работает».
  async function loadItems(): Promise<void> {
    const loaded = await apiGetDependencies();
    const previous = new Map(items.value.map((dep) => [dep.name, dep]));
    items.value = loaded.map((dep) => {
      const old = previous.get(dep.name);
      if (old !== undefined && old.version === dep.version && dep.latestVersion === "") {
        return { ...dep, latestVersion: old.latestVersion, updateAvailable: old.updateAvailable };
      }
      return dep;
    });
  }

  // Загружает список зависимостей с локальным статусом (без обращения к сети).
  async function refresh(): Promise<void> {
    isLoading.value = true;
    error.value = null;
    try {
      await loadItems();
    } catch (err) {
      error.value = toMessage(err);
    } finally {
      isLoading.value = false;
    }
  }

  // Проверяет свежие версии всех зависимостей (обращается к сети).
  async function checkUpdates(): Promise<void> {
    isLoading.value = true;
    error.value = null;
    try {
      items.value = await apiCheckUpdates();
    } catch (err) {
      error.value = toMessage(err);
    } finally {
      isLoading.value = false;
    }
  }

  // Устанавливает или обновляет зависимость и обновляет её запись в списке.
  async function install(name: string): Promise<void> {
    busyName.value = name;
    error.value = null;
    progress.value = null;
    try {
      replaceItem(await apiInstall(name));
    } catch (err) {
      error.value = toMessage(err);
    } finally {
      busyName.value = null;
      progress.value = null;
    }
  }

  // Удаляет управляемую копию зависимости и перечитывает состояние: после удаления
  // роли может перейти к системной копии.
  async function remove(name: string): Promise<void> {
    busyName.value = name;
    error.value = null;
    try {
      await apiRemove(name);
      await loadItems();
    } catch (err) {
      error.value = toMessage(err);
    } finally {
      busyName.value = null;
    }
  }

  // Удаляет все управляемые (скачанные приложением) зависимости, не трогая системные.
  async function removeManaged(): Promise<void> {
    isClearing.value = true;
    error.value = null;
    try {
      await apiRemoveAllManaged();
      await loadItems();
    } catch (err) {
      error.value = toMessage(err);
    } finally {
      isClearing.value = false;
    }
  }

  // Сохраняет политику источника зависимости и перечитывает состояние.
  async function setSource(name: string, policy: string): Promise<void> {
    error.value = null;
    try {
      await apiSetSource(name, policy);
      await loadItems();
    } catch (err) {
      error.value = toMessage(err);
    }
  }

  // Обновляет прогресс установки (событие `dep:progress`).
  function setProgress(next: DepProgress): void {
    progress.value = next;
  }

  // Применяет смену состояния зависимости (событие `dep:state`).
  function applyState(state: DepState): void {
    const index = items.value.findIndex((dep) => dep.name === state.name);
    const current = items.value[index];
    if (index === -1 || current === undefined) {
      return;
    }
    const next = items.value.slice();
    next[index] = { ...current, status: state.status, version: state.version };
    items.value = next;
  }

  return {
    items,
    isLoading,
    busyName,
    isClearing,
    progress,
    error,
    hasUpdates,
    refresh,
    checkUpdates,
    install,
    remove,
    removeManaged,
    setSource,
    setProgress,
    applyState,
  };
});
