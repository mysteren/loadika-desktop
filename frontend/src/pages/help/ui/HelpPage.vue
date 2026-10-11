<script setup lang="ts">
// Страница «Справка» — пользовательское руководство по программе.
// Контент статический; из бэкенда берём только текущую папку загрузки (чтобы показать реальный путь).
import { onMounted, ref } from "vue";

import { getSettings } from "@/shared/api";

// Папка загрузки из настроек — показываем её в разделе «Настройки».
const downloadsDir = ref("");

onMounted(async () => {
  try {
    downloadsDir.value = (await getSettings()).downloadsDir;
  } catch {
    // Не критично для справки — оставляем путь пустым.
    downloadsDir.value = "";
  }
});

// Оглавление: id якорей соответствуют id секций ниже.
const SECTIONS: ReadonlyArray<{ id: string; label: string }> = [
  { id: "overview", label: "Что умеет Loadika" },
  { id: "deps", label: "Зависимости" },
  { id: "download", label: "Как скачать видео" },
  { id: "quality", label: "Выбор качества" },
  { id: "queue", label: "Очередь и пауза" },
  { id: "journal", label: "Журнал и диагностика" },
  { id: "library", label: "Библиотека «Видео»" },
  { id: "proxy", label: "Прокси" },
  { id: "cookies", label: "Авторизация YouTube" },
  { id: "settings", label: "Настройки и данные" },
  { id: "faq", label: "Частые проблемы" },
];
</script>

