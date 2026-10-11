import { createPinia } from "pinia";
import type { App } from "vue";

// Подключает Pinia к приложению.
export function installPinia(app: App): void {
  app.use(createPinia());
}
