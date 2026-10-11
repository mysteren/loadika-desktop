import { Events } from "@wailsio/runtime";
import { onScopeDispose } from "vue";

// Подписка на событие Wails с автоматической отпиской при уничтожении реактивного
// скоупа (компонента). Так исключаются утечки подписок. Возвращает функцию отписки,
// если нужно снять обработчик раньше срока.
export function onWailsEvent<TData>(name: string, callback: (data: TData) => void): () => void {
  const off = Events.On(name, (event) => callback(event.data as TData));
  onScopeDispose(off);
  return off;
}
