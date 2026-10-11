<script setup lang="ts">
// Компонент `FormatTable` — детальный список форматов с расшифровкой кодеков.
// Служит подсказкой при выборе: качество, вид потока, кодеки, размер.
import { computed } from "vue";

import type { Format } from "@/shared/api";
import { formatBytes } from "@/shared/lib/format";

const props = defineProps<{ formats: Format[] }>();

// Подписи видов потока (согласовано с internal/ytdlp: muxed/video/audio).
const KIND_LABELS: Readonly<Record<string, string>> = {
  muxed: "видео+аудио",
  video: "видео",
  audio: "аудио",
};

// Сортируем: видео/мультиплекс — по высоте и частоте кадров, затем аудио — по битрейту.
const rows = computed(() => [...props.formats].sort(compare));

function compare(a: Format, b: Format): number {
  const groupA = a.kind === "audio" ? 1 : 0;
  const groupB = b.kind === "audio" ? 1 : 0;
  if (groupA !== groupB) {
    return groupA - groupB;
  }
  if (groupA === 0) {
    if (b.height !== a.height) {
      return b.height - a.height;
    }
    if (b.fps !== a.fps) {
      return b.fps - a.fps;
    }
    return b.totalBitrate - a.totalBitrate;
  }
  return b.audioBitrate - a.audioBitrate;
}

function kindLabel(kind: string): string {
  return KIND_LABELS[kind] ?? "прочее";
}

function codecs(format: Format): string {
  const list = [format.videoCodec, format.audioCodec].filter((codec) => codec !== "");
  return list.join(" / ") || "—";
}

function size(format: Format): string {
  return format.filesize > 0 ? formatBytes(format.filesize) : "—";
}
</script>

<template>
  <div class="formats">
    <div class="formats__head">
      <span>Качество</span>
      <span>Вид</span>
      <span>Кодеки</span>
      <span>Размер</span>
      <span>ID</span>
    </div>
    <ul class="formats__list">
      <li v-for="format in rows" :key="format.id" class="formats__row">
        <span class="formats__quality">{{ format.quality || "—" }}</span>
        <span class="formats__kind" :class="`formats__kind--${format.kind || 'other'}`">
          {{ kindLabel(format.kind) }}
        </span>
        <span class="formats__codecs">{{ codecs(format) }}</span>
        <span class="formats__size">{{ size(format) }}</span>
        <span class="formats__id">{{ format.id }}</span>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.formats {
  font-size: 12px;
}

.formats__head,
.formats__row {
  display: grid;
  grid-template-columns: 92px 92px 1fr 88px 72px;
  gap: var(--space-2);
  align-items: center;
  padding: var(--space-1) var(--space-2);
}

.formats__head {
  color: var(--color-text-muted);
  font-size: 11px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.formats__list {
  margin: 0;
  padding: 0;
  list-style: none;
}

.formats__row {
  border-radius: var(--radius-sm);
  font-variant-numeric: tabular-nums;
}

.formats__row:nth-child(odd) {
  background: var(--color-bg-elevated);
}

.formats__quality {
  color: var(--color-text);
  font-weight: 600;
}

.formats__kind {
  color: var(--color-text-muted);
}

.formats__kind--muxed {
  color: var(--color-success);
}

.formats__kind--audio {
  color: var(--color-accent);
}

.formats__codecs {
  color: var(--color-text-muted);
}

.formats__size,
.formats__id {
  color: var(--color-text-muted);
}

.formats__id {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
