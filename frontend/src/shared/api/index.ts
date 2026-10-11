import { AppService, DepsService, DownloadService, HistoryService, ProxyService, SettingsService, SystemService } from "@bindings/loadika-desktop/internal/services";
import type { AppInfo, DownloadMeta, ImportResult, Settings } from "@bindings/loadika-desktop/internal/services/models";
import type { Dependency } from "@bindings/loadika-desktop/internal/deps";
import type { Inspect, LogLine as TaskLogLine, QualityOption, Task } from "@bindings/loadika-desktop/internal/download";
import type { Entry as HistoryEntry } from "@bindings/loadika-desktop/internal/history";
import type { View as ProxyView } from "@bindings/loadika-desktop/internal/proxy";
import type { Format } from "@bindings/loadika-desktop/internal/ytdlp";
import type { Browser as CookieBrowser, Profile as CookieProfile } from "@bindings/loadika-desktop/internal/browsers";

// Слой доступа к бэкенду: здесь оборачиваются сгенерированные Wails-биндинги.
// Компоненты и слайсы обязаны использовать этот слой, а не импортировать биндинги напрямую.

// Реэкспорт типов бэкенда для удобства потребителей.
export type {
  AppInfo,
  DownloadMeta,
  ImportResult,
  Settings,
  Dependency,
  Task,
  Inspect,
  QualityOption,
  Format,
  HistoryEntry,
  ProxyView,
  CookieBrowser,
  CookieProfile,
  TaskLogLine,
};

// Подписка на события Wails с автоматической отпиской (см. shared/api/events.ts).
export { onWailsEvent } from "./events";

// Возвращает базовые метаданные приложения.
export function getAppInfo(): Promise<AppInfo> {
  return AppService.GetAppInfo();
}

// Возвращает текущие настройки.
export function getSettings(): Promise<Settings> {
  return SettingsService.GetSettings();
}

// Задаёт папку загрузки.
export function setDownloadsDir(dir: string): Promise<void> {
  return SettingsService.SetDownloadsDir(dir);
}

// Задаёт канал обновления yt-dlp.
export function setYtDlpChannel(channel: string): Promise<void> {
  return SettingsService.SetYtDlpChannel(channel);
}

// Включает/выключает использование пула прокси для загрузок (настройка).
export function setUseProxy(enabled: boolean): Promise<void> {
  return SettingsService.SetUseProxy(enabled);
}

// Открывает нативный диалог выбора папки и возвращает выбранный путь.
// При отмене возвращает текущую папку.
export function pickDownloadsDir(): Promise<string> {
  return SettingsService.PickDownloadsDir();
}

// Задаёт браузер для импорта cookies ((--cookies-from-browser)). Пустая строка — отключить.
export function setCookiesBrowser(browser: string): Promise<void> {
  return SettingsService.SetCookiesBrowser(browser);
}

// Задаёт путь к файлу cookies ((--cookies)). Пустая строка — отключить.
export function setCookiesFile(path: string): Promise<void> {
  return SettingsService.SetCookiesFile(path);
}

// Открывает нативный диалог выбора файла cookies и возвращает выбранный путь.
// При отмене возвращает текущий путь.
export function pickCookiesFile(): Promise<string> {
  return SettingsService.PickCookiesFile();
}

// Возвращает обнаруженные браузеры с сохранёнными cookies и их профили.
export function getCookieBrowsers(): Promise<CookieBrowser[]> {
  return SettingsService.GetCookieBrowsers().then((list) => list ?? []);
}

// Возвращает список зависимостей с локальным статусом (без обращения к сети).
export function getDependencies(): Promise<Dependency[]> {
  return DepsService.GetDependencies().then((list) => list ?? []);
}

// Проверяет свежие версии зависимостей (обращается к сети).
export function checkUpdates(): Promise<Dependency[]> {
  return DepsService.CheckUpdates().then((list) => list ?? []);
}

// Устанавливает или обновляет зависимость по имени.
export function installDependency(name: string): Promise<Dependency> {
  return DepsService.Install(name);
}

// Удаляет управляемую копию зависимости по имени.
export function removeDependency(name: string): Promise<void> {
  return DepsService.Remove(name);
}

// Удаляет все управляемые (скачанные приложением) копии зависимостей,
// не затрагивая системные.
export function removeAllManagedDependencies(): Promise<void> {
  return DepsService.RemoveAllManaged();
}

