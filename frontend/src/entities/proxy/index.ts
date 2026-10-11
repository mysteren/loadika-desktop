// Публичный API сущности `proxy`.
export { useProxyStore } from "./model/store";
export { useProxyEvents } from "./model/events";
export { PROXY_HEALTH, isCoolingDown, proxyAddress, proxyStatusLabel } from "./model/selectors";
