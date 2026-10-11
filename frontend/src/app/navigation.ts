import type { Component } from "vue";

import { DownloadsPage } from "@/pages/downloads";
import { HelpPage } from "@/pages/help";
import { HistoryPage } from "@/pages/history";
import { ProxiesPage } from "@/pages/proxies";
import { SettingsPage } from "@/pages/settings";
import type { PageId } from "@/shared/config/navigation";

export interface NavItem {
  id: PageId;
  label: string;
  component: Component;
}

// Реестр экранов: используется сайдбаром (пункты меню) и областью контента.
export const NAV_ITEMS: ReadonlyArray<NavItem> = [
  { id: "downloads", label: "Загрузки", component: DownloadsPage },
  { id: "proxies", label: "Прокси", component: ProxiesPage },
  { id: "history", label: "Видео", component: HistoryPage },
  { id: "settings", label: "Настройки", component: SettingsPage },
  { id: "help", label: "Справка", component: HelpPage },
];
