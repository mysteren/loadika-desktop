import type { ProxyView } from "@/shared/api";

// Состояния работоспособности прокси (синхронизировано с internal/proxy).
export const PROXY_HEALTH = {
  unknown: "unknown",
  ok: "ok",
  fail: "fail",
} as const;

// Адрес прокси для отображения (без пароля): scheme://host:port.
export function proxyAddress(proxy: ProxyView): string {
  return `${proxy.scheme}://${proxy.host}:${proxy.port}`;
}

// Прокси сейчас в cooldown (после серии сбоев).
export function isCoolingDown(proxy: ProxyView): boolean {
  return proxy.cooldownUntil > Date.now();
}

// Короткая подпись состояния для интерфейса.
export function proxyStatusLabel(proxy: ProxyView): string {
  if (!proxy.enabled) {
    return "выключен";
  }
  if (isCoolingDown(proxy)) {
    return "cooldown";
  }
  switch (proxy.health) {
    case PROXY_HEALTH.ok:
      return proxy.latencyMs > 0 ? `ok · ${proxy.latencyMs} мс` : "ok";
    case PROXY_HEALTH.fail:
      return "ошибка";
    default:
      return "не проверен";
  }
}
