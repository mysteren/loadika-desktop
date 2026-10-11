<script setup lang="ts">
// Виджет `deps-status` — ненавязчивая inline-строка чипсов со статусом зависимостей.
// Размещается в футере страницы, чтобы не отвлекать от основного контента.
import { storeToRefs } from "pinia";
import { onMounted } from "vue";

import { isInstalled, isUpdateAvailable, useDependencyStore } from "@/entities/dependency";
import { IconCheck } from "@/shared/ui";

const store = useDependencyStore();
const { items, isLoading, error } = storeToRefs(store);

// При появлении виджета показываем локальный статус (без обращения к сети).
onMounted(() => {
  void store.refresh();
});

// Проверяем свежие версии всех зависимостей (сеть).
function checkAll(): void {
  void store.checkUpdates();
}
</script>

<template>
  <div class="deps-inline">
    <span class="deps-inline__label">Зависимости</span>

    <ul class="deps-inline__list">
      <li
        v-for="dep in items"
        :key="dep.name"
        class="deps-chip"
        :class="{ 'deps-chip--missing': !isInstalled(dep) }"
        :title="dep.path"
      >
        <IconCheck v-if="isInstalled(dep)" class="deps-chip__check" />
        <span v-else class="deps-chip__dash" aria-hidden="true">—</span>
        <span class="deps-chip__name">{{ dep.name }}</span>
        <span v-if="dep.version" class="deps-chip__version">{{ dep.version }}</span>
        <span
          v-if="isUpdateAvailable(dep)"
          class="deps-chip__update"
          :title="`Доступно обновление ${dep.latestVersion}`"
          >↻</span
        >
      </li>
      <li v-if="items.length === 0 && !isLoading" class="deps-inline__empty">нет данных</li>
    </ul>

    <span v-if="error" class="deps-inline__error">{{ error }}</span>

    <button
      type="button"
      class="deps-inline__action"
      :disabled="isLoading"
      @click="checkAll"
    >
      {{ isLoading ? "Проверяем…" : "Проверить" }}
    </button>
  </div>
</template>

<style scoped>
.deps-inline {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-2);
  font-size: 12px;
  color: var(--color-text-muted);
}

.deps-inline__label {
  color: var(--color-text-muted);
  font-weight: 600;
  font-size: 11px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  opacity: 0.7;
}

.deps-inline__list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

/* Ненавязчивый чип: тонкая рамка вместо фона и тени. */
.deps-chip {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: 1px var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: 999px;
  color: var(--color-text-muted);
}

.deps-chip__check {
  flex: none;
  width: 12px;
  height: 12px;
  color: var(--color-success);
}

.deps-chip__dash {
  width: 12px;
  text-align: center;
}

.deps-chip__name {
  color: var(--color-text);
}

.deps-chip--missing .deps-chip__name {
  color: var(--color-text-muted);
}

.deps-chip__version {
  font-variant-numeric: tabular-nums;
  opacity: 0.8;
}

.deps-chip__update {
  color: var(--color-accent);
}

.deps-inline__error {
  color: var(--color-danger);
}

.deps-inline__empty {
  color: var(--color-text-muted);
}

/* Действие — как текстовая ссылка, без кнопочного вида. */
.deps-inline__action {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  text-decoration: underline;
  text-underline-offset: 2px;
  transition: color 0.16s ease;
}

.deps-inline__action:hover:not(:disabled) {
  color: var(--color-text);
}

.deps-inline__action:disabled {
  opacity: 0.5;
  cursor: default;
}
</style>
