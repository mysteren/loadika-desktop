// Публичный API сущности `dependency`.
export { useDependencyStore } from "./model/store";
export { dependencyMeta } from "./model/catalog";
export {
  DEPENDENCY_SOURCES,
  DEPENDENCY_STATUS,
  dependencyStatusLabel,
  isInstalled,
  isManaged,
  isMissing,
  isUpdateAvailable,
} from "./model/selectors";
export { useDependencyEvents } from "./model/events";