<template>
  <section class="page">
    <header class="page-header">
      <h1 class="page-title">Справка</h1>
    </header>

    <div class="help">
      <p class="help__lead">
        Loadika — удобная обёртка над <strong>yt-dlp</strong>: скачивает видео и аудио с сайтов,
        которые поддерживает библиотека, помогает выбрать качество и надёжно качает через пул
        прокси. Ниже — как всё настроить и использовать.
      </p>

      <!-- Оглавление: быстрый переход по разделам. -->
      <nav class="help__toc" aria-label="Оглавление">
        <a v-for="section in SECTIONS" :key="section.id" class="help__toc-link" :href="`#${section.id}`">
          {{ section.label }}
        </a>
      </nav>

      <section id="overview" class="help__section">
        <h2 class="help__title">Что умеет Loadika</h2>
        <ul class="help__list">
          <li>Скачивание видео и аудио с YouTube, Vimeo и тысяч других сайтов (силами yt-dlp).</li>
          <li>Выбор качества по <em>фактическим</em> форматам файла — с расшифровкой кодеков и оценкой размера.</li>
          <li>Очередь загрузок: по одной, с паузой, докачкой и отменой.</li>
          <li>Библиотека «Видео»: завершённые загрузки с метаданными и действиями (открыть, перекачать, удалить).</li>
          <li>Пул прокси с проверкой, стратегиями и автоматическим переключением.</li>
          <li>Обход проверки YouTube через cookies вашего браузера.</li>
        </ul>
      </section>

      <section id="deps" class="help__section">
        <h2 class="help__title">Зависимости (первый запуск)</h2>
        <p>
          Для работы нужны внешние инструменты. Их можно установить из приложения:
          <strong>Настройки → Зависимости</strong>.
        </p>
        <ul class="help__list">
          <li><strong>yt-dlp</strong> — обязательно: сама библиотека загрузки.</li>
          <li><strong>deno</strong> — рекомендуется: помогает разбирать сложные сайты (в т.ч. YouTube).</li>
          <li><strong>ffmpeg / ffprobe</strong> — для склейки видео и аудио дорожек и конвертации форматов.</li>
        </ul>
        <p>
          Нажмите «Установить / Обновить» у нужного компонента — приложение скачает файл и проверит
          его по контрольной сумме. Статус зависимостей также виден в футере главной страницы.
        </p>
        <p>
          <strong>Источник</strong> определяет, откуда брать инструмент: <em>auto</em> — системный,
          если найден, иначе скачанный приложением; <em>system</em> — только системный;
          <em>managed</em> — только скачанный приложением. <strong>Канал yt-dlp</strong> (стабильный
          или ночной) выбирается на той же вкладке: ночной чаще содержит свежие исправления для YouTube.
        </p>
      </section>

      <section id="download" class="help__section">
        <h2 class="help__title">Как скачать видео</h2>
        <ol class="help__steps">
          <li>Откройте «Загрузки» и вставьте ссылку на видео.</li>
          <li>При необходимости отметьте <strong>«Использовать прокси»</strong> для этой загрузки
            (по умолчанию — значение из настройки на вкладке «Прокси»).</li>
          <li>Нажмите <strong>«Определить форматы»</strong> — приложение запросит у yt-dlp список
            доступных форматов.</li>
          <li>Появятся кнопки качества и раскрывающийся список форматов (см. следующий раздел).</li>
          <li>Нажмите нужную кнопку качества — задача попадёт в очередь и начнёт скачиваться.</li>
        </ol>
        <p class="help__note">
          Файл сохраняется в папку загрузки (по умолчанию —
          <code class="help__code">{{ downloadsDir || "папка загрузок по умолчанию" }}</code>).
          В имя файла добавляется разрешение, поэтому разные качества одного видео не перезаписывают
          друг друга.
        </p>
      </section>

      <section id="quality" class="help__section">
        <h2 class="help__title">Выбор качества</h2>
        <p>
          Loadika показывает не абстрактный список, а варианты, выведенные из <em>реальных</em>
          форматов конкретного видео:
        </p>
        <ul class="help__list">
          <li><strong>Кнопки качества</strong> — «Лучшее», конкретные разрешения (1080p, 720p…),
            отдельно «Только аудио». На кнопке указана оценка размера («≈ …»), в подсказке — кодеки.</li>
          <li><strong>«Доступные форматы»</strong> — развёрнутая таблица: вид (видео/аудио), контейнер,
            расшифровка кодеков, fps, битрейт и размер. Помогает понять, что именно вы выбираете.</li>
          <li><strong>«Диагностика (ответ yt-dlp)»</strong> — команда и сырой ответ библиотеки.
            Пригодится, чтобы понять причину, если что-то не работает.</li>
        </ul>
        <p class="help__note">
          Если размер не показан — он неизвестен до начала загрузки. Формат «Лучшее» обычно даёт
          оптимальное сочетание видео и аудио.
        </p>
      </section>

      <section id="queue" class="help__section">
        <h2 class="help__title">Очередь, пауза и докачка</h2>
        <p>Загрузки выполняются последовательно — по одной. Это стабильнее для сайта, сети и прокси.</p>
        <ul class="help__list">
          <li><strong>Пауза</strong> — останавливает загрузку; уже скачанное сохраняется.</li>
          <li><strong>Докачать</strong> — продолжает с места остановки (файл не скачивается заново).</li>
          <li><strong>Отменить</strong> — прекращает задачу.</li>
        </ul>
        <p>
          В шапке списка «Задачи» видна сводка («1 загружается · 2 в очереди · 1 на паузе»).
          Можно поставить одну загрузку на паузу и запустить другую — очередь подхватит её.
        </p>
      </section>

      <section id="journal" class="help__section">
        <h2 class="help__title">Журнал и диагностика</h2>
        <ul class="help__list">
          <li><strong>«Журнал загрузки»</strong> в строке задачи — что происходило: попытки, выбранный
            и число использованных прокси, переключения, ошибки, итог. Кнопка «Копировать» копирует
            журнал целиком.</li>
          <li>У завершившихся с ошибкой есть кнопка <strong>«Логи»</strong> — откроет полный лог
            команды yt-dlp.</li>
          <li>По каждому новому определению форматов блок «Диагностика» обновляется.</li>
        </ul>
        <p class="help__note">
          Если что-то идёт не так — откройте журнал и скопируйте его: там видно, на чём именно
          споткнулась загрузка.
        </p>
      </section>

      <section id="library" class="help__section">
        <h2 class="help__title">Библиотека «Видео»</h2>
        <p>
          Вкладка «Видео» — это список завершённых загрузок с заголовком, качеством, контейнером,
          длительностью, размером, кодеками и датой. Действия:
        </p>
        <ul class="help__list">
          <li><strong>Открыть</strong> — запустить файл приложением/плеером по умолчанию.</li>
          <li><strong>В папке</strong> — показать файл в файловом менеджере.</li>
          <li><strong>Перекачать</strong> — выбрать другое качество (форматы определятся прямо в
            карточке, задача уйдёт в «Загрузки»).</li>
          <li><strong>Удалить запись</strong> — убрать из библиотеки; <strong>Удалить с файлом</strong>
            — удалить и сам файл (с подтверждением).</li>
        </ul>
      </section>

      <section id="proxy" class="help__section">
        <h2 class="help__title">Прокси</h2>
        <p>
          Прокси помогают обойти блокировки и региональные ограничения, повысить надёжность и
          скорость загрузки. Всё управление — на вкладке «Прокси».
        </p>

        <h3 class="help__subtitle">Импорт списка</h3>
        <p>Вставьте список — по одному прокси в строке; приложение разберёт его автоматически. Поддерживаются форматы:</p>
        <ul class="help__list">
          <li><code class="help__code">scheme://user:pass@host:port</code></li>
          <li><code class="help__code">host:port</code></li>
          <li><code class="help__code">host:port:user:pass</code></li>
        </ul>
        <p>
          Схемы: <code class="help__code">http</code>, <code class="help__code">https</code>,
          <code class="help__code">socks4</code>, <code class="help__code">socks4a</code>,
          <code class="help__code">socks5</code>, <code class="help__code">socks5h</code>.
        </p>
        <p class="help__note">
          Для SOCKS5 предпочитайте <code class="help__code">socks5h://</code> — тогда имена сайтов
          разрешает сам прокси (DNS на его стороне). Это часто решает проблемы с подключением.
        </p>

        <h3 class="help__subtitle">Управление и стратегии</h3>
        <ul class="help__list">
          <li>Каждый прокси можно включить/выключить, удалить; есть статус, задержка и измеренная скорость.</li>
          <li><strong>«Проверить»</strong> — проверить доступность всех включённых прокси.</li>
          <li><strong>Стратегия выбора</strong>: по кругу, случайный, реже используемый, наименьшая
            задержка, фиксированный (sticky).</li>
        </ul>

        <h3 class="help__subtitle">Две настройки — глобальная и для загрузки</h3>
        <ul class="help__list">
          <li><strong>Глобальная</strong> — переключатель «Использовать прокси» на вкладке «Прокси».
            Задаёт значение по умолчанию.</li>
          <li><strong>Для конкретной загрузки</strong> — галочка «Использовать прокси» в панели
            «Новая загрузка». Удобно, когда часть загрузок должна идти напрямую.</li>
        </ul>

        <h3 class="help__subtitle">Автоматика</h3>
        <ul class="help__list">
          <li>Перед определением качества и загрузкой приложение быстро проверяет прокси и отсеивает
            нерабочие — чтобы не ждать их таймаут.</li>
          <li>При сетевых сбоях, таймаутах и слишком низкой скорости загрузка переключается на другой
            прокси и продолжается (недокачанный файл докачивается).</li>
        </ul>
      </section>

      <section id="cookies" class="help__section">
        <h2 class="help__title">Авторизация YouTube (cookies)</h2>
        <p>
          Если YouTube просит «Sign in to confirm you're not a bot» или видео недоступно, укажите
          cookies своего браузера: <strong>Настройки → Основные → Авторизация YouTube</strong>.
        </p>
        <ul class="help__list">
          <li><strong>Cookies из браузера</strong> — выберите браузер: приложение само найдёт
            установленные браузеры. Если в браузере несколько профилей, выберите нужный профиль — тот,
            в котором вы вошли в YouTube.</li>
          <li><strong>Файл cookies</strong> — файл в формате Netscape. Он имеет приоритет над
            браузером.</li>
        </ul>
        <p class="help__note">
          Совет: войдите в YouTube в браузере, где вы обычно смотрите видео, и выберите тот же профиль.
          Если в браузере несколько профилей, а нужный не выбран, cookies могут взяться не оттуда.
        </p>
      </section>

      <section id="settings" class="help__section">
        <h2 class="help__title">Настройки и данные</h2>
        <ul class="help__list">
          <li><strong>Основные</strong> — папка загрузки, авторизация YouTube, тема оформления.</li>
          <li><strong>Зависимости</strong> — канал yt-dlp и установка/обновление/удаление компонентов.</li>
        </ul>
        <p class="help__note">
          Настройки, список прокси, история загрузок и логи команд хранятся в каталоге конфигурации
          вашей ОС (например, <code class="help__code">~/.config/loadika-desktop</code> на Linux).
          Логи команд — в подкаталоге <code class="help__code">logs</code>.
        </p>
      </section>

      <section id="faq" class="help__section">
        <h2 class="help__title">Частые проблемы</h2>

        <details class="help__faq">
          <summary>YouTube пишет «Sign in to confirm you're not a bot»</summary>
          <p>
            Настройте cookies браузера (раздел «Авторизация YouTube») и при необходимости включите
            прокси. Часто достаточно выбрать правильный профиль браузера, где вы вошли в YouTube.
          </p>
        </details>

        <details class="help__faq">
          <summary>«Нет доступных прокси (все выключены или в cooldown)»</summary>
          <p>
            Откройте вкладку «Прокси», включите рабочие прокси и нажмите «Проверить». Нерабочие узлы
            приложение на время помещает в «карантин» (cooldown), пока они снова не понадобятся.
          </p>
        </details>

        <details class="help__faq">
          <summary>Загрузка идёт очень медленно</summary>
          <p>
            Включите прокси или смените стратегию выбора. Loadika сама попытается переключиться на
            более быстрый прокси, если скорость долго держится низкой.
          </p>
        </details>

        <details class="help__faq">
          <summary>Определение форматов зависло или прервалось</summary>
          <p>
            Скорее всего, не ответил прокси — приложение переключится на другой. Попробуйте определить
            форматы ещё раз; блок «Диагностика (ответ yt-dlp)» покажет, что именно произошло.
          </p>
        </details>

        <details class="help__faq">
          <summary>Не видно, что ответила библиотека</summary>
          <p>
            Откройте блок <strong>«Диагностика (ответ yt-dlp)»</strong> при определении форматов и
            <strong>«Журнал загрузки»</strong> у самой задачи — там команда, вывод и все события.
          </p>
        </details>
      </section>
    </div>
  </section>
