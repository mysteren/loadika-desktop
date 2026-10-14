# 08. Текущий статус и продолжение работы

Документ для передачи работы в новую сессию. Обновляйте его в конце каждого этапа.

## Кратко

Проект **Loadika Desktop** (Wails v3 + Go + Vue 3/TS strict по FSD) — **этапы 0–6 завершены**,
всё собирается, тесты проходят. **Этап 3** — зависимости (`deno`, `ffmpeg`, UI, события `dep:*`).
**Этап 4** — от аргументов до UI: `ytdlp` (аргументы, запуск, прогресс,
`-J` с качеством/кодеками), домен `download`, `DownloadService` + события `task:*`,
`QualityOptions` (кнопки по фактическим разрешениям с оценкой размера), а на фронте —
стор `download-task`, панель добавления с кнопками качества и раскрывающимся списком форматов,
список задач с прогрессом/отменой, открытие результата (файл/папка); логи команды пишутся в файл.
**Этап 5 завершён**: история завершённых загрузок (`internal/history`, `HistoryService`, экран
«Видео») и **очередь загрузок** — строго последовательный запуск (`concurrency = 1`), состояния
`queued`/`running`/`paused`, **пауза/докачка** (`Pause`/`Resume`, недокачанный файл сохраняется),
«запустить другую» (пауза освобождает слот), сводка и группировка задач в UI. **Этап 6 завершён**
(прокси): модель/импорт/персистенция, пул и стратегии ротации, health-check, автопереключение
с retry (классификация ошибок + новый `--proxy`) и UI (`ProxyService`, экран «Прокси»).
Осталось: этапы 7–9.

## Команды

```sh
# Бэкенд
go build . ./internal/...        # сборка (НЕ `go build ./...` — см. «Ограничения»)
go vet . ./internal/...
go test ./internal/...           # тесты (config, deps)

# Фронтенд
cd frontend && npm install
npm run typecheck                # vue-tsc
npm run build                    # vue-tsc + vite (обновляет frontend/dist)

# Биндинги Go → TS (после изменения сервисов/DTO)
wails3 generate bindings -clean=true -ts -i

# Запуск GUI (нужны системные зависимости — см. «Ограничения»)
wails3 dev
```

## Ограничения окружения (важно)

