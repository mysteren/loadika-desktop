<script setup lang="ts">
// Экран «Видео» — библиотека завершённых загрузок: что за видео, качество, размер, длительность,
// кодеки и действия (открыть плеером по умолчанию, показать в папке, перекачать, удалить).
import { onMounted, ref } from "vue";

import { taskStateLabel } from "@/entities/download-task";
import { useAddDownloadStore } from "@/features/add-download";
import {
  clearHistory,
  getHistory,
  openPath,
  removeHistory,
  removeHistoryWithFile,
  revealPath,
} from "@/shared/api";
import type { HistoryEntry, QualityOption } from "@/shared/api";
import { toMessage } from "@/shared/lib/error";
import { formatBytes, formatDuration, formatTimestamp } from "@/shared/lib/format";
import { AppButton } from "@/shared/ui";

const addStore = useAddDownloadStore();

const entries = ref<HistoryEntry[]>([]);
const isLoading = ref(false);
const isClearing = ref(false);
const error = ref<string | null>(null);
// Идентификатор записи, ожидающей подтверждения удаления вместе с файлом.
const confirmingId = ref<string | null>(null);
// Идентификатор записи, для которой открыт выбор качества (перекачка прямо здесь).
const redownloadId = ref<string | null>(null);

onMounted(() => {
  void load();
});

async function load(): Promise<void> {
  isLoading.value = true;
  error.value = null;
  try {
    entries.value = await getHistory();
  } catch (err) {
    error.value = toMessage(err);
  } finally {
    isLoading.value = false;
  }
}

async function clear(): Promise<void> {
  isClearing.value = true;
  error.value = null;
  try {
    await clearHistory();
    entries.value = [];
  } catch (err) {
    error.value = toMessage(err);
  } finally {
    isClearing.value = false;
  }
}

// Заголовок записи; если его нет — показываем ссылку.
function title(entry: HistoryEntry): string {
  return entry.title || entry.url;
}

// Есть ли отдельный заголовок (если нет — название и есть ссылка, дублировать не нужно).
function hasTitle(entry: HistoryEntry): boolean {
  return entry.title !== "";
}

// Длительность видео, если известна.
function durationText(entry: HistoryEntry): string {
  return formatDuration(entry.duration);
}

// Размер файла, если известен.
function sizeText(entry: HistoryEntry): string {
  return entry.size > 0 ? formatBytes(entry.size) : "";
}

// Расширение файла (контейнер) из пути — например, MP4.
function extText(entry: HistoryEntry): string {
  if (entry.filePath === "") {
    return "";
  }
  const name = entry.filePath.split(/[\\/]/).pop() ?? "";
  const dot = name.lastIndexOf(".");
  return dot >= 0 ? name.slice(dot + 1).toUpperCase() : "";
}

// Открывает файл приложением по умолчанию (для видео — видеоплеером по умолчанию).
async function open(path: string): Promise<void> {
  error.value = null;
  try {
    await openPath(path);
  } catch (err) {
    error.value = toMessage(err);
  }
}

// Открывает файловый менеджер с выделением файла.
async function reveal(path: string): Promise<void> {
  error.value = null;
  try {
    await revealPath(path);
  } catch (err) {
    error.value = toMessage(err);
  }
}

// Открывает/закрывает выбор качества для перекачки прямо на этой странице (без перехода на другую вкладку).
async function redownload(entry: HistoryEntry): Promise<void> {
  if (redownloadId.value === entry.id) {
    redownloadId.value = null;
    return;
  }
  redownloadId.value = entry.id;
  await addStore.redownload(entry.url);
}

// Запускает перекачку выбранного качества и закрывает выбор.
async function startRedownload(option: QualityOption): Promise<void> {
  await addStore.start(option);
  redownloadId.value = null;
}

// Удаляет запись (без файла).
async function removeRecord(entry: HistoryEntry): Promise<void> {
  error.value = null;
  try {
    await removeHistory(entry.id);
    entries.value = entries.value.filter((e) => e.id !== entry.id);
  } catch (err) {
    error.value = toMessage(err);
  }
}

// Удаляет запись вместе с файлом. Требует повторного нажатия (подтверждение).
async function removeWithFile(entry: HistoryEntry): Promise<void> {
  if (confirmingId.value !== entry.id) {
    confirmingId.value = entry.id;
    window.setTimeout(() => {
      if (confirmingId.value === entry.id) {
        confirmingId.value = null;
      }
    }, 4000);
    return;
  }
  error.value = null;
  try {
    await removeHistoryWithFile(entry.id);
    entries.value = entries.value.filter((e) => e.id !== entry.id);
  } catch (err) {
    error.value = toMessage(err);
  } finally {
    confirmingId.value = null;
  }
}
</script>

