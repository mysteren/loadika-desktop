<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    variant?: "primary" | "secondary";
    disabled?: boolean;
  }>(),
  {
    variant: "primary",
    disabled: false,
  },
);

const classes = computed(() => ["app-button", `app-button--${props.variant}`]);
</script>

<template>
  <button class="app-button" :class="classes" type="button" :disabled="disabled">
    <slot />
  </button>
</template>

<style scoped>
.app-button {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-2);
  border: 0;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition:
    background-color 0.16s ease,
    box-shadow 0.16s ease,
    transform 0.06s ease;
}

.app-button--primary {
  background: var(--color-accent);
  color: var(--color-accent-contrast);
  box-shadow: var(--shadow-sm);
}

.app-button--primary:hover:not(:disabled) {
  background: var(--color-accent-hover);
  box-shadow: var(--shadow-md);
}

/* Вторичная кнопка приподнята тенью вместо рамки. */
.app-button--secondary {
  background: var(--color-bg-elevated);
  color: var(--color-text);
  box-shadow: var(--shadow-xs);
}

.app-button--secondary:hover:not(:disabled) {
  box-shadow: var(--shadow-sm);
}

.app-button:active:not(:disabled) {
  transform: translateY(1px);
}

.app-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  box-shadow: none;
}
</style>
