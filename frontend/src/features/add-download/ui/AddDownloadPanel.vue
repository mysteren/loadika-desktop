<script setup lang="ts">
// Компонент `AddDownloadPanel` — сценарий добавления загрузки по ссылке.
// Сначала определяем форматы файла (`-J`), затем рендерим кнопки по фактическим разрешениям
// (с оценкой размера) и раскрывающийся список форматов для точного выбора.
// Состояние живёт в сторе, поэтому не теряется при переключении вкладок.
import { onMounted } from "vue";

import { formatBytes } from "@/shared/lib/format";
import { AppButton } from "@/shared/ui";

import { useAddDownloadStore } from "../model/store";
import FormatTable from "./FormatTable.vue";

const store = useAddDownloadStore();

// Значение галочки по умолчанию берём из настроек и подгружаем пул прокси.
onMounted(() => {
  void store.init();
});
</script>

<template>
  <div class="add-download">
    <div class="add-download__input-row">
      <input
        v-model="store.url"
        class="add-download__input"
        type="url"
        placeholder="Вставьте ссылку на видео"
        spellcheck="false"
        @input="store.onUrlInput"
        @keyup.enter="store.detect"
      />
      <AppButton :disabled="!store.canAct || store.isDetecting" @click="store.detect">
        {{ store.isDetecting ? "Определяем…" : "Определить форматы" }}
      </AppButton>
    </div>

    <label class="add-download__proxy">
      <input v-model="store.useProxy" type="checkbox" :disabled="!store.proxyAvailable" />
      <span>Использовать прокси</span>
      <span v-if="!store.proxyAvailable" class="add-download__proxy-hint">
        (прокси не добавлены — скачается напрямую)
      </span>
    </label>

    <p v-if="store.error" class="add-download__error">{{ store.error }}</p>
    <p v-else-if="store.isCreating" class="add-download__status">Создаём задачу…</p>

    <p v-if="store.title" class="add-download__title" :title="store.title">{{ store.title }}</p>

    <div v-if="store.qualities.length > 0" class="add-download__actions">
      <button
        v-for="option in store.qualities"
        :key="option.id"
        type="button"
        class="quality"
        :title="option.hint"
        :disabled="store.isCreating"
        @click="store.start(option)"
      >
        <span class="quality__label">{{ option.label }}</span>
        <span v-if="option.size > 0" class="quality__size">≈ {{ formatBytes(option.size) }}</span>
      </button>
    </div>

    <!-- Список форматов — подсказка: свёрнут по умолчанию и ограничен по высоте. -->
    <details v-if="store.mediaFormats.length > 0" class="add-download__hint">
      <summary class="add-download__hint-title">
        Доступные форматы ({{ store.mediaFormats.length }}) — для точного выбора
      </summary>
      <div class="add-download__hint-body">
        <FormatTable :formats="store.mediaFormats" />
      </div>
    </details>

    <!-- Сырой ответ yt-dlp: команда и её вывод (для «точно знать, что отвечает библиотека»). -->
    <details
      v-if="store.result && (store.result.command || store.result.output)"
      class="add-download__output"
      :open="store.error !== null"
    >
      <summary class="add-download__output-title">
        Диагностика (ответ yt-dlp)
      </summary>
      <div class="add-download__output-body">
        <p v-if="store.result.command" class="add-download__output-command">
          {{ store.result.command }}
        </p>
        <pre v-if="store.result.output" class="add-download__output-text">{{ store.result.output }}</pre>
        <p v-else class="add-download__output-empty">Пустой вывод.</p>
      </div>
    </details>
  </div>
</template>

<style scoped>
.add-download {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.add-download__input-row {
  display: flex;
  gap: var(--space-2);
}

.add-download__input {
  flex: 1;
  min-width: 0;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text);
  font: inherit;
}

.add-download__input:focus {
  border-color: var(--color-accent);
  outline: none;
}

.add-download__proxy {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--color-text);
  font-size: 13px;
  user-select: none;
}

.add-download__proxy-hint {
  color: var(--color-text-muted);
  font-size: 12px;
}

.add-download__error {
  margin: 0;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-danger);
  font-size: 13px;
  line-height: 1.4;
  word-break: break-word;
  white-space: pre-wrap;
}

.add-download__status {
  margin: 0;
  color: var(--color-text-muted);
  font-size: 13px;
}

.add-download__title {
  margin: 0;
  overflow: hidden;
  color: var(--color-text);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.add-download__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

/* Кнопка качества — компактная, вторичного вида. */
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
  transition:
    box-shadow 0.16s ease,
    transform 0.06s ease;
}

.quality__size {
  color: var(--color-text-muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.quality:hover:not(:disabled) {
  box-shadow: var(--shadow-sm);
}

.quality:active:not(:disabled) {
  transform: translateY(1px);
}

.quality:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  box-shadow: none;
}

.add-download__hint {
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
}

.add-download__hint-title {
  color: var(--color-text-muted);
  font-size: 12px;
  cursor: pointer;
  user-select: none;
}

/* Ограничиваем высоту списка — длинные наборы форматов прокручиваются, не растягивая страницу. */
.add-download__hint-body {
  max-height: 260px;
  margin-top: var(--space-2);
  overflow-y: auto;
}

.add-download__output {
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
}

.add-download__output-title {
  color: var(--color-text-muted);
  font-size: 12px;
  cursor: pointer;
  user-select: none;
}

.add-download__output-body {
  margin-top: var(--space-2);
}

.add-download__output-command {
  margin: 0 0 var(--space-2);
  color: var(--color-text-muted);
  font-family: monospace;
  font-size: 12px;
  word-break: break-all;
}

.add-download__output-text {
  max-height: 240px;
  margin: 0;
  padding: var(--space-2);
  overflow: auto;
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text);
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-word;
}

.add-download__output-empty {
  margin: 0;
  color: var(--color-text-muted);
  font-size: 12px;
}
</style>
