<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

import { SelectDownloadDirButton } from "@/features/select-download-dir";
import { ThemeSwitcher } from "@/features/theme-switcher";
import {
  getCookieBrowsers,
  getSettings,
  pickCookiesFile,
  setCookiesBrowser,
  setCookiesFile,
  setYtDlpChannel,
} from "@/shared/api";
import type { CookieBrowser, Settings } from "@/shared/api";
import { COOKIES_BROWSERS, YTDLP_CHANNELS, isYtDlpChannel } from "@/shared/config/settings";
import { DepsDetail } from "@/widgets/deps-detail";

// Вкладки настроек. Первой по умолчанию открывается «Основные» — там самые важные опции.
type TabId = "general" | "dependencies";

const TABS: ReadonlyArray<{ id: TabId; label: string }> = [
  { id: "general", label: "Основные" },
  { id: "dependencies", label: "Зависимости" },
];

const activeTab = ref<TabId>("general");

const settings = ref<Settings | null>(null);
const channel = ref<string>("stable");
// Обнаруженные браузеры с cookies и выбранные значения (браузер + профиль).
const detectedBrowsers = ref<CookieBrowser[]>([]);
const browserName = ref<string>("");
const profileName = ref<string>("");

onMounted(async () => {
  const [loaded, detected] = await Promise.all([getSettings(), getCookieBrowsers()]);
  settings.value = loaded;
  channel.value = loaded.ytDlpChannel;
  detectedBrowsers.value = detected;
  [browserName.value, profileName.value] = splitCookieBrowser(loaded.cookiesBrowser);
});

// Браузеры для выпадающего списка: обнаруженные; при пустом определении — резервный список.
// Сохранённое значение добавляем, даже если оно не обнаружено, чтобы не потерять настройку.
const browserOptions = computed<Array<{ value: string; label: string }>>(() => {
  const options = detectedBrowsers.value.map((b) => ({ value: b.name, label: b.label }));
  if (options.length === 0) {
    options.push(...COOKIES_BROWSERS.filter((b) => b.value !== ""));
  }
  if (browserName.value !== "" && !options.some((o) => o.value === browserName.value)) {
    options.unshift({ value: browserName.value, label: browserName.value });
  }
  return options;
});

// Профили выбранного браузера (пусто — браузер не обнаружен или у него нет профилей).
const browserProfiles = computed(
  () => detectedBrowsers.value.find((b) => b.name === browserName.value)?.profiles ?? [],
);

// Разбирает сохранённую строку `браузер[:профиль]`.
function splitCookieBrowser(value: string): [string, string] {
  const index = value.indexOf(":");
  if (index === -1) {
    return [value, ""];
  }
  return [value.slice(0, index), value.slice(index + 1)];
}

// Сохраняет выбор в формате `браузер[:профиль]` (его понимает yt-dlp).
async function saveCookiesChoice(): Promise<void> {
  const value =
    profileName.value === "" ? browserName.value : `${browserName.value}:${profileName.value}`;
  await setCookiesBrowser(value);
  if (settings.value) {
    settings.value = { ...settings.value, cookiesBrowser: value };
  }
}

// Смена браузера сбрасывает профиль: наборы профилей у браузеров разные.
async function changeBrowser(): Promise<void> {
  profileName.value = "";
  await saveCookiesChoice();
}

async function changeProfile(): Promise<void> {
  await saveCookiesChoice();
}

// Смена канала: сохраняем на бэкенде и обновляем локальное состояние.
async function changeChannel(): Promise<void> {
  if (!isYtDlpChannel(channel.value)) {
    return;
  }
  await setYtDlpChannel(channel.value);
  if (settings.value) {
    settings.value = { ...settings.value, ytDlpChannel: channel.value };
  }
}

// Выбор папки уже сохранён на бэкенде — обновляем только отображение.
function onDirSelected(dir: string): void {
  if (settings.value) {
    settings.value = { ...settings.value, downloadsDir: dir };
  }
}

// Выбор файла cookies уже сохранён на бэкенде — обновляем только отображение.
async function chooseCookiesFile(): Promise<void> {
  const path = await pickCookiesFile();
  if (settings.value) {
    settings.value = { ...settings.value, cookiesFile: path };
  }
}

// Сбрасывает выбранный файл cookies.
async function clearCookiesFile(): Promise<void> {
  await setCookiesFile("");
  if (settings.value) {
    settings.value = { ...settings.value, cookiesFile: "" };
  }
}
</script>

