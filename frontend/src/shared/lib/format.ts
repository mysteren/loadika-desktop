// Утилиты форматирования. Только чистые функции, без побочных эффектов.

const BYTE_UNITS = ["Б", "КБ", "МБ", "ГБ", "ТБ"] as const;

// Форматирует размер в байтах в человекочитаемую строку.
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return "0 Б";
  }

  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), BYTE_UNITS.length - 1);
  const value = bytes / 1024 ** exponent;
  const unit = BYTE_UNITS[exponent] ?? "Б";

  return `${value.toFixed(exponent === 0 ? 0 : 1)} ${unit}`;
}

// Форматирует ISO-8601 строку как локальные дату и время.
// Пустая или некорректная строка превращается в «—».
export function formatDateTime(iso: string): string {
  if (iso === "") {
    return "—";
  }
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return "—";
  }
  return formatDate(date);
}

// Форматирует метку времени (epoch-ms) как локальные дату и время.
export function formatTimestamp(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) {
    return "—";
  }
  return formatDate(new Date(ms));
}

// Форматирует метку времени (epoch-ms) как локальное время (чч:мм:сс).
export function formatClock(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) {
    return "—";
  }
  return new Date(ms).toLocaleTimeString("ru-RU", { hour12: false });
}

// Форматирует длительность в секундах как «1:23:45» или «3:07». 0 — неизвестно.
export function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) {
    return "";
  }
  const total = Math.floor(seconds);
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;
  const pad = (value: number): string => String(value).padStart(2, "0");
  return hours > 0 ? `${hours}:${pad(minutes)}:${pad(secs)}` : `${minutes}:${pad(secs)}`;
}

function formatDate(date: Date): string {
  return date.toLocaleString("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}
