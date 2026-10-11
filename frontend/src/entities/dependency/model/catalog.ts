// Статичные сведения об известных зависимостях — для отображения «назначения»
// и «источника» в детальном блоке. Рантайм-данные (версия, путь) приходят из бэкенда.
interface DependencyMeta {
  title: string; // человекочитаемое имя
  purpose: string; // назначение
  source: string; // источник скачивания
}

const CATALOG: Record<string, DependencyMeta> = {
  "yt-dlp": {
    title: "yt-dlp",
    purpose: "Скачивание видео и аудио — основной движок загрузок.",
    source: "GitHub · yt-dlp/yt-dlp",
  },
  deno: {
    title: "deno",
    purpose: "JS-рантайм для yt-dlp-ejs — извлечение части видео (YouTube).",
    source: "GitHub · denoland/deno",
  },
  ffmpeg: {
    title: "ffmpeg",
    purpose: "Склейка видео и аудио, извлечение аудио и постобработка (вместе с ffprobe).",
    source: "GitHub · ffmpeg-static (Windows ARM64 — yt-dlp/FFmpeg-Builds)",
  },
};

// Возвращает сведения о зависимости; для неизвестной — нейтральные значения по умолчанию.
export function dependencyMeta(name: string): DependencyMeta {
  const known = CATALOG[name];
  if (known !== undefined) {
    return known;
  }
  return { title: name, purpose: "—", source: "—" };
}