<template>
  <section class="page">
    <header class="page-header">
      <h1 class="page-title">Настройки</h1>
    </header>

    <nav class="tabs" role="tablist">
      <button
        v-for="tab in TABS"
        :key="tab.id"
        class="tabs__item"
        :class="{ 'is-active': tab.id === activeTab }"
        type="button"
        role="tab"
        :aria-selected="tab.id === activeTab"
        @click="activeTab = tab.id"
      >
        {{ tab.label }}
      </button>
    </nav>

    <div v-if="activeTab === 'general'" class="settings">
      <div class="settings-group">
        <h2 class="settings-group__title">Загрузки</h2>
        <div class="settings-field">
          <span class="settings-field__label">Папка загрузки</span>
          <div class="settings-field__control">
            <span class="settings-field__value" :title="settings?.downloadsDir">
              {{ settings?.downloadsDir ?? "…" }}
            </span>
            <SelectDownloadDirButton @selected="onDirSelected" />
          </div>
        </div>
      </div>

      <div class="settings-group">
        <h2 class="settings-group__title">Авторизация YouTube</h2>
        <div class="settings-field">
          <label class="settings-field__label" for="cookies-browser">Cookies из браузера</label>
          <div class="settings-field__control">
            <select
              id="cookies-browser"
              v-model="browserName"
              class="settings-select"
              @change="changeBrowser"
            >
              <option value="">Не использовать</option>
              <option v-for="item in browserOptions" :key="item.value" :value="item.value">
                {{ item.label }}
              </option>
            </select>
          </div>
        </div>

        <div v-if="browserProfiles.length > 0" class="settings-field">
          <label class="settings-field__label" for="cookies-profile">Профиль</label>
          <div class="settings-field__control">
            <select
              id="cookies-profile"
              v-model="profileName"
              class="settings-select"
              @change="changeProfile"
            >
              <option value="">По умолчанию (последний использованный)</option>
              <option v-for="item in browserProfiles" :key="item.name" :value="item.name">
                {{ item.label }}
              </option>
            </select>
          </div>
        </div>

        <div class="settings-field">
          <span class="settings-field__label">Файл cookies (формат Netscape)</span>
          <div class="settings-field__control">
            <span class="settings-field__value" :title="settings?.cookiesFile">
              {{ settings?.cookiesFile || "не выбран" }}
            </span>
            <button class="settings-button" type="button" @click="chooseCookiesFile">
              Выбрать…
            </button>
            <button
              v-if="settings?.cookiesFile"
              class="settings-button"
              type="button"
              @click="clearCookiesFile"
            >
              Очистить
            </button>
          </div>
        </div>

        <p class="settings-hint">
          Помогает обойти проверку YouTube «Sign in to confirm you're not a bot». Файл имеет
          приоритет над браузером.
          <template v-if="detectedBrowsers.length > 0">
            Обнаружено браузеров с cookies: {{ detectedBrowsers.length }}.
          </template>
          <template v-else>
            Браузеры с cookies не обнаружены — выберите вручную или укажите файл ниже.
          </template>
        </p>
      </div>

      <div class="settings-group">
        <h2 class="settings-group__title">Оформление</h2>
        <div class="settings-field">
          <span class="settings-field__label">Тема</span>
          <div class="settings-field__control">
            <ThemeSwitcher />
          </div>
        </div>
      </div>
    </div>

    <div v-else class="settings">
      <div class="settings-group">
        <h2 class="settings-group__title">Источник обновлений</h2>
        <div class="settings-field">
          <label class="settings-field__label" for="ytdlp-channel">Канал обновления yt-dlp</label>
          <div class="settings-field__control">
            <select
              id="ytdlp-channel"
              v-model="channel"
              class="settings-select"
              @change="changeChannel"
            >
              <option v-for="item in YTDLP_CHANNELS" :key="item.value" :value="item.value">
                {{ item.label }}
              </option>
            </select>
          </div>
        </div>
      </div>

      <DepsDetail />
    </div>
  </section>
</template>

<style scoped>
.tabs {
  display: flex;
  gap: var(--space-1);
  margin-bottom: var(--space-4);
  border-bottom: 1px solid var(--color-border);
}

.tabs__item {
  padding: var(--space-2) var(--space-3);
  border: 0;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  transition:
    color 0.16s ease,
    border-color 0.16s ease;
}

.tabs__item:hover {
  color: var(--color-text);
}

.tabs__item.is-active {
  border-bottom-color: var(--color-accent);
  color: var(--color-text);
  font-weight: 600;
}

.settings {
  max-width: 640px;
}

.settings-group {
  margin-bottom: var(--space-5);
}

.settings-group__title {
  margin-bottom: var(--space-3);
  color: var(--color-text-muted);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.settings-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.settings-field__label {
  color: var(--color-text-muted);
  font-size: 13px;
}

.settings-field__control {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.settings-field__value {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: var(--color-text);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.settings-select {
  padding: var(--space-1) var(--space-2);
  border: 0;
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text);
  box-shadow: var(--shadow-xs);
}

.settings-button {
  padding: var(--space-1) var(--space-3);
  border: 0;
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text);
  box-shadow: var(--shadow-xs);
  cursor: pointer;
  white-space: nowrap;
}

.settings-button:hover {
  box-shadow: var(--shadow-sm);
}

.settings-hint {
  margin-top: var(--space-2);
  color: var(--color-text-muted);
  font-size: 12px;
  line-height: 1.4;
}
</style>
