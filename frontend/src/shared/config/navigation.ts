// Идентификаторы экранов приложения. Живут в shared, чтобы их могли использовать и реестр
// навигации (app), и стор навигации, доступный страницам (FSD: импорт строго вниз).

// Идентификаторы экранов приложения.
export type PageId = "downloads" | "proxies" | "history" | "settings" | "help";

// Экран, открываемый при старте.
export const DEFAULT_PAGE_ID: PageId = "downloads";
