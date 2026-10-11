<script setup lang="ts">
import { ref } from "vue";

import { pickDownloadsDir } from "@/shared/api";
import { AppButton } from "@/shared/ui";

const emit = defineEmits<{ (e: "selected", dir: string): void }>();

const isPicking = ref(false);

// Открывает нативный диалог выбора папки и сообщает наружу выбранный путь.
async function pick(): Promise<void> {
  isPicking.value = true;
  try {
    const dir = await pickDownloadsDir();
    if (dir !== "") {
      emit("selected", dir);
    }
  } finally {
    isPicking.value = false;
  }
}
</script>

<template>
  <AppButton :disabled="isPicking" @click="pick">Выбрать папку</AppButton>
</template>
