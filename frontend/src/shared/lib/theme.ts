import { computed, ref, watchEffect } from "vue";
import type { ComputedRef, Ref } from "vue";

// Режим темы, выбранный пользователем.
export type ThemeMode = "light" | "dark" | "system";
// Фактически применяемая тема (после разрешения режима "system").
export type ResolvedTheme = "light" | "dark";

const STORAGE_KEY = "loadika.theme";
const THEME_MODES: readonly ThemeMode[] = ["light", "dark", "system"];

// Единственный источник истины для темы (модульный синглтон, без внешних библиотек).
const mode: Ref<ThemeMode> = ref(readStoredMode());

const darkQuery = window.matchMedia("(prefers-color-scheme: dark)");
const systemTheme: Ref<ResolvedTheme> = ref(darkQuery.matches ? "dark" : "light");

// Итоговая тема: при режиме "system" берём системную, иначе — выбранную.
const resolvedTheme: ComputedRef<ResolvedTheme> = computed(() =>
  mode.value === "system" ? systemTheme.value : mode.value,
);

let installed = false;

// Инициализирует тему: следит за системной темой и применяет результат к <html data-theme>.
// Вызывается один раз при старте приложения.
export function installTheme(): void {
  if (installed) {
    return;
  }
  installed = true;

  darkQuery.addEventListener("change", (event) => {
    systemTheme.value = event.matches ? "dark" : "light";
  });

  watchEffect(() => {
    document.documentElement.dataset.theme = resolvedTheme.value;
  });
}

// Доступ к теме: текущий режим, итоговая тема и функция переключения режима.
export function useTheme() {
  function setMode(next: ThemeMode): void {
    mode.value = next;
    localStorage.setItem(STORAGE_KEY, next);
  }

  return { mode, resolvedTheme, setMode } as const;
}

function readStoredMode(): ThemeMode {
  const stored = localStorage.getItem(STORAGE_KEY);
  return isThemeMode(stored) ? stored : "system";
}

function isThemeMode(value: string | null): value is ThemeMode {
  return value !== null && (THEME_MODES as readonly string[]).includes(value);
}
