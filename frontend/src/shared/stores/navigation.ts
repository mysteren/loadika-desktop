// Стор навигации: какой экран активен. Живёт в Pinia, чтобы другие слайсы могли программно
// переключать вкладку (например, «Перекачать» из «Видео» → экран «Загрузки»).
import { defineStore } from "pinia";
import { ref } from "vue";

import { DEFAULT_PAGE_ID } from "@/shared/config/navigation";
import type { PageId } from "@/shared/config/navigation";

export const useNavigationStore = defineStore("navigation", () => {
  const activeId = ref<PageId>(DEFAULT_PAGE_ID);

  // Переключает активный экран.
  function navigate(id: PageId): void {
    activeId.value = id;
  }

  return { activeId, navigate };
});
