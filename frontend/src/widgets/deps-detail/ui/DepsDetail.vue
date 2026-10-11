<script setup lang="ts">
// Виджет `deps-detail` — детальный блок управления зависимостями (вкладка «Зависимости»).
// Показывает назначение, статус, версию, источник, путь и время обновления, позволяет выбрать
// источник (системная/локальная) и выполнить действия; во время установки показывает прогресс.
import { storeToRefs } from "pinia";
import { computed, onMounted } from "vue";

import {
  DEPENDENCY_SOURCES,
  dependencyMeta,
  dependencyStatusLabel,
  isManaged,
  isMissing,
  isUpdateAvailable,
  useDependencyStore,
} from "@/entities/dependency";
import { AppButton, AppProgress } from "@/shared/ui";
import { formatDateTime } from "@/shared/lib/format";

const store = useDependencyStore();
const { items, isLoading, isClearing, busyName, progress, error } = storeToRefs(store);

// Полная очистка имеет смысл, только если есть управляемые (скачанные) копии.
const hasManaged = computed(() => items.value.some(isManaged));

// Представление строки: объединяем рантайм-данные из стора со статичными сведениями,
// чтобы разметка оставалась простой, а вычисления не повторялись в шаблоне.
const rows = computed(() =>
  items.value.map((dep) => {
    const meta = dependencyMeta(dep.name);
    return {
      name: dep.name,
      title: meta.title,
      purpose: meta.purpose,
      source: meta.source,
      path: dep.path,
      status: dep.status,
      statusLabel: dependencyStatusLabel(dep.status),
      sourcePolicy: dep.source,
      version: dep.version,
      updatedAtLabel: formatDateTime(dep.updatedAt),
      latestVersion: dep.latestVersion,
      updateAvailable: isUpdateAvailable(dep),
      managed: isManaged(dep),
      missing: isMissing(dep),
    };
  }),
);

// Показываем локальный статус при открытии вкладки (без обращения к сети).
onMounted(() => {
  void store.refresh();
});

function checkUpdates(): void {
  void store.checkUpdates();
}

function install(name: string): void {
  void store.install(name);
}

function remove(name: string): void {
  void store.remove(name);
}

function clearManaged(): void {
  void store.removeManaged();
}

// Смена политики источника. Выбор «Локальная» дополнительно принудительно ставит
// управляемую копию, если её ещё нет (принудительное скачивание локальной версии).
async function changeSource(name: string, event: Event): Promise<void> {
  const value = (event.target as HTMLSelectElement).value;
  await store.setSource(name, value);
  if (value !== "managed") {
    return;
  }
  const dep = items.value.find((item) => item.name === name);
  if (dep !== undefined && dep.status !== "managed") {
    await store.install(name);
  }
}

// Доля прогресса установки зависимости (0..1).
function ratioFor(name: string): number {
  const p = progress.value;
  if (p === null || p.name !== name) {
    return 0;
  }
  if (p.phase === "install" || p.total <= 0) {
    return p.phase === "install" ? 1 : 0;
  }
  return p.received / p.total;
}

// Идёт ли сейчас установка/обновление конкретной зависимости.
function isBusy(name: string): boolean {
  return busyName.value === name;
}
</script>