</template>

<style scoped>
.help {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  max-width: 760px;
}

.help__lead {
  color: var(--color-text-muted);
  line-height: 1.6;
}

.help__toc {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-elevated);
}

.help__toc-link {
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  background: var(--color-accent-soft);
  color: var(--color-text);
  font-size: 12px;
  text-decoration: none;
}

.help__toc-link:hover {
  text-decoration: underline;
}

.help__section {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  scroll-margin-top: var(--space-4);
}

.help__title {
  font-size: 16px;
  font-weight: 600;
}

.help__subtitle {
  margin-top: var(--space-2);
  color: var(--color-text-muted);
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.help__section p {
  margin: 0;
  line-height: 1.6;
}

.help__list,
.help__steps {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
  padding-left: var(--space-4);
  line-height: 1.6;
}

.help__steps {
  padding-left: var(--space-5);
}

.help__note {
  padding: var(--space-3);
  border-left: 3px solid var(--color-accent);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-muted);
  font-size: 13px;
}

.help__code {
  padding: 1px 5px;
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  font-size: 12px;
}

.help__faq {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
}

.help__faq + .help__faq {
  margin-top: var(--space-2);
}

.help__faq summary {
  cursor: pointer;
  font-weight: 600;
  user-select: none;
}

.help__faq p {
  margin-top: var(--space-2);
  color: var(--color-text-muted);
}
</style>
