<script setup lang="ts">
// Строка задачи загрузки: заголовок, состояние, прогресс, кнопки управления и журнал.
import { computed, ref } from "vue";

import {
  TASK_STATES,
  isFinished,
  isRunning,
  progressRatio,
  taskStateLabel,
  useDownloadTaskStore,
} from "@/entities/download-task";
import type { Task, TaskLogLine } from "@/shared/api";
import { copyText, openPath, revealPath } from "@/shared/api";
import { toMessage } from "@/shared/lib/error";
import { formatBytes, formatClock, formatTimestamp } from "@/shared/lib/format";
import { AppButton, AppProgress } from "@/shared/ui";

const props = defineProps<{ task: Task }>();

const store = useDownloadTaskStore();
const error = ref<string | null>(null);
const copiedId = ref<string | null>(null);
// Идёт запрос управления (пауза/докачка/отмена) — блокируем кнопки, чтобы не нажать дважды.
const busy = ref(false);

const running = computed(() => isRunning(props.task));
const queued = computed(() => props.task.state === TASK_STATES.queued);
const paused = computed(() => props.task.state === TASK_STATES.paused);
const finished = computed(() => isFinished(props.task));

// Строки журнала задачи (реактивно — journals из стора).
function journalLines(): TaskLogLine[] {
  return store.journalFor(props.task.id);
}

// Текст прогресса: «скачано / всего» либо только скачано, если размер неизвестен.
function progressText(): string {
  if (props.task.total > 0) {
    return `${formatBytes(props.task.downloaded)} / ${formatBytes(props.task.total)}`;
  }
  return props.task.downloaded > 0 ? formatBytes(props.task.downloaded) : "";
}

// Скорость загрузки.
function speedText(): string {
  return props.task.speed > 0 ? `${formatBytes(props.task.speed)}/с` : "";
}

// Остаток времени в формате м:сс.
function etaText(): string {
  if (props.task.eta <= 0) {
    return "";
  }
  const minutes = Math.floor(props.task.eta / 60);
  const seconds = props.task.eta % 60;
  return `${minutes}:${String(seconds).padStart(2, "0")}`;
}

// Обобщённый вызов действия с блокировкой кнопок и показом ошибки.
async function act(fn: (id: string) => Promise<void>): Promise<void> {
  error.value = null;
  busy.value = true;
  try {
    await fn(props.task.id);
  } catch (err) {
    error.value = toMessage(err);
  } finally {
    busy.value = false;
  }
}

const cancel = (): Promise<void> => act(store.cancel);
const pause = (): Promise<void> => act(store.pause);
const resume = (): Promise<void> => act(store.resume);

async function open(path: string): Promise<void> {
  error.value = null;
  try {
    await openPath(path);
  } catch (err) {
    error.value = toMessage(err);
  }
}

async function reveal(path: string): Promise<void> {
  error.value = null;
  try {
    await revealPath(path);
  } catch (err) {
    error.value = toMessage(err);
  }
}

// При первом раскрытии журнала подтягиваем его с бэкенда, если в памяти пусто.
function onJournalToggle(event: Event): void {
  const details = event.target as HTMLDetailsElement | null;
  if (details?.open === true && journalLines().length === 0) {
    void store.loadJournal(props.task.id);
  }
}

// Текст журнала для копирования (с полными датой и временем).
function journalText(): string {
  return journalLines()
    .map((line) => `[${formatTimestamp(line.time)}] ${line.level}: ${line.message}`)
    .join("\n");
}

// Копирует журнал задачи в системный буфер обмена и показывает обратную связь.
async function copyJournal(): Promise<void> {
  error.value = null;
  try {
    await copyText(journalText());
    copiedId.value = props.task.id;
    window.setTimeout(() => {
      if (copiedId.value === props.task.id) {
        copiedId.value = null;
      }
    }, 1500);
  } catch (err) {
    error.value = toMessage(err);
  }
}
</script>

