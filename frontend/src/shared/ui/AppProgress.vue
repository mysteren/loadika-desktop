<script setup lang="ts">
import { computed } from "vue";

// value — доля выполнения в диапазоне 0..1.
const props = defineProps<{ value: number }>();

const width = computed(() => `${Math.round(Math.min(Math.max(props.value, 0), 1) * 100)}%`);
</script>

<template>
  <div
    class="app-progress"
    role="progressbar"
    :aria-valuenow="Math.round(Math.min(Math.max(value, 0), 1) * 100)"
    aria-valuemin="0"
    aria-valuemax="100"
  >
    <div class="app-progress__bar" :style="{ width }" />
  </div>
</template>

<style scoped>
.app-progress {
  height: 4px;
  overflow: hidden;
  border-radius: var(--radius-sm);
  background: var(--color-accent-soft);
}

.app-progress__bar {
  height: 100%;
  border-radius: inherit;
  background: var(--color-accent);
  transition: width 0.16s ease;
}
</style>
