import type { Dependency } from "@/shared/api";

// Статусы зависимости (синхронизировано с internal/deps).
export const DEPENDENCY_STATUS = {
  missing: "missing",
  system: "system",
  managed: "managed",
} as const;

// Признак доступного обновления: зависимость установлена (системная или управляемая) и её
// свежая версия отличается от текущей. Определяется на бэкенде (с нормализацией версий).
export function isUpdateAvailable(dep: Dependency): boolean {
  return dep.updateAvailable;
}

// Признак управляемой копии — установленной самим приложением. Такие копии можно удалить,
// в отличие от системных.
export function isManaged(dep: Dependency): boolean {
  return dep.status === DEPENDENCY_STATUS.managed;
}

// Признак отсутствующей зависимости — её можно установить.
export function isMissing(dep: Dependency): boolean {
  return dep.status === DEPENDENCY_STATUS.missing;
}

// Признак установленной зависимости (системная или управляемая).
export function isInstalled(dep: Dependency): boolean {
  return dep.status === DEPENDENCY_STATUS.system || dep.status === DEPENDENCY_STATUS.managed;
}

// Варианты политики источника для селектора в интерфейсе.
export const DEPENDENCY_SOURCES: ReadonlyArray<{ value: string; label: string }> = [
  { value: "auto", label: "Авто" },
  { value: "system", label: "Системная" },
  { value: "managed", label: "Локальная" },
];

// Человекочитаемая подпись статуса для интерфейса.
export function dependencyStatusLabel(status: string): string {
  switch (status) {
    case DEPENDENCY_STATUS.managed:
      return "установлена";
    case DEPENDENCY_STATUS.system:
      return "системная";
    case DEPENDENCY_STATUS.missing:
      return "не найдена";
    default:
      return status;
  }
}
