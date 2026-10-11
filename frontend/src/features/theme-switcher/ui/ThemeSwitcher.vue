<script setup lang="ts">
import type { Component } from "vue";

import { useTheme } from "@/shared/lib/theme";
import type { ThemeMode } from "@/shared/lib/theme";
import { IconMonitor, IconMoon, IconSun } from "@/shared/ui";

const { mode, setMode } = useTheme();

const options: ReadonlyArray<{ value: ThemeMode; label: string; icon: Component }> = [
  { value: "light", label: "Светлая", icon: IconSun },
  { value: "dark", label: "Тёмная", icon: IconMoon },
  { value: "system", label: "Системная", icon: IconMonitor },
];
</script>

<template>
  <div class="theme-switcher" role="group" aria-label="Тема оформления">
    <button
      v-for="option in options"
      :key="option.value"
      class="theme-switcher__item"
      :class="{ 'is-active': mode === option.value }"
      type="button"
      :title="option.label"
      :aria-label="option.label"
      :aria-pressed="mode === option.value"
      @click="setMode(option.value)"
    >
      <component :is="option.icon" />
    </button>
  </div>
</template>

<style scoped>
/* Дорожка без рамки: углубление задаётся подложкой, активный сегмент — тенью. */
.theme-switcher {
  display: flex;
  gap: 2px;
  padding: 2px;
  border-radius: var(--radius-sm);
  background: color-mix(in srgb, var(--color-text) 7%, transparent);
}

.theme-switcher__item {
  display: inline-flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  padding: 3px;
  border: 0;
  border-radius: calc(var(--radius-sm) - 2px);
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  transition:
    background-color 0.16s ease,
    color 0.16s ease,
    box-shadow 0.16s ease;
}

/* Компактный размер иконок — дорожка должна умещаться в узком сайдбаре. */
.theme-switcher__item :deep(.app-icon) {
  width: 15px;
  height: 15px;
}

.theme-switcher__item:hover {
  color: var(--color-text);
}

.theme-switcher__item.is-active {
  background: var(--color-accent);
  color: var(--color-accent-contrast);
  box-shadow: var(--shadow-xs);
}
</style>
