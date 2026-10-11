<script setup lang="ts">
// Экран «Прокси» — импорт списком, статусы, стратегия и проверка.
import { storeToRefs } from "pinia";
import { onMounted, ref } from "vue";

import { proxyAddress, proxyStatusLabel, useProxyStore } from "@/entities/proxy";
import type { ImportResult } from "@/shared/api";
import { formatBytes } from "@/shared/lib/format";
import { AppButton } from "@/shared/ui";

const store = useProxyStore();
const { items, strategy, strategies, useProxy, isLoading, error } = storeToRefs(store);

const importText = ref("");
const isImporting = ref(false);
const lastResult = ref<ImportResult | null>(null);

// Человекочитаемые подписи стратегий (id приходят с бэкенда).
const STRATEGY_LABELS: Readonly<Record<string, string>> = {
  "round-robin": "по кругу",
  random: "случайный",
  "least-used": "наименее используемый",
  "lowest-latency": "наименьшая задержка",
  sticky: "закреплённый",
};

function strategyLabel(value: string): string {
  return STRATEGY_LABELS[value] ?? value;
}

onMounted(() => {
  void store.load();
});

async function addProxies(): Promise<void> {
  if (importText.value.trim() === "") {
    return;
  }
  isImporting.value = true;
  lastResult.value = await store.importList(importText.value);
  isImporting.value = false;
  if (lastResult.value !== null && lastResult.value.added > 0) {
    importText.value = "";
  }
}

async function changeStrategy(): Promise<void> {
  await store.setStrategy(strategy.value);
}

// Сохраняет настройку использования пула прокси для загрузок.
async function changeUseProxy(): Promise<void> {
  await store.setUseProxy(useProxy.value);
}

async function toggle(id: string, enabled: boolean): Promise<void> {
  await store.setEnabled(id, enabled);
}
</script>

<template>
  <section class="page">
    <header class="page-header">
      <h1 class="page-title">Прокси</h1>
      <div class="page-actions">
        <AppButton variant="secondary" :disabled="isLoading || items.length === 0" @click="store.check()">
          {{ isLoading ? "Проверяем…" : "Проверить" }}
        </AppButton>
        <AppButton
          variant="secondary"
          :disabled="isLoading || items.length === 0"
          @click="store.clear()"
        >
          Очистить
        </AppButton>
      </div>
    </header>

    <p v-if="error" class="proxies__error">{{ error }}</p>

    <div class="proxies__controls">
      <label class="proxies__toggle">
        <input v-model="useProxy" type="checkbox" @change="changeUseProxy" />
        <span>Использовать прокси</span>
      </label>
      <span class="proxies__toggle-hint">для определения форматов и загрузки</span>
    </div>

    <div class="proxies__import">
      <textarea
        v-model="importText"
        class="proxies__textarea"
        rows="5"
        spellcheck="false"
        placeholder="Вставьте список прокси — по одному в строке, например:
socks5://1.2.3.4:1080
socks5://user:pass@5.6.7.8:1080
9.9.9.9:8080"
      />
      <div class="proxies__import-actions">
        <AppButton :disabled="isImporting || importText.trim() === ''" @click="addProxies">
          {{ isImporting ? "Добавляем…" : "Добавить" }}
        </AppButton>
        <span v-if="lastResult" class="proxies__import-result">
          добавлено {{ lastResult.added }}, пропущено {{ lastResult.skipped }}, всего
          {{ lastResult.total }}
        </span>
      </div>
    </div>

    <div class="proxies__strategy">
      <label class="proxies__strategy-label" for="proxy-strategy">Стратегия выбора</label>
      <select
        id="proxy-strategy"
        v-model="strategy"
        class="proxies__select"
        @change="changeStrategy"
      >
        <option v-for="value in strategies" :key="value" :value="value">
          {{ strategyLabel(value) }}
        </option>
      </select>
    </div>

    <p v-if="items.length === 0 && !isLoading" class="page-hint">Прокси не добавлены.</p>

    <ul v-else class="proxies__list">
      <li
        v-for="proxy in items"
        :key="proxy.id"
        class="proxy"
        :class="{ 'proxy--off': !proxy.enabled }"
      >
        <label class="proxy__toggle">
          <input
            type="checkbox"
            :checked="proxy.enabled"
            @change="toggle(proxy.id, !proxy.enabled)"
          />
        </label>
        <span class="proxy__address" :title="proxyAddress(proxy)">{{ proxyAddress(proxy) }}</span>
        <span class="proxy__status" :class="`proxy__status--${proxy.health}`">
          {{ proxyStatusLabel(proxy) }}
        </span>
        <span v-if="proxy.speedBps > 0" class="proxy__speed">
          ≈ {{ formatBytes(proxy.speedBps) }}/с
        </span>
        <span v-if="proxy.failCount > 0" class="proxy__fails">сбоев: {{ proxy.failCount }}</span>
        <button type="button" class="proxy__remove" @click="store.remove(proxy.id)">Удалить</button>
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

.page-actions {
  display: flex;
  gap: var(--space-2);
}

.proxies__error {
  margin: 0;
  color: var(--color-danger);
  font-size: 13px;
}

.proxies__controls {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.proxies__toggle {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--color-text);
  font-size: 13px;
  cursor: pointer;
  user-select: none;
}

.proxies__toggle-hint {
  color: var(--color-text-muted);
  font-size: 12px;
}

.proxies__import {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.proxies__textarea {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text);
  font: inherit;
  resize: vertical;
}

.proxies__textarea:focus {
  border-color: var(--color-accent);
  outline: none;
}

.proxies__import-actions {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.proxies__import-result {
  color: var(--color-text-muted);
  font-size: 12px;
}

.proxies__strategy {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.proxies__strategy-label {
  color: var(--color-text-muted);
  font-size: 13px;
}

.proxies__select {
  padding: var(--space-1) var(--space-2);
  border: 0;
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text);
  box-shadow: var(--shadow-xs);
}

.proxies__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.proxy {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
}

.proxy--off {
  opacity: 0.55;
}

.proxy__address {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.proxy__status {
  flex: none;
  color: var(--color-text-muted);
  font-size: 12px;
}

.proxy__status--ok {
  color: var(--color-success);
}

.proxy__status--fail {
  color: var(--color-danger);
}

.proxy__fails {
  flex: none;
  color: var(--color-text-muted);
  font-size: 12px;
}

.proxy__speed {
  flex: none;
  color: var(--color-text-muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.proxy__remove {
  flex: none;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  text-decoration: underline;
  text-underline-offset: 2px;
}

.proxy__remove:hover {
  color: var(--color-danger);
}
</style>