<template>
  <li class="task" :class="`task--${task.state}`">
    <div class="task__head">
      <span class="task__title" :title="task.url">{{ task.title || task.url }}</span>
      <span class="task__state">{{ taskStateLabel(task.state) }}</span>

      <!-- Управление по состоянию: пауза/докачка/отмена; у завершённых — действия с файлом. -->
      <template v-if="running || queued">
        <AppButton variant="secondary" :disabled="busy" @click="pause">Пауза</AppButton>
        <AppButton variant="secondary" :disabled="busy" @click="cancel">Отменить</AppButton>
      </template>
      <template v-else-if="paused">
        <AppButton :disabled="busy" @click="resume">Докачать</AppButton>
        <AppButton variant="secondary" :disabled="busy" @click="cancel">Отменить</AppButton>
      </template>
      <template v-else-if="finished">
        <AppButton
          v-if="task.state === TASK_STATES.failed && task.logPath"
          variant="secondary"
          @click="open(task.logPath)"
        >
          Логи
        </AppButton>
        <template v-if="task.filePath">
          <AppButton variant="secondary" @click="reveal(task.filePath)">В папке</AppButton>
          <AppButton variant="secondary" @click="open(task.filePath)">Открыть</AppButton>
        </template>
        <AppButton v-else-if="task.outputDir" variant="secondary" @click="open(task.outputDir)">
          Папка
        </AppButton>
      </template>
    </div>

    <!-- Ссылка на источник: мелкий лейбл под названием. -->
    <button
      v-if="task.title"
      type="button"
      class="task__url"
      :title="'Открыть в браузере: ' + task.url"
      @click="open(task.url)"
    >
      {{ task.url }}
    </button>

    <AppProgress v-if="running && task.total > 0" :value="progressRatio(task)" />

    <div v-if="running" class="task__meta">
      <span v-if="progressText()">{{ progressText() }}</span>
      <span v-if="speedText()">{{ speedText() }}</span>
      <span v-if="etaText()">осталось {{ etaText() }}</span>
      <span v-if="task.status" class="task__status">{{ task.status }}</span>
      <span v-else-if="task.total <= 0" class="task__status">Подготовка…</span>
    </div>

    <p v-else-if="queued" class="task__note">Ожидает своей очереди.</p>
    <p v-else-if="paused" class="task__note">Пауза — недокачанный файл сохранён, можно докачать.</p>

    <p v-if="task.state === TASK_STATES.failed && task.error" class="task__error">
      {{ task.error }}
    </p>

    <!-- Журнал загрузки: подробные события (попытки/прокси/ошибки); открывается в любой момент. -->
    <details class="task__journal" @toggle="onJournalToggle">
      <summary class="task__journal-title">Журнал загрузки ({{ journalLines().length }})</summary>
      <div class="task__journal-body">
        <div class="task__journal-actions">
          <AppButton variant="secondary" :disabled="journalLines().length === 0" @click="copyJournal">
            {{ copiedId === task.id ? "Скопировано" : "Копировать" }}
          </AppButton>
        </div>
        <p v-if="journalLines().length === 0" class="task__journal-empty">Пока пусто.</p>
        <ul v-else class="task__journal-list">
          <li
            v-for="(line, index) in journalLines()"
            :key="index"
            class="task__log-line"
            :class="`task__log-line--${line.level}`"
          >
            <span class="task__log-time">{{ formatClock(line.time) }}</span>
            <span class="task__log-message">{{ line.message }}</span>
          </li>
        </ul>
      </div>
    </details>

    <p v-if="error" class="task__error">{{ error }}</p>
  </li>
</template>

<style scoped>
.task {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
}

.task__head {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.task__title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: var(--color-text);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task__state {
  flex: none;
  color: var(--color-text-muted);
  font-size: 12px;
}

.task--completed .task__state {
  color: var(--color-success);
}

.task--failed .task__state {
  color: var(--color-danger);
}

.task--paused .task__state {
  color: var(--color-warning);
}

.task__url {
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

.task__url:hover {
  color: var(--color-accent);
}

.task__meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
  color: var(--color-text-muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.task__status {
  flex: 1 1 100%;
  word-break: break-word;
}

.task__note {
  margin: 0;
  color: var(--color-text-muted);
  font-size: 12px;
}

.task__error {
  margin: 0;
  color: var(--color-danger);
  font-size: 12px;
  line-height: 1.4;
  word-break: break-word;
  white-space: pre-wrap;
}

.task__journal {
  border-top: 1px solid var(--color-border);
  padding-top: var(--space-2);
}

.task__journal-title {
  color: var(--color-text-muted);
  font-size: 12px;
  cursor: pointer;
  user-select: none;
}

.task__journal-body {
  margin-top: var(--space-2);
}

.task__journal-actions {
  display: flex;
  justify-content: flex-end;
  margin-bottom: var(--space-1);
}

.task__journal-empty {
  margin: 0;
  color: var(--color-text-muted);
  font-size: 12px;
}

.task__journal-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 240px;
  margin: 0;
  padding: var(--space-2);
  overflow-y: auto;
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  list-style: none;
}

.task__log-line {
  display: flex;
  gap: var(--space-2);
  color: var(--color-text-muted);
  font-size: 12px;
  line-height: 1.4;
}

.task__log-time {
  flex: none;
  font-variant-numeric: tabular-nums;
}

.task__log-message {
  word-break: break-word;
}

.task__log-line--warn .task__log-message {
  color: var(--color-warning);
}

.task__log-line--error .task__log-message {
  color: var(--color-danger);
}
</style>
