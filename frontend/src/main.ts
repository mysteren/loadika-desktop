import { createApp } from "vue";

import App from "@/app/App.vue";
import { installPinia } from "@/app/providers/pinia";
import { installTheme } from "@/shared/lib/theme";

import "@/app/styles/index.css";

// Тему применяем до монтирования, чтобы не было «вспышки» неверной темы.
installTheme();

const app = createApp(App);

installPinia(app);

app.mount("#app");
