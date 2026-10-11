// Стор фичи `add-download` — состояние панели добавления загрузки.
// Живёт в Pinia, поэтому переживает размонтирование страницы при переключении вкладок
// (страницы рендерятся через <component :is>, а не keep-alive).
// Компонент остаётся «глупым»: только разметка и вызовы действий стора.
import { defineStore } from "pinia";
import { computed, ref, shallowRef } from "vue";

import { useDownloadTaskStore } from "@/entities/download-task";
import { useProxyStore } from "@/entities/proxy";
import { getSettings, inspect as apiInspect } from "@/shared/api";
import type { DownloadMeta, Inspect, QualityOption } from "@/shared/api";
import { toMessage } from "@/shared/lib/error";

export const useAddDownloadStore = defineStore("add-download", () => {
  const taskStore = useDownloadTaskStore();
  const proxyStore = useProxyStore();

  // --- Состояние формы ---
  const url = ref("");
  // Результат определения форматов; shallowRef — объект меняем целиком.
  const result = shallowRef<Inspect | null>(null);
  // Идти ли через прокси для ЭТОЙ загрузки. По умолчанию — из глобальной настройки.
  const useProxy = ref(true);
  const isDetecting = ref(false);
  const isCreating = ref(false);
  const error = ref<string | null>(null);

  // Значение по умолчанию из настроек подставляем один раз, чтобы не затирать выбор пользователя.
  let proxyDefaultLoaded = false;

  // --- Производные значения для вёрстки ---
  const qualities = computed<QualityOption[]>(() => result.value?.qualities ?? []);
  // Служебные форматы без медиа (раскадровки/mhtml) в подсказке не нужны — только шум.
  const mediaFormats = computed(() => (result.value?.formats ?? []).filter((f) => f.kind !== ""));
  const title = computed(() => result.value?.title ?? "");
  const canAct = computed(() => url.value.trim() !== "");
  // Галочка «через прокси» осмысленна, только если есть включённые прокси.
  const proxyAvailable = computed(() => proxyStore.enabledCount > 0);

  // Подставляет значение по умолчанию из настроек (один раз) и подгружает пул прокси.
  async function init(): Promise<void> {
    if (!proxyDefaultLoaded) {
      try {
        useProxy.value = (await getSettings()).useProxy;
      } catch {
        // Не критично — остаётся значение по умолчанию.
      }
      proxyDefaultLoaded = true;
    }
    if (proxyStore.items.length === 0) {
      await proxyStore.load();
    }
  }

  // Правка ссылки делает ранее определённые форматы неактуальными.
  function onUrlInput(): void {
    result.value = null;
    error.value = null;
  }

  // Запрашивает метаданные ссылки; при неудаче показывает и сообщение, и сырой ответ yt-dlp.
  async function detect(): Promise<void> {
    if (!canAct.value) {
      return;
    }
    // Очищаем прошлый разбор (в т.ч. блок «Ответ yt-dlp»), чтобы не показывать устаревшее.
    result.value = null;
    isDetecting.value = true;
    error.value = null;
    try {
      const inspected = await apiInspect(url.value.trim(), useProxy.value);
      result.value = inspected;
      if (!inspected.ok) {
        error.value = inspected.error || "не удалось получить метаданные";
      }
    } catch (err) {
      result.value = null;
      error.value = toMessage(err);
    } finally {
      isDetecting.value = false;
    }
  }

  // Запускает загрузку с выбранным вариантом качества (метаданные попадут в историю).
  async function start(option: QualityOption): Promise<void> {
    if (!canAct.value) {
      return;
    }
    isCreating.value = true;
    error.value = null;
    try {
      const meta: DownloadMeta = {
        title: result.value?.title ?? "",
        duration: result.value?.duration ?? 0,
        quality: option.label,
        details: option.hint,
      };
      await taskStore.create(url.value.trim(), option.selector, useProxy.value, meta);
      // Загрузка начата — очищаем разбор (в т.ч. блок «Ответ yt-dlp»).
      result.value = null;
    } catch (err) {
      error.value = toMessage(err);
    } finally {
      isCreating.value = false;
    }
  }

  // Готовит повторную загрузку: подставляет ссылку и сразу определяет форматы (для истории).
  async function redownload(nextUrl: string): Promise<void> {
    url.value = nextUrl;
    result.value = null;
    error.value = null;
    await detect();
  }

  return {
    url,
    result,
    useProxy,
    isDetecting,
    isCreating,
    error,
    qualities,
    mediaFormats,
    title,
    canAct,
    proxyAvailable,
    init,
    onUrlInput,
    detect,
    start,
    redownload,
  };
});
