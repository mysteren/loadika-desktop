// Стор сущности `proxy` — единый источник истины о пуле прокси.
// UI не обращается к бэкенду напрямую: только через этот стор поверх `shared/api`.
import { defineStore } from "pinia";
import { computed, ref, shallowRef } from "vue";

import {
  checkProxies as apiCheck,
  clearProxies as apiClear,
  getProxies as apiGet,
  getProxyStrategies as apiGetStrategies,
  getSettings,
  importProxies as apiImport,
  removeProxy as apiRemove,
  setProxyEnabled as apiSetEnabled,
  setProxyStrategy as apiSetStrategy,
  setUseProxy as apiSetUseProxy,
} from "@/shared/api";
import type { ImportResult, ProxyView } from "@/shared/api";
import type { ProxyHealth } from "@/shared/config/events";
import { toMessage } from "@/shared/lib/error";

export const useProxyStore = defineStore("proxy", () => {
  // Прокси с рантайм-состоянием (health, задержка, сбои, cooldown).
  const items = shallowRef<ProxyView[]>([]);
  const strategies = ref<string[]>([]);
  const strategy = ref<string>("");
  // Использовать ли пул прокси для загрузок (настройка).
  const useProxy = ref(false);
  const isLoading = ref(false);
  const error = ref<string | null>(null);

  const total = computed(() => items.value.length);
  const enabledCount = computed(() => items.value.filter((p) => p.enabled).length);

  // Заменяет запись прокси по id.
  function patch(id: string, update: (p: ProxyView) => ProxyView): void {
    const index = items.value.findIndex((p) => p.id === id);
    const current = items.value[index];
    if (current === undefined) {
      return;
    }
    const next = items.value.slice();
    next[index] = update(current);
    items.value = next;
  }

  // Загружает пул, список стратегий и текущую стратегию.
  async function load(): Promise<void> {
    isLoading.value = true;
    error.value = null;
    try {
      const [list, availableStrategies, settings] = await Promise.all([
        apiGet(),
        apiGetStrategies(),
        getSettings(),
      ]);
      items.value = list;
      strategies.value = availableStrategies;
      strategy.value = settings.proxyStrategy;
      useProxy.value = settings.useProxy;
    } catch (err) {
      error.value = toMessage(err);
    } finally {
      isLoading.value = false;
    }
  }

  // Импортирует список прокси и перечитывает пул.
  async function importList(text: string): Promise<ImportResult | null> {
    error.value = null;
    try {
      const result = await apiImport(text);
      items.value = await apiGet();
      return result;
    } catch (err) {
      error.value = toMessage(err);
      return null;
    }
  }

  // Удаляет прокси.
  async function remove(id: string): Promise<void> {
    error.value = null;
    try {
      await apiRemove(id);
      items.value = items.value.filter((p) => p.id !== id);
    } catch (err) {
      error.value = toMessage(err);
    }
  }

  // Включает/выключает прокси.
  async function setEnabled(id: string, enabled: boolean): Promise<void> {
    error.value = null;
    try {
      await apiSetEnabled(id, enabled);
      patch(id, (p) => ({ ...p, enabled }));
    } catch (err) {
      error.value = toMessage(err);
    }
  }

  // Очищает пул.
  async function clear(): Promise<void> {
    error.value = null;
    try {
      await apiClear();
      items.value = [];
    } catch (err) {
      error.value = toMessage(err);
    }
  }

  // Задаёт стратегию выбора прокси.
  async function setStrategy(next: string): Promise<void> {
    error.value = null;
    try {
      await apiSetStrategy(next);
      strategy.value = next;
    } catch (err) {
      error.value = toMessage(err);
    }
  }

  // Включает/выключает использование пула прокси для загрузок.
  async function setUseProxy(enabled: boolean): Promise<void> {
    error.value = null;
    try {
      await apiSetUseProxy(enabled);
      useProxy.value = enabled;
    } catch (err) {
      error.value = toMessage(err);
    }
  }

  // Проверяет доступность прокси и обновляет список.
  async function check(): Promise<void> {
    isLoading.value = true;
    error.value = null;
    try {
      items.value = await apiCheck();
    } catch (err) {
      error.value = toMessage(err);
    } finally {
      isLoading.value = false;
    }
  }

  // Применяет результат проверки (событие proxy:health).
  function applyHealth(health: ProxyHealth): void {
    patch(health.proxyId, (p) => ({
      ...p,
      health: health.ok ? "ok" : "fail",
      latencyMs: health.ok ? health.latencyMs : p.latencyMs,
    }));
  }

  return {
    items,
    strategies,
    strategy,
    useProxy,
    isLoading,
    error,
    total,
    enabledCount,
    load,
    importList,
    remove,
    setEnabled,
    clear,
    setStrategy,
    setUseProxy,
    check,
    applyHealth,
  };
});
