<script setup lang="ts">
import { AddDownloadPanel } from "@/features/add-download";
import { DepsStatus } from "@/widgets/deps-status";
import { DownloadList } from "@/widgets/download-list";
</script>

<template>
  <section class="page">
    <header class="page-header">
      <h1 class="page-title">Загрузки</h1>
    </header>

    <section class="block">
      <h2 class="block__title">Новая загрузка</h2>
      <AddDownloadPanel />
    </section>

    <DownloadList class="downloads-list" />

    <!-- Зависимости — ненавязчиво, в футере страницы. -->
    <footer class="downloads-footer">
      <DepsStatus />
    </footer>
  </section>
</template>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-height: 100%;
}

.block {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.block__title {
  margin: 0;
  color: var(--color-text-muted);
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.downloads-list {
  flex: 1;
}

.downloads-footer {
  margin-top: auto;
  padding-top: var(--space-4);
  border-top: 1px solid var(--color-border);
}
</style>
