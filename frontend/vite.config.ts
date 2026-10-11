import { fileURLToPath, URL } from "node:url";

import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import wails from "@wailsio/runtime/plugins/vite";

// https://vite.dev/config/
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  resolve: {
    alias: {
      // Алиасы синхронизированы с `paths` в tsconfig.json.
      "@": fileURLToPath(new URL("./src", import.meta.url)),
      "@bindings": fileURLToPath(new URL("./bindings", import.meta.url)),
    },
  },
  plugins: [vue(), wails("./bindings")],
});
