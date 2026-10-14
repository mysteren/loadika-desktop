<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

import { NAV_ITEMS } from "@/app/navigation";
import { useDependencyEvents } from "@/entities/dependency";
import { useDownloadTaskEvents } from "@/entities/download-task";
import { useProxyEvents } from "@/entities/proxy";
import { ThemeSwitcher } from "@/features/theme-switcher";
import { getAppInfo } from "@/shared/api";
import type { AppInfo } from "@/shared/api";
import { useNavigationStore } from "@/shared/stores/navigation";

// Подписки на события (зависимости, задачи, прокси) живут вместе с приложением.
useDependencyEvents();
useDownloadTaskEvents();
useProxyEvents();

const appInfo = ref<AppInfo | null>(null);

// Активный экран и соответствующий компонент берём из реестра навигации (состояние — в сторе).
const nav = useNavigationStore();
const activePage = computed(() => NAV_ITEMS.find((item) => item.id === nav.activeId)?.component);

// Метаданные приложения грузим один раз при монтировании — заодно проверяем связку с Go.
onMounted(async () => {
  appInfo.value = await getAppInfo();
});
</script>

<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <span class="brand-logo" aria-hidden="true" />
        <span class="brand-name">Лоадика</span>
      </div>

      <nav class="nav">
        <button
          v-for="item in NAV_ITEMS"
          :key="item.id"
          class="nav-item"
          :class="{ 'is-active': item.id === nav.activeId }"
          type="button"
          @click="nav.navigate(item.id)"
        >
          {{ item.label }}
        </button>
      </nav>

      <div class="sidebar-bottom">
        <ThemeSwitcher />
        <div v-if="appInfo" class="sidebar-meta">
          {{ appInfo.name }} {{ appInfo.version }} · {{ appInfo.platform }}
        </div>
      </div>
    </aside>

    <main class="content">
      <component :is="activePage" v-if="activePage" />
    </main>
  </div>
</template>

<style scoped>
.app-shell {
  display: grid;
  grid-template-columns: var(--sidebar-width) 1fr;
  height: 100%;
}

/* Отделяем сайдбар мягкой тенью, а не жёсткой линией. */
.sidebar {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  background: var(--color-sidebar);
  box-shadow: var(--shadow-panel);
}

.brand {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-3);
  padding-top: calc(var(--space-4));
}

.brand-logo {
  width: 48px;
  height: 48px;
  background-color: var(--color-accent);
  /* Знак отрисовывается маской, поэтому наследует primary-цвет и меняется вместе с темой. */
  -webkit-mask: url("/logo.svg") center / contain no-repeat;
  mask: url("/logo.svg") center / contain no-repeat;
}

.brand-name {
  font-size: 20px;
  font-weight: 600;
}

.nav {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
}

.nav-item {
  padding: var(--space-2) var(--space-3);
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
  transition:
    background-color 0.16s ease,
    color 0.16s ease,
    box-shadow 0.16s ease;
}

.nav-item:hover {
  background: var(--color-accent-soft);
}

.nav-item.is-active {
  background: var(--color-accent);
  color: var(--color-accent-contrast);
  box-shadow: var(--shadow-active);
}

.sidebar-bottom {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin-top: auto;
  padding: var(--space-3);
}

.sidebar-meta {
  color: var(--color-text-muted);
  font-size: 12px;
}

.content {
  overflow: auto;
}
</style>