<template>
  <div class="deps-detail">
    <div class="deps-detail__toolbar">
      <AppButton
        v-if="hasManaged"
        variant="secondary"
        :disabled="isClearing"
        @click="clearManaged"
      >
        {{ isClearing ? "Удаляем…" : "Удалить загруженные" }}
      </AppButton>
      <AppButton variant="secondary" :disabled="isLoading" @click="checkUpdates">
        {{ isLoading ? "Проверяем…" : "Проверить обновления" }}
      </AppButton>
    </div>

    <p v-if="error" class="deps-detail__error">{{ error }}</p>

    <article v-for="row in rows" :key="row.name" class="dep-card">
      <header class="dep-card__head">
        <div class="dep-card__title">
          <span class="dep-card__name">{{ row.title }}</span>
          <span class="dep-card__status" :class="`dep-card__status--${row.status}`">
            {{ row.statusLabel }}
          </span>
        </div>
        <span class="dep-card__version">{{ row.version || "—" }}</span>
      </header>

      <dl class="dep-fields">
        <div class="dep-field">
          <dt class="dep-field__label">Назначение</dt>
          <dd class="dep-field__value">{{ row.purpose }}</dd>
        </div>
        <div class="dep-field">
          <dt class="dep-field__label">Источник загрузки</dt>
          <dd class="dep-field__value">{{ row.source }}</dd>
        </div>
        <div class="dep-field">
          <dt class="dep-field__label">Использовать</dt>
          <dd class="dep-field__value">
            <select
              class="dep-select"
              :value="row.sourcePolicy"
              :disabled="isBusy(row.name)"
              @change="changeSource(row.name, $event)"
            >
              <option v-for="option in DEPENDENCY_SOURCES" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          </dd>
        </div>
        <div class="dep-field">
          <dt class="dep-field__label">Путь</dt>
          <dd class="dep-field__value dep-field__value--mono" :title="row.path">
            {{ row.path || "—" }}
          </dd>
        </div>
        <div class="dep-field">
          <dt class="dep-field__label">Обновлено</dt>
          <dd class="dep-field__value">{{ row.updatedAtLabel }}</dd>
        </div>
        <div class="dep-field">
          <dt class="dep-field__label">Свежая версия</dt>
          <dd class="dep-field__value">
            <template v-if="row.latestVersion">{{ row.latestVersion }}</template>
            <span v-else class="dep-field__muted">не проверялась</span>
          </dd>
        </div>
      </dl>

      <AppProgress v-if="isBusy(row.name)" class="dep-card__progress" :value="ratioFor(row.name)" />

      <div class="dep-card__actions">
        <template v-if="row.missing">
          <AppButton :disabled="isBusy(row.name)" @click="install(row.name)">
            Установить
          </AppButton>
        </template>
        <template v-else-if="row.managed">
          <AppButton
            :variant="row.updateAvailable ? 'primary' : 'secondary'"
            :disabled="isBusy(row.name)"
            @click="install(row.name)"
          >
            {{ row.updateAvailable ? "Обновить" : "Переустановить" }}
          </AppButton>
          <AppButton
            variant="secondary"
            :disabled="isBusy(row.name)"
            @click="remove(row.name)"
          >
            Удалить
          </AppButton>
        </template>
        <span v-else class="dep-card__note" :class="{ 'dep-card__note--update': row.updateAvailable }">
          <template v-if="row.updateAvailable">
            Используется системная копия. Доступно обновление {{ row.latestVersion }} — обновите
            системную версию или выберите «Локальная».
          </template>
          <template v-else>Используется системная копия — установка не требуется.</template>
        </span>
      </div>
    </article>

    <p v-if="rows.length === 0 && !isLoading" class="deps-detail__empty">
      Нет данных о зависимостях.
    </p>
  </div>
</template>

<style scoped>
.deps-detail {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.deps-detail__toolbar {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

.deps-detail__error {
  color: var(--color-danger);
  font-size: 13px;
}

.deps-detail__empty {
  color: var(--color-text-muted);
  font-size: 13px;
}

.dep-card {
  padding: var(--space-4);
  border-radius: var(--radius-md);
  background: var(--color-bg-elevated);
  box-shadow: var(--shadow-sm);
}

.dep-card__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-3);
  margin-bottom: var(--space-3);
}

.dep-card__title {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
}

.dep-card__name {
  font-size: 15px;
  font-weight: 600;
}

.dep-card__status {
  color: var(--color-text-muted);
  font-size: 12px;
}

.dep-card__status--missing {
  color: var(--color-danger);
}

.dep-card__version {
  color: var(--color-text-muted);
  font-variant-numeric: tabular-nums;
}

.dep-fields {
  display: grid;
  grid-template-columns: 140px 1fr;
  gap: var(--space-2) var(--space-3);
  margin: 0 0 var(--space-4);
}

.dep-field {
  display: contents;
}

.dep-field__label {
  color: var(--color-text-muted);
  font-size: 13px;
}

.dep-field__value {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dep-field__value--mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
}

.dep-field__muted {
  color: var(--color-text-muted);
}

.dep-select {
  max-width: 180px;
  padding: var(--space-1) var(--space-2);
  border: 0;
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  box-shadow: var(--shadow-xs);
}

.dep-card__progress {
  margin-bottom: var(--space-3);
}

.dep-card__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.dep-card__note {
  color: var(--color-text-muted);
  font-size: 13px;
}

.dep-card__note--update {
  color: var(--color-accent);
}
</style>