<template>
  <section class="page">
    <header class="page-header">
      <h1 class="page-title">Видео</h1>
      <AppButton
        variant="secondary"
        :disabled="entries.length === 0 || isClearing"
        @click="clear"
      >
        {{ isClearing ? "Очищаем…" : "Очистить список" }}
      </AppButton>
    </header>

    <p v-if="error" class="video__error">{{ error }}</p>
    <p v-if="entries.length === 0" class="page-hint">
      {{ isLoading ? "Загружаем…" : "Пока ничего не скачано." }}
    </p>

    <ul v-else class="video__list">
      <li
        v-for="entry in entries"
        :key="entry.id"
        class="video"
        :class="`video--${entry.state}`"
      >
        <div class="video__head">
          <span class="video__title" :title="title(entry)">{{ title(entry) }}</span>
          <span class="video__state">{{ taskStateLabel(entry.state) }}</span>
        </div>

        <!-- Ссылка на источник: мелкий лейбл под названием (открывается в браузере по умолчанию). -->
        <button
          v-if="hasTitle(entry)"
          type="button"
          class="video__url"
          :title="'Открыть в браузере: ' + entry.url"
          @click="open(entry.url)"
        >
          {{ entry.url }}
        </button>

        <div class="video__meta">
          <span v-if="entry.quality" class="video__quality">{{ entry.quality }}</span>
          <span v-if="extText(entry)">{{ extText(entry) }}</span>
          <span v-if="durationText(entry)">{{ durationText(entry) }}</span>
          <span v-if="sizeText(entry)">{{ sizeText(entry) }}</span>
          <span>{{ formatTimestamp(entry.finishedAt) }}</span>
          <span v-if="entry.proxy" :title="entry.proxy">прокси</span>
        </div>

        <p v-if="entry.details" class="video__details">{{ entry.details }}</p>

        <p v-if="entry.error" class="video__entry-error">{{ entry.error }}</p>

        <div class="video__actions">
          <template v-if="entry.filePath">
            <AppButton variant="secondary" @click="open(entry.filePath)">Открыть</AppButton>
            <AppButton variant="secondary" @click="reveal(entry.filePath)">В папке</AppButton>
          </template>
          <AppButton
            v-else-if="entry.outputDir"
            variant="secondary"
            @click="open(entry.outputDir)"
          >
            Папка
          </AppButton>

          <AppButton variant="secondary" @click="redownload(entry)">Перекачать</AppButton>
          <AppButton variant="secondary" @click="removeRecord(entry)">Удалить запись</AppButton>
          <AppButton
            v-if="entry.filePath"
            variant="secondary"
            @click="removeWithFile(entry)"
          >
            {{ confirmingId === entry.id ? "Точно удалить файл?" : "Удалить с файлом" }}
          </AppButton>
        </div>

        <!-- Выбор качества для перекачки прямо в карточке (форматы определяются по ссылке). -->
        <div v-if="redownloadId === entry.id" class="video__redownload">
          <p v-if="addStore.isDetecting" class="video__redownload-status">Определяем форматы…</p>
          <p v-else-if="addStore.error" class="video__redownload-error">
            {{ addStore.error }}
          </p>
          <div v-else-if="addStore.qualities.length > 0" class="video__qualities">
            <button
              v-for="option in addStore.qualities"
              :key="option.id"
              type="button"
              class="quality"
              :title="option.hint"
              :disabled="addStore.isCreating"
              @click="startRedownload(option)"
            >
              <span class="quality__label">{{ option.label }}</span>
              <span v-if="option.size > 0" class="quality__size">
                ≈ {{ formatBytes(option.size) }}
              </span>
            </button>
          </div>
        </div>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.video__error {
  margin: 0;
  color: var(--color-danger);
  font-size: 13px;
}

.video__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.video {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
}

.video__head {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
}

.video__title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: var(--color-text);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.video__state {
  flex: none;
  color: var(--color-text-muted);
  font-size: 12px;
}

.video__url {
  align-self: flex-start;
  max-width: 100%;
  padding: 0;
  border: 0;
  background: transparent;
  overflow: hidden;
  color: var(--color-text-muted);
  font-size: 12px;
  text-align: left;
  text-decoration: underline;
  text-underline-offset: 2px;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
}

.video__url:hover {
  color: var(--color-accent);
}

.video--completed .video__state {
  color: var(--color-success);
}

.video--failed .video__state {
  color: var(--color-danger);
}

.video__meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
  color: var(--color-text-muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.video__quality {
  color: var(--color-text);
  font-weight: 600;
}

.video__details {
  margin: 0;
  color: var(--color-text-muted);
  font-size: 12px;
  word-break: break-word;
}

.video__entry-error {
  margin: 0;
  color: var(--color-danger);
  font-size: 12px;
  word-break: break-word;
}

.video__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.video__redownload {
  padding-top: var(--space-2);
  border-top: 1px solid var(--color-border);
}

.video__redownload-status {
  margin: 0;
  color: var(--color-text-muted);
  font-size: 12px;
}

.video__redownload-error {
  margin: 0;
  color: var(--color-danger);
  font-size: 12px;
  word-break: break-word;
  white-space: pre-wrap;
}

.video__qualities {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

/* Кнопка качества — компактная, вторичного вида (как на панели добавления). */
.quality {
  display: inline-flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  padding: var(--space-1) var(--space-3);
  border: 0;
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text);
  box-shadow: var(--shadow-xs);
  cursor: pointer;
  text-align: left;
}

.quality:hover:not(:disabled) {
  box-shadow: var(--shadow-sm);
}

.quality:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.quality__size {
  color: var(--color-text-muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}
</style>
