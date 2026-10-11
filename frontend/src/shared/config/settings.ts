// Типы и значения настроек, разделяемые между слоями.

// Канал обновления yt-dlp (синхронизировано с internal/config).
export type YtDlpChannel = "stable" | "nightly" | "master";

export const YTDLP_CHANNELS: ReadonlyArray<{ value: YtDlpChannel; label: string }> = [
  { value: "stable", label: "Stable" },
  { value: "nightly", label: "Nightly" },
  { value: "master", label: "Master" },
];

// Проверяет, что строка — допустимый канал обновления.
export function isYtDlpChannel(value: string): value is YtDlpChannel {
  return YTDLP_CHANNELS.some((channel) => channel.value === value);
}

// Резервный список браузеров для ручного выбора, когда автопределение ничего не нашло.
// В обычном режиме список формируется из обнаруженных браузеров (`getCookieBrowsers`).
// Пустое значение — не брать cookies из браузера.
export const COOKIES_BROWSERS: ReadonlyArray<{ value: string; label: string }> = [
  { value: "", label: "Не использовать" },
  { value: "firefox", label: "Firefox" },
  { value: "chrome", label: "Chrome" },
  { value: "chromium", label: "Chromium" },
  { value: "brave", label: "Brave" },
  { value: "edge", label: "Edge" },
  { value: "opera", label: "Opera" },
  { value: "vivaldi", label: "Vivaldi" },
  { value: "whale", label: "Whale" },
  { value: "safari", label: "Safari" },
];
