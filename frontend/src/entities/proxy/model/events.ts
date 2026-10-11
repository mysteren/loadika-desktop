import { onWailsEvent } from "@/shared/api";
import { EVENTS } from "@/shared/config/events";
import type { ProxyHealth } from "@/shared/config/events";

import { useProxyStore } from "./store";

// Подписка стора на результаты проверки прокси. Вызывать один раз в корне приложения.
export function useProxyEvents(): void {
  const store = useProxyStore();
  onWailsEvent<ProxyHealth>(EVENTS.proxyHealth, (payload) => store.applyHealth(payload));
}