- Полноценный `wails3 dev`/`wails3 build` в текущем окружении **не запускался**: нет `webkit2gtk-4.1`
  (и `task`). Для запуска GUI на Linux нужно установить `libwebkit2gtk-4.1-dev` (+ обычно `libgtk-3-dev`,
  `libsoup-3.0-dev`) и [Task](https://taskfile.dev).
- `go build ./...` спотыкается о `build/ios` (шаблон Wails с build-тегами). Используйте
  `go build . ./internal/...`. Аналогично `gofmt -l internal` (а не `gofmt -l .`, который задевает
  файлы скаффолда в `build/ios`).

## Статус этапов

| Этап | Статус | Ключевое |
|------|--------|----------|
| 0. Фундамент | ✅ | `docs/`, скилы, чистка скаффолда, TS strict, FSD-каркас, токены |
| 1. Каркас приложения | ✅ | реестр навигации, экраны-заглушки, темы (light/dark/system), иконки |
| 2. Настройки и пути | ✅ | `paths`, `config` (JSON, атомарная запись), `SettingsService`, экран настроек. + `useProxy` (использовать пул прокси) и `cookiesBrowser`/`cookiesFile` (обход проверки YouTube) |
| 3.1a–3.1d. Зависимости (yt-dlp) | ✅ | `deps`: чистые функции, сеть+SHA-256, манифест+`Manager`, `DepsService`, биндинги, `shared/api` |
| 3.1e. Стор `dependency` | ✅ | `entities/dependency`: Pinia-стор (`refresh`/`checkUpdates`/`install`/`remove`/`removeManaged`, `hasUpdates`) + селекторы |
| 3.1f. Чеклист на главной | ✅ | виджет `deps-status`: inline-чипсы в футере главной (галочка у установленных, версия, «Проверить») |
| 3.1g. Детальный блок | ✅ | виджет `deps-detail` на вкладке «Зависимости» в «Настройках»: назначение/источник/путь/обновлено/свежая версия + действия |
| 3.2. `deno` | ✅ | `spec`-реестр зависимостей + распаковка zip; deno: системный → скачивание. `--js-runtimes` — в Этапе 4 |
| 3.3. `ffmpeg`/`ffprobe` | ✅ | модель «компонентов» (несколько файлов, gzip/zip, сумма через `digest`); источники: ffmpeg-static (Win arm64 — FFmpeg-Builds) |
| 3.4. События `dep:*` и источники | ✅ | прогресс установки (`dep:progress`/`dep:state`, троттлинг); политика источника `auto`/`system`/`managed` (селектор в детальном блоке) |
| 4. Одиночная загрузка | ✅ | добавление URL, кнопки качества по фактическим разрешениям (с размером), раскрывающийся список форматов с кодеками, прогресс/статус, отмена; события `task:*`. Состояние панели в Pinia — переживает переключение вкладок |
| 5. История и очередь | ✅ | история завершённых загрузок (`internal/history`, `HistoryService`, экран «Видео»); **очередь**: строго последовательный запуск (`concurrency = 1`), состояния `queued`/`running`/`paused`, пауза/докачка (`Pause`/`Resume`), сводка и группировка задач в UI |
| 6. Прокси | ✅ | модель/импорт, пул/стратегии, health-check, автопереключение с retry, `ProxyService` + UI |
| 7. Пресеты | ➖ | не делаем (пользователь закрыл тему; качество/форматы выбираются в панели загрузки) |
| 8. Полировка | ⏳ | виртуализация списков, дружелюбные ошибки, локализация |
| 9. Упаковка и поставка | ✅ | метаданные и версия приведены в порядок, `LICENSE` (MIT); пакеты Linux/Windows/macOS; **CI-релиз** по тегу (`.github/workflows/release.yml`), гайд `docs/BUILD-AND-RELEASE.md`; подпись — позже |

## Ключевые файлы

**Backend (Go):**
- `main.go`, `internal/app/app.go` — точка входа и сборка приложения (регистрация сервисов).
- `internal/paths/paths.go` — каталоги ОС (`ConfigDir`, `ConfigFile`, `HistoryFile`, `DepsDir`, `LogsDir`, `DefaultDownloadsDir`).
- `internal/config/config.go` — настройки (`config.json`, дефолты, атомарная запись; `cookiesBrowser`/`cookiesFile` — авторизация YouTube) + `config_test.go`.
- `internal/deps/` — менеджер зависимостей:
  - `deps.go` (типы, статусы, имена, каналы), `spec.go` (реестр `spec` + модель «компонентов»: `yt-dlp`, `deno`, `ffmpeg`),
  - `ytdlp.go` / `deno.go` / `ffmpeg.go` (описания зависимостей), `sums.go` (`parseSums`, `parseSHA256`),
  - `github.go` (релиз по репозиторию + `digest` ассетов), `download.go` (скачивание+SHA-256),
  - `manifest.go`, `install.go` (установка, распаковка gzip/zip), `manager.go` (`Inspect`/`InspectAll`/`CheckUpdates`/`Install`/`Remove`/`RemoveAll`, `Resolve` — путь без запуска `--version`), `*_test.go`.
- `internal/services/` — `app_service.go`, `settings_service.go`, `deps_service.go` (эмиссия `dep:*`), `download_service.go` (`Inspect(url, useProxy)`/`Create(url, format, useProxy, meta)`/`GetTasks`/`Cancel`/`GetTaskJournal` — использование прокси задаётся конкретной загрузкой, `meta` — заголовок/качество для истории, cookies берётся из настроек, резолв путей через `deps.Resolve` (без запуска процессов), предварительная проверка прокси, запасной заголовок со страницы (`pagetitle`) если yt-dlp не дал, эмиссия `task:*`/`task:log` с троттлингом, запись в историю по завершении с реальным размером файла), `history_service.go` (`GetHistory`/`ClearHistory`/`RemoveHistory`/`RemoveHistoryWithFile` — удаление записи и записи вместе с файлом), `system_service.go` (`OpenPath`/`RevealPath`/`CopyText` — буфер обмена для кнопки «Копировать»), `proxy_service.go` (`GetProxies`/`Import`/`Remove`/`SetEnabled`/`Clear`/`SetStrategy`/`Check`, эмиссия `proxy:health`), `throttle.go`.
- `internal/shell/shell.go` — открытие файлов/каталогов средствами ОС (`Open`, `Reveal`) без зависимости от Wails. На Windows — через `ShellExecuteW` (`shell_windows.go`): файл открывается приложением по умолчанию, «в папке» — нужный каталог (прежние `rundll32 url.dll,FileProtocolHandler` и `explorer /select,<path>` ломались на путях с пробелами).
- `internal/procattr/procattr.go` — атрибуты запуска дочерних процессов: на Windows `HideConsole()` ставит `CREATE_NO_WINDOW`, чтобы yt-dlp/ffmpeg/deno не открывали окна терминала.
- `internal/procenv/procenv.go` — «чистое» окружение для внешних инструментов: внутри AppImage (`APPDIR`) убирает `PYTHONHOME`/`PYTHONPATH`/`LD_LIBRARY_PATH`, которые ломают Python-`yt-dlp` («Failed to import encodings module») + тест.
- `internal/proxy/` — пул прокси: `proxy.go` (`Proxy` с `URL()`/`Display()` без пароля, стабильный `ID`; `URL()` для `socks5` выдаёт `socks5h` — remote DNS; `DialContext` — туннель через SOCKS5 для HTTP-запросов), `parse.go` (импорт списком: `scheme://user:pass@host:port`, `host:port`, `host:port:user:pass`; схемы `http`/`https`/`socks4`/`socks4a`/`socks5`/`socks5h`; дедупликация, учёт пропущенных), `store.go` (потокобезопасный `Store`: `Load`/`List`/`Add`/`Remove`/`SetEnabled`/`Clear`, атомарная запись `proxies.json` с правами `0600`), `pool.go` (`Pool`: выбор по стратегиям round-robin/random/least-used/lowest-latency/sticky, `PickFastest` (по измеренной скорости), **`PickBest`** (по здоровью/задержке/скорости — для определения качества), `HasAvailable`, доступность, `failCount`+cooldown/backoff, телеметрия `ReportSpeed` (`SpeedBps` в `View` — EMA байт/с), `View` для UI), `health.go` (`Check` — SOCKS5-рукопожатие/TCP до цели, `CheckAll` — параллельная проверка) + `*_test.go`.
- `internal/history/` — история завершённых загрузок: `history.go` (`Entry` с `Title`/`Duration`/`Quality`/`Details`/`FilePath`/`LogPath`, потокобезопасный `Store`: `Load`/`Add`/`List`/`Get`/`Remove`/`Clear`, атомарная запись `history.json`, лимит записей) + `history_test.go`.
- `internal/browsers/` — автоопределение браузеров и профилей с cookies (`detect.go`: `Detect` → `[]Browser{Name, Label, Profiles}`; пути/семантика как в `yt_dlp/cookies.py` — Firefox (XDG/legacy/flatpak/snap), Chromium-семейство, Safari на macOS) + `detect_test.go`.
- `internal/pagetitle/` — запасное определение заголовка страницы (`Fetch` через `http.Client` + чистый `Title`: `og:title`/`twitter:title` → `<h1>` → `<title>`; декодирование сущностей, ограничение размера) + `pagetitle_test.go`.
- `internal/events/events.go` — имена и payload'ы событий `dep:*`, `task:progress`/`task:state` (с `status`)/`task:log` (журнал) и `proxy:health`.
- `internal/ytdlp/` — работа с yt-dlp:
  - `args.go` — сборка аргументов (чистые `BuildArgs` и `BuildInfoArgs` для `-J`; `--newline`/`--progress-template`, `--socket-timeout` (защита от зависания на «мёртвом» прокси), `--print after_move` (итоговый путь файла), `-P`/`-o` (имя включает `%(resolution)s` — чтобы разные качества одного видео не конфликтовали и не пропускались), `-f` (выбор формата), `--proxy` (и для загрузки, и для `-J`), `--js-runtimes`/`--ffmpeg-location`) + `args_test.go`;
  - `classify.go` — `ClassifyError`: класс ошибки по тексту (`ErrorNetwork`/`ErrorLogical`) — для решения о смене прокси + `classify_test.go`;
  - `process.go` — запуск: `Run` (обработчики `Handlers{Progress, File, Log}`) и `RunInfo`/`Output` (stdout+stderr для `-J`); привязка к `context.Context`, graceful-стоп → kill; при таймауте/отмене — понятное сообщение (`interruptError` + хвост stderr);
  - `progress.go` — `Progress` и чистый `parseProgressLine` (снимает префикс `loadika-progress:`, разбирает JSON `%(progress)j`) + `progress_test.go`;
  - `info.go` — `Info`/`Format` и чистый `ParseInfo` (разбор `-J`): вид потока (`muxed`/`video`/`audio`), метка качества (`1080p60`, `128 kbps`), **расшифровка кодеков** (`avc1…`→`H.264`, `vp09…`→`VP9`, `mp4a…`→`AAC`), длительность, размеры/fps/битрейты — чтобы UI помог выбрать лучший формат + `info_test.go`.
- `internal/download/` — домен задач загрузки: `manager.go` (создание/запуск/отмена/**пауза/докачка**, `List`/`Get`/`Shutdown`, **очередь**: состояния `queued`/`running`/`paused` + `IsFinalState`/`DefaultConcurrency`, планировщик `schedule`/`nextQueuedLocked` — не более `concurrency` задач одновременно (по умолчанию 1, строго последовательно), `Pause`/`Resume`/`Cancel`, `Inspect` через `-J` (выбор прокси — **`PickBest`** по здоровью/задержке/скорости) → `Inspect{Title, Duration, Formats, Qualities, OK, Error, Command, Output}` (включая **сырой ответ yt-dlp** — команда и stdout/stderr для диагностики); выбор формата — `Request.Format`; `Task.FilePath` — итоговый путь файла; `nextID` уникален между запусками (`task-<ms>-<seq>`); `Task.LogPath` — лог команды (`SetLogDir`); `Task.Status` — информативное состояние («Попытка 2/3 (через …)», «Сбой: … — пробуем другой прокси»); при таймауте `-J` (`context.DeadlineExceeded`) в `Inspect.Output` добавляется пояснение, что «Interrupted by user» — это наш graceful-стоп; каждая попытка `-J` ограничена собственным временем (`attemptTimeout`), ретраи со сменой прокси при сетевых сбоях (`SetProxyPool`, `attempt`, флаг `Request.UseProxy`); **адаптивное переключение по скорости**: сторож (`speed.go`) при низкой скорости (< 100 КиБ/с дольше 20 с) меняет прокси на более быстрый/непробованный, только если есть альтернатива (иначе не дёргаем), недокачанный файл продолжается (resume); колбэки прогресса/состояния/журнала (`LogFunc`); **журнал событий задачи** (`LogLine`, `addLog`/`Journal`, лимит 500 строк: попытки, выбор/смена прокси, счётчик использованных прокси, ошибки, итог); запуск процесса инъектируется — тесты офлайн) + `quality.go` (`QualityOptions`: **варианты качества, выведенные из фактических разрешений файла** — «Лучшее»/по разрешению/аудио, с расшифровкой кодеков в подсказке и **оценкой размера** — из `filesize` или по битрейту×длительности) + `*_test.go`.

**Frontend (Vue 3 + TS, FSD) — `frontend/src/`:**
- `app/` — `App.vue` (shell), `navigation.ts` (реестр экранов), `providers/pinia.ts`, `styles/` (токены/база).
- `pages/` — `downloads` (главная: секция **«Новая загрузка»** + виджет **«Задачи»** — шапка со сводкой состояний («1 загружается · 2 в очереди · 1 на паузе») и переходом на «Видео»), `proxies` (переключатель «Использовать прокси» (`useProxy`), импорт списком, статусы (+ измеренная скорость), стратегия, «Проверить»), `history` (вкладка **«Видео»** — библиотека: заголовок, качество, контейнер, длительность, размер, кодеки, дата; действия «Открыть» (плеером по умолчанию), «В папке», «Перекачать» (выбор качества прямо в карточке), «Удалить запись», «Удалить с файлом» (с подтверждением)), `settings` (табы «Основные»/«Зависимости»), `help` (**Справка** — пользовательское руководство: возможности, зависимости, загрузка, выбор качества, очередь/пауза, журнал, библиотека «Видео», прокси, авторизация YouTube, настройки/пути, FAQ с оглавлением; контент статический, из бэкенда — только текущая папка загрузки).
- `features/` — `theme-switcher`, `select-download-dir`, `add-download` (панель: URL + галочка «Использовать прокси» для конкретной загрузки + «Определить форматы» → кнопки качества с оценкой размера + свёрнутый список форматов + раскрывающийся блок **«Ответ yt-dlp»** (команда и сырой вывод; **очищается при новом определении и при старте загрузки**); состояние — в сторе `model/store.ts`, поэтому не теряется при переключении вкладок). Две настройки прокси: глобальная `useProxy` (переключатель на вкладке «Прокси») задаёт значение по умолчанию для галочки панели.
- `entities/` — `dependency` (стор + селекторы + `useDependencyEvents`), `download-task` (стор: `load`/`create`/`cancel`/**`pause`/`resume`** + `applyProgress`/`applyState`/`applyLog`/`journalFor`/`loadJournal`; счётчики `runningCount`/`queuedCount`/`pausedCount`; `useDownloadTaskEvents`), `proxy` (стор: `load`/`importList`/`remove`/`setEnabled`/`clear`/`setStrategy`/`setUseProxy`/`check` + `applyHealth`; `useProxyEvents`; селекторы статуса).
- `widgets/` — `deps-status` (inline-чипсы в футере главной), `deps-detail` (детальный блок: источник, действия, прогресс), `download-list` (шапка «Задачи» со сводкой («1 загружается · 2 в очереди · 1 на паузе») и переходом «Видео»; **незавершённые задачи отдельно, завершённые — сворачиваемый блок**; строка задачи вынесена в `TaskItem.vue`: заголовок видео + мелкая ссылка, прогресс/статус, **кнопки по состоянию (Пауза/Докачать/Отменить) с блокировкой на время запроса**, журнал загрузки с копированием и подгрузкой с бэкенда).
`shared/` — `api/` (обёртки биндингов + `onWailsEvent`; загрузки: `inspect`/`createDownload`/`cancelDownload`/`pauseDownload`/`resumeDownload`/`getDownloadTasks`/`getTaskJournal`; история: `getHistory`/`clearHistory`; система: `openPath`/`revealPath`/`copyText`), `config/events.ts` (имена `dep:*`/`task:*`/`task:log`),
  `config/settings.ts` (каналы `yt-dlp`), `config/navigation.ts` (`PageId`/`DEFAULT_PAGE_ID`), `stores/navigation.ts` (стор навигации — программное переключение вкладок), `lib/theme.ts`, `lib/format.ts` (`formatBytes`, `formatDateTime`, `formatTimestamp`, `formatClock`, `formatDuration`), `lib/error.ts` (`toMessage`), `ui/` (`AppButton`, `AppProgress`, `IconCheck`, иконки).
> Прокси в `shared/api`: `getProxies`/`getProxyStrategies`/`importProxies`/`removeProxy`/`setProxyEnabled`/`clearProxies`/`setProxyStrategy`/`checkProxies`.
- `frontend/bindings/` — **генерируется**, не редактировать вручную.
- `frontend/public/logo.svg` — логотип: монохромный силуэт-портрет в акцентном
  цвете, прозрачный фон; используется как CSS-маска (сайдбар) и favicon.

**Иконки / инструменты:**
- `frontend/public/logo.svg` — исходник знака: один цвет (`#9b5aa8`, совпадает с `brandColor`
  генератора) и прозрачный фон; знак — составной контур (несколько `<path>`, включая детали
  лица). Генератор перекрашивает его в белый и кладёт на акцентный фон.
- `build/appicon.png` — иконка приложения (собрана из `frontend/public/logo.svg` через
  `tools/appicon`: акцентный скруглённый фон + белый знак); из неё `build/Taskfile.yml` →
  `generate:icons` (`wails3 generate icons`) делает `build/windows/icon.ico` и `build/darwin/icons.icns`.
  Linux/AppImage/deb/rpm и `install:user:desktop` используют `appicon.png` напрямую.
- `build/android/.../mipmap-*/` — статичные копии `appicon.png` (обычная и круглая);
  iOS-иконки (`build/ios/Assets.xcassets`) генерируются из `appicon.png` при сборке (`wails3 ios xcode:gen`).
- Иконка **встроена в бинарник** (`//go:embed build/appicon.png` в `main.go`) и передаётся в
  `app.Run` → `application.Options.Icon` (Windows/macOS; GTK3-сборка Linux).
- На Linux по умолчанию GTK4: иконка окна берётся не из приложения, а из `.desktop` по app-id
  `ru.mysterentc.loadika-desktop` (`LinuxOptions.ApplicationID`); имя `.desktop` и иконки в теме
  должно совпадать с этим id (см. `docs/BUILD-AND-RELEASE.md` → «Иконки, метаданные и AppImage»).
- `tools/appicon/` — отдельный Go-модуль-генератор `appicon.png` из логотипа (не тянет зависимости
  в основной модуль; `-scale` по умолчанию 0.9 — под квадратный viewBox нового знака).
  Запуск: `cd tools/appicon && go run . -in ../../frontend/public/logo.svg -out ../../build/appicon.png`.
  Рядом с `appicon.png` кладётся уменьшенная копия `appicon-256.png` — её требует linuxdeploy
  при сборке AppImage (принимает только иконки из фиксированного набора размеров).
- `build/appicon.icon/` — исходник иконки macOS 26 (Icon Composer): `Assets/loadika_logo.svg`
  (копия логотипа) + `icon.json` (фон — брендовый `#9b5aa8`, знак — белый). `build/darwin/Assets.car`
  собирается из него **только на macOS** (`wails3 generate icons -iconcomposerinput`); на Linux
  инструмент его пропускает, поэтому `.car` в репозитории не хранится — macOS до пересборки
  использует `icons.icns` (`CFBundleIconFile`).
- `build/darwin/dmg-file-icon.{png,icns}` — иконка файла в DMG (пересобрана из `appicon.png`).

**Релиз / CI:**
- `.github/workflows/release.yml` — сборка и публикация релиза по тегу `v*` (Linux/Windows/macOS); см. `docs/BUILD-AND-RELEASE.md`.
  AppImage собирается через `linuxdeploy` **без бандлинга** библиотек (системные GTK4/WebKitGTK 6.0) — там же.
- `LICENSE` — MIT; та же лицензия в `build/linux/nfpm/nfpm.yaml`.
- Версия (0.1.4) живёт в `internal/version/version.go`, `build/config.yml`,
  `build/windows/info.json`, `build/windows/nsis/project.nsi`, `build/windows/msix/*.xml`,
  `build/darwin/Info*.plist`, `build/linux/nfpm/nfpm.yaml`.

## Решение по детальному блоку (3.1g)

Принято: **(b)** — развёрнутый блок внутри «Настроек», но с **табами**. Экран «Настройки»
разделён на вкладки: «Основные» (папка загрузки, тема — открывается по умолчанию) и
«Зависимости» (канал обновлений yt-dlp + виджет `deps-detail`).

## Следующие шаги (в порядке выполнения)

1. **3.1e** ✅ — `entities/dependency`: Pinia-стор (список, `refresh`, `checkUpdates`, `install`,
   `remove`, `removeManaged`, `hasUpdates`), поверх `shared/api`.
2. **3.1f** ✅ — виджет `deps-status` в футере главной («Загрузки»): inline-чипсы с галочкой
   у установленных, версией и ненавязчивым «Проверить».
3. **3.1g** ✅ — виджет `deps-detail` на вкладке «Зависимости» в «Настройках» (назначение, версия,
   источник, путь, когда обновлялось, свежая версия, «Установить/Обновить», «Переустановить», «Удалить»).
4. **3.2** ✅ — `deno`: `spec`-реестр зависимостей + распаковка zip, обнаружение системного
   рантайма → скачивание. Передача `--js-runtimes deno:<путь>` в `yt-dlp` — при реализации загрузок.
5. **3.3** ✅ — `ffmpeg`/`ffprobe`: модель «компонентов» (несколько файлов, gzip/zip, сумма
   через `digest`); источники: `ffmpeg-static` (Windows ARM64 — `FFmpeg-Builds`).
6. **3.4** ✅ — события `dep:*` (прогресс установки, троттлинг) и политики источника
   `auto`/`system`/`managed` (селектор в детальном блоке).
7. **Этап 4** ✅ — одиночная загрузка:
   - ✅ сборка аргументов `ytdlp` (`internal/ytdlp/args.go`: `BuildArgs`, `BuildInfoArgs`);
   - ✅ запуск процесса (`internal/ytdlp/process.go`: `Run`/`Output`, `context.Context`, graceful-стоп → kill)
     и разбор прогресса (`internal/ytdlp/progress.go`) + метаданных `-J` (`internal/ytdlp/info.go`:
     качество видео и расшифровка аудио/видео-кодеков — для выбора лучшего формата);
   - ✅ домен `internal/download` (задачи: создание, отмена, `List`/`Get`/`Formats`, `Shutdown`; выбор формата `Request.Format`; офлайн-тесты);
   - ✅ `DownloadService` (резолв путей через `deps`, эмиссия `task:*` с троттлингом, `ListFormats`/`Create`/`GetTasks`/`Cancel`), регистрация в `app.go` (+ `OnShutdown`), биндинги, обёртки `shared/api`;
   - ✅ `download.QualityOptions` — варианты качества, **выведенные из фактических разрешений файла** (`ListFormats` → `Formats{qualities}`), с **оценкой размера**; кнопки рендерятся по этому списку и показывают «≈ размер»;
   - ✅ фронтенд: `entities/download-task` (стор + `useDownloadTaskEvents`), `features/add-download` (панель: URL + кнопки качества + таблица форматов), `widgets/download-list` (прогресс/статус/отмена) на главной;
   - ✅ открытие результата: `yt-dlp --print after_move` → `Task.FilePath`, `internal/shell` + `SystemService` (`OpenPath`/`RevealPath`), кнопки «Открыть»/«В папке» в списке задач и истории;
   - ✅ логи команды: `--print`/`--progress` сохраняют полный вывод yt-dlp в `<config>/logs/<taskId>.log` (`Handlers.Log`, `Task.LogPath`/`Entry.LogPath`); в UI — «Открыть»/«В папке» (файл/папку результата).
8. **Этап 5** ✅ — очередь и история:
   - ✅ история: `internal/history` (потокобезопасный `Store`, `history.json`, лимит записей; тесты), запись по завершении в `DownloadService`, `HistoryService` + биндинги + `shared/api`, экран «История»;
   - ✅ персистентность панели добавления: состояние формы (`features/add-download/model/store.ts`) вынесено в Pinia — переключение вкладок больше не сбрасывает URL, найденные форматы и галочку прокси;
   - ✅ **авторизация YouTube**: cookies (`--cookies-from-browser` из браузера или `--cookies <файл>`) — обход проверки «Sign in to confirm you're not a bot»; настройки `cookiesBrowser`/`cookiesFile`, применены и к `-J`, и к загрузке; секция «Авторизация YouTube» в «Настройках» (нативный выбор файла). **Автоопределение браузеров**: `internal/browsers` находит установленные браузеры и их профили (пути/семантика синхронизированы с `yt_dlp/cookies.py`, `--cookies-from-browser браузер:профиль`), `SettingsService.GetCookieBrowsers` + выбор браузера и профиля в UI;
   - ✅ `--socket-timeout 30` в аргументах yt-dlp + **таймаут на каждую попытку** `-J` (`inspectAttemptTimeout` = 15 с, поле `Manager.attemptTimeout`): «зависший» прокси не съедает общий бюджет, и остальным прокси достаётся честная попытка (быстрее переключение); на само скачивание этот предел **не** действует; пометки в выводе различают таймаут попытки и исчерпание общего лимита;
   - ✅ **предварительная проверка прокси** перед `-J` и загрузкой: `DownloadService.preflightProxies` параллельно проверяет включённые прокси (цель — хост ссылки:443, `preflightAttemptTimeout`=3 с), нерабочие сразу уводит в cooldown (`Pool.Reject`) — не тратим время на мёртвые узлы; целевые прокси больше проверяются рукопожатием SOCKS5, поддерживаются схемы `socks5h`/`socks4a`;
   - ✅ **адаптивное переключение прокси по скорости**: телеметрия (пул копит измеренную скорость `ReportSpeed`/`SpeedBps`), сторож в `download` при низкой скорости (< 100 КиБ/с дольше 20 с) меняет прокси на более быстрый/непробованный и продолжает недокачанный файл (resume, `--continue` по умолчанию); смена только при наличии альтернативы (без «дёргания»); в UI видна измеренная скорость прокси;
   - ⏳ (идея дальше) **параллельный замер скорости кандидатов**: во время загрузки на медленном прокси запускать короткие ограниченные по времени/размеру загрузки через другие прокси (по прямым URL формата) и переключаться на самый быстрый. Ограничения: yt-dlp не умеет передавать живой поток между прокси — только resume-перезапуск; параллельные замеры конкурируют за ту же полосу и могут попасть под троттлинг, поэтому делать их осторожно/опционально;
   - ✅ **журнал загрузки в клиенте**: события `task:log` (уровни info/warn/error) — попытка/соединение, выбранный и счётчик использованных прокси, переключения (медленный/сбойный прокси), ошибки, итог; раскрывающийся журнал в строке задачи с кнопкой «Копировать» (буфер обмена через `SystemService.CopyText`) и подгрузкой с бэкенда (`GetTaskJournal`); журнал свой у каждой задачи (не смешивается между загрузками);
      - ✅ **вкладка «Видео»** (бывшая «История») как библиотека загрузок: в записи сохраняются заголовок, качество, длительность, реальный размер файла, расшифровка кодеков; действия — «Открыть» (приложением/плеером по умолчанию), «В папке», **«Перекачать»** (выбор качества прямо в карточке — форматы определяются по ссылке, без перехода на другую вкладку), «Удалить запись», «Удалить с файлом» (с подтверждением); метаданные передаются в `Create` из результата `-J`;
         - ✅ **заголовок в списке загрузок** (`Task.Title` из `meta`) + **запасной заголовок** (`internal/pagetitle`): если yt-dlp не дал названия, достаём `og:title`/`twitter:title`/`<h1>`/`<title>` со страницы (через SOCKS5-прокси пула, best-effort);
   - ✅ **очередь и лимит параллелизма**: `download.Manager` получил состояния `queued`/`running`/`paused` (+`IsFinalState`, `DefaultConcurrency = 1`), счётчики `running`/`concurrency` и планировщик `schedule`/`nextQueuedLocked` — запускается не более `concurrency` задач, по умолчанию **строго одна** (последовательно). `Start` теперь ставит задачу в очередь (снимок в состоянии `queued`); по завершении/паузе слот освобождается (`runEntry`: `running--` + `schedule`) — следующая задача из очереди стартует сама;
   - ✅ **пауза/докачка**: `Manager.Pause` (running → graceful-стоп, `pauseReq` отличает паузу от отмены → состояние `paused`; queued → сразу `paused`) и `Manager.Resume` (paused → `queued`, yt-dlp докачивает через `--continue`); `Manager.Cancel` для running/queued/paused; `Shutdown` запрещает запуск новых. Лог задачи открывается в режиме дозаписи (`O_APPEND`), чтобы resume/повтор не затирал историю; в историю попадают только финальные состояния (`IsFinalState`) — paused/queued не пишутся. Сервис: `DownloadService.Pause`/`Resume` + биндинги + `shared/api`;
   - ✅ **UI очереди**: стор `entities/download-task` хранит `runningCount`/`queuedCount`/`pausedCount`, селекторы `isRunning`/`isFinished`/`isPending` (вместо `isActive`); виджет `download-list` делит задачи на незавершённые (в очереди/выполняются/на паузе) и сворачиваемый блок «Завершённые», показывает сводку («1 загружается · 2 в очереди · 1 на паузе»); `TaskItem` рендерит кнопки по состоянию (running/queued → «Пауза»+«Отменить»; paused → «Докачать»+«Отменить»; завершённые → логи/файл) и блокирует их на время запроса (`busy`);
   - далее: Этапы 7–9.
9. Далее — по `07-roadmap.md` (Этапы 6–9).

## Промпт для новой сессии

Скопируйте текст ниже в новую сессию:

```
Продолжаем разработку Loadika Desktop (папка проекта: /home/timofey/ws/lab/loadika-desktop).
Это Wails v3 (Go) + Vue 3 + TypeScript (strict) по FSD.

Сначала прочитай:
- docs/README.md и docs/08-progress.md (текущий статус и промпт),
- docs/02-backend.md (архитектура бэкенда и поток загрузки) и docs/04-dependencies.md,
- docs/06-conventions.md и docs/03-frontend-fsd.md (соглашения: комментарии на русском,
  строгие типы, FSD-границы, «без утечек»),
- скилы .agents/skills/loadika-conventions и .agents/skills/loadika-dependencies.

Работаем МАЛЕНЬКИМИ аккуратными шагами (по одному), с русскими комментариями и проверкой после
каждого шага. Не запускай `wails3 dev` (нет webkit2gtk-4.1); валидация — `gofmt -l internal`,
`go build . ./internal/...`, `go vet ./internal/...`, `go test ./internal/...`,
`cd frontend && npm run typecheck && npm run build`; при изменениях сервисов/DTO —
`wails3 generate bindings -clean=true -ts -i`.

Текущий статус: **Этапы 0–6 завершены**. Готовы: зависимости (`internal/deps`), одиночная
загрузка + очередь + пауза/докачка (`internal/download`, `DownloadService`, события `task:*`),
история («Видео»), прокси (пул/стратегии/health-check/автопереключение).

Следующая задача — **Этап 7: пресеты загрузки** (`docs/07-roadmap.md`):
1. `internal/preset` (или `download-preset`): именованные наборы настроек (формат/качество, папка,
   субтитры, метаданные, использование прокси…) + сохранение в JSON и офлайн-тесты.
2. `PresetService` + регистрация в `app.go` + биндинги + обёртки `shared/api`.
3. Фронтенд: `entities/preset` (стор + события, если нужны), применение пресета в панели «Новая
   загрузка» (`features/add-download`).

Соглашения: комментарии — на русском, идентификаторы — английские; TS strict (без `any`); FSD —
импорт строго вниз, публичный API через `index.ts`, всё к бэкенду — только через `shared/api`.
Валидация после каждого шага: `gofmt -l internal`; `go build . ./internal/...`;
`go vet ./internal/...`; `go test ./internal/...`; `cd frontend && npm run typecheck && npm run build`;
при изменениях сервисов/DTO — `wails3 generate bindings -clean=true -ts -i`. **Не запускать `wails3 dev`**
(нет webkit2gtk-4.1).
```