// Задаёт политику использования источника зависимости (auto | system | managed).
export function setDependencySource(name: string, policy: string): Promise<void> {
  return DepsService.SetSourcePolicy(name, policy);
}

// Запрашивает метаданные ссылки (yt-dlp -J), включая сырой ответ библиотеки (команда + stdout/stderr).
// useProxy — идти ли через пул прокси (значение конкретной загрузки).
export function inspect(url: string, useProxy: boolean): Promise<Inspect> {
  return DownloadService.Inspect(url, useProxy);
}

// Возвращает снимки всех задач загрузки.
export function getDownloadTasks(): Promise<Task[]> {
  return DownloadService.GetTasks().then((list) => list ?? []);
}

// Создаёт и запускает задачу загрузки. format — селектор -f (разрешение/аудио).
// useProxy — идти ли через пул прокси (значение конкретной загрузки); meta — метаданные для истории.
export function createDownload(
  url: string,
  format: string,
  useProxy: boolean,
  meta: DownloadMeta,
): Promise<Task> {
  return DownloadService.Create(url, format, useProxy, meta);
}

// Отменяет выполняющуюся задачу загрузки.
export function cancelDownload(id: string): Promise<void> {
  return DownloadService.Cancel(id);
}

// Ставит задачу на паузу (недокачанный файл сохраняется).
export function pauseDownload(id: string): Promise<void> {
  return DownloadService.Pause(id);
}

// Возвращает задачу с паузы (докачка).
export function resumeDownload(id: string): Promise<void> {
  return DownloadService.Resume(id);
}

// Возвращает журнал событий задачи (для просмотра в любой момент).
export function getTaskJournal(taskID: string): Promise<TaskLogLine[]> {
  return DownloadService.GetTaskJournal(taskID).then((lines) => lines ?? []);
}

// Копирует текст в системный буфер обмена.
export function copyText(text: string): Promise<void> {
  return SystemService.CopyText(text);
}

// Возвращает историю завершённых загрузок (свежие сверху).
export function getHistory(): Promise<HistoryEntry[]> {
  return HistoryService.GetHistory().then((list) => list ?? []);
}

// Очищает историю загрузок.
export function clearHistory(): Promise<void> {
  return HistoryService.ClearHistory();
}

// Удаляет запись истории.
export function removeHistory(id: string): Promise<void> {
  return HistoryService.RemoveHistory(id);
}

// Удаляет запись истории вместе с её файлом.
export function removeHistoryWithFile(id: string): Promise<void> {
  return HistoryService.RemoveHistoryWithFile(id);
}

// Открывает файл или каталог приложением по умолчанию.
export function openPath(path: string): Promise<void> {
  return SystemService.OpenPath(path);
}

// Открывает файловый менеджер с выделением файла (на Linux — содержащий каталог).
export function revealPath(path: string): Promise<void> {
  return SystemService.RevealPath(path);
}

// Возвращает прокси с рантайм-состоянием.
export function getProxies(): Promise<ProxyView[]> {
  return ProxyService.GetProxies().then((list) => list ?? []);
}

// Возвращает допустимые стратегии выбора прокси.
export function getProxyStrategies(): Promise<string[]> {
  return ProxyService.GetStrategies().then((list) => list ?? []);
}

// Импортирует список прокси (по строке на прокси) и возвращает статистику.
export function importProxies(text: string): Promise<ImportResult> {
  return ProxyService.Import(text);
}

// Удаляет прокси по идентификатору.
export function removeProxy(id: string): Promise<void> {
  return ProxyService.Remove(id);
}

// Включает/выключает прокси.
export function setProxyEnabled(id: string, enabled: boolean): Promise<void> {
  return ProxyService.SetEnabled(id, enabled);
}

// Очищает пул прокси.
export function clearProxies(): Promise<void> {
  return ProxyService.Clear();
}

// Задаёт стратегию выбора прокси.
export function setProxyStrategy(strategy: string): Promise<void> {
  return ProxyService.SetStrategy(strategy);
}

// Проверяет доступность прокси и возвращает обновлённый список.
export function checkProxies(): Promise<ProxyView[]> {
  return ProxyService.Check().then((list) => list ?? []);
}
