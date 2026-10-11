<script setup lang="ts">
// Виджет `download-list` — задачи: ожидающие/выполняемые/на паузе отдельно, завершённые — сворачиваемы.
import { storeToRefs } from "pinia";
import { computed, onMounted, ref } from "vue";

import { isFinished, isPending, useDownloadTaskStore } from "@/entities/download-task";
import { toMessage } from "@/shared/lib/error";
import { useNavigationStore } from "@/shared/stores/navigation";
import { AppButton } from "@/shared/ui";

import TaskItem from "./TaskItem.vue";

const store = useDownloadTaskStore();
const nav = useNavigationStore();
const { tasks, runningCount, queuedCount, pausedCount } = storeToRefs(store);
const error = ref<string | null>(null);

// Незавершённые (в очереди/выполняются/на паузе) и завершённые задачи.
const pendingTasks = computed(() => tasks.value.filter(isPending));
const finishedTasks = computed(() => tasks.value.filter(isFinished));

// Показываем задачи при появлении виджета.
onMounted(() => {
  void load();
});

async function load(): Promise<void> {
  error.value = null;
  try {
    await store.load();
  } catch (err) {
    error.value = toMessage(err);
  }
}

// Форма множественного числа для русского («1 загружается», «2 загружаются»).
function plural(count: number, one: string, many: string): string {
  const mod10 = count % 10;
  const mod100 = count % 100;
  return mod10 >= 2 && mod10 <= 4 && !(mod100 >= 12 && mod100 <= 14) ? many : one;
}

// Сводка состояний для шапки: «1 загружается · 2 в очереди · 1 на паузе».
// computed — чтобы не пересчитывать строку на каждый ре-рендер от событий прогресса.
const summary = computed(() => {
  const parts: string[] = [];
  if (runningCount.value > 0) {
    parts.push(`${runningCount.value} ${plural(runningCount.value, "загружается", "загружаются")}`);
  }
  if (queuedCount.value > 0) {
    parts.push(`${queuedCount.value} в очереди`);
  }
  if (pausedCount.value > 0) {
    parts.push(`${pausedCount.value} на паузе`);
  }
  return parts.length > 0 ? parts.join(" · ") : "нет активных загрузок";
});
</script>

<template>
  <div class="downloads">
    <div class="downloads__header">
      <h2 class="downloads__title">Задачи</h2>
      <span class="downloads__count">{{ summary }}</span>
      <AppButton variant="secondary" @click="nav.navigate('history')">Видео</AppButton>
    </div>

    <p v-if="error" class="downloads__error">{{ error }}</p>
    <p v-if="tasks.length === 0" class="downloads__empty">Задач пока нет.</p>

    <template v-else>
      <ul v-if="pendingTasks.length > 0" class="downloads__list">
        <TaskItem v-for="task in pendingTasks" :key="task.id" :task="task" />
      </ul>
      <p v-else class="downloads__idle">Активных загрузок нет.</p>

      <details
        v-if="finishedTasks.length > 0"
        class="downloads__finished"
        :open="pendingTasks.length === 0"
      >
        <summary class="downloads__finished-title">Завершённые ({{ finishedTasks.length }})</summary>
        <ul class="downloads__list">
          <TaskItem v-for="task in finishedTasks" :key="task.id" :task="task" />
        </ul>
      </details>
    </template>
  </div>
</template>

<style scoped>
.downloads {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.downloads__header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.downloads__title {
  flex: 1;
  margin: 0;
  color: var(--color-text);
  font-size: 14px;
  font-weight: 600;
}

.downloads__count {
  color: var(--color-text-muted);
  font-size: 12px;
}

.downloads__error {
  margin: 0;
  color: var(--color-danger);
  font-size: 13px;
}

.downloads__empty,
.downloads__idle {
  margin: 0;
  color: var(--color-text-muted);
  font-size: 13px;
}

.downloads__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin: 0;
  padding: 0;
  list-style: none;
}

.downloads__finished {
  border-top: 1px solid var(--color-border);
  padding-top: var(--space-2);
}

.downloads__finished-title {
  color: var(--color-text-muted);
  font-size: 13px;
  cursor: pointer;
  user-select: none;
}

.downloads__finished[open] .downloads__finished-title {
  margin-bottom: var(--space-3);
}
</style>
