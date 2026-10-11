package download

import (
	"fmt"
	"sort"
	"strings"

	"loadika-desktop/internal/ytdlp"
)

// QualityOption — вариант выбора качества, выведенный из форматов конкретного файла
// (для рендеринга кнопок в интерфейсе). Height/Size == 0 у «Лучшее» и «Только аудио»
// означают соответственно «без ограничения по высоте» и «размер неизвестен».
type QualityOption struct {
	ID       string `json:"id"`       // стабильный ключ: "best" | "1080p" | "audio"
	Label    string `json:"label"`    // "1080p", "Только аудио"
	Hint     string `json:"hint"`     // расшифровка (кодеки, fps, битрейт) — помогает выбрать
	Selector string `json:"selector"` // значение флага -f
	Height   int    `json:"height"`
	Size     int64  `json:"size"` // оценка размера в байтах (0 — неизвестно)
}

// Inspect — результат запроса метаданных ссылки: заголовок, форматы и варианты качества, а также
// СЫРОЙ ответ yt-dlp (команда и stdout/stderr) для диагностики. OK=false — запрос не удался
// (см. Error/Output).
type Inspect struct {
	Title     string          `json:"title"`
	Duration  int64           `json:"duration"` // длительность видео в секундах (0 — неизвестно)
	Formats   []ytdlp.Format  `json:"formats"`
	Qualities []QualityOption `json:"qualities"`
	OK        bool            `json:"ok"`
	Error     string          `json:"error"`
	Command   string          `json:"command"`
	Output    string          `json:"output"`
}

// QualityOptions выводит варианты качества из доступных форматов: «Лучшее», по одному пункту
// на каждое доступное разрешение (по убыванию) и «Только аудио». Функция чистая — результат
// зависит только от info.
func QualityOptions(info ytdlp.Info) []QualityOption {
	heights := videoHeights(info.Formats)
	hasAudio := hasAudioFormats(info.Formats)

	options := make([]QualityOption, 0, len(heights)+2)
	if len(heights) > 0 || hasAudio {
		best := QualityOption{
			ID:       "best",
			Label:    "Лучшее",
			Selector: "bv*+ba/b",
			Hint:     "Наилучшее доступное качество",
			Size:     qualitySize(info, 0),
		}
		if len(heights) > 0 {
			best.Hint = fmt.Sprintf("Наилучшее доступное качество, до %dp", heights[0])
		}
		options = append(options, best)
	}

	for _, h := range heights {
		options = append(options, QualityOption{
			ID:       fmt.Sprintf("%dp", h),
			Label:    fmt.Sprintf("%dp", h),
			Hint:     videoHint(info.Formats, h),
			Selector: fmt.Sprintf("bv*[height<=%d]+ba/b[height<=%d]", h, h),
			Height:   h,
			Size:     qualitySize(info, h),
		})
	}

	if hasAudio {
		options = append(options, QualityOption{
			ID:       "audio",
			Label:    "Только аудио",
			Hint:     audioHint(info.Formats),
			Selector: "ba/b",
			Size:     audioSize(info),
		})
	}
	return options
}

// videoHeights возвращает уникальные высоты видеопотоков по убыванию.
func videoHeights(formats []ytdlp.Format) []int {
	seen := make(map[int]bool)
	var heights []int
	for _, f := range formats {
		if f.Height <= 0 || (f.Kind != ytdlp.KindVideo && f.Kind != ytdlp.KindMuxed) {
			continue
		}
		if !seen[f.Height] {
			seen[f.Height] = true
			heights = append(heights, f.Height)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(heights)))
	return heights
}

// hasAudioFormats сообщает, есть ли у файла хотя бы один аудиопоток.
func hasAudioFormats(formats []ytdlp.Format) bool {
	for _, f := range formats {
		if f.AudioCodec != "" {
			return true
		}
	}
	return false
}

// qualitySize оценивает размер итогового файла для разрешения maxHeight (0 — без ограничения):
// лучший видеопоток плюс лучшая аудиодорожка (если видео без звука). 0 — размер неизвестен.
func qualitySize(info ytdlp.Info, maxHeight int) int64 {
	video, hasVideo := bestVideo(info.Formats, maxHeight)
	audio, hasAudio := bestAudio(info.Formats)

	var total int64
	if hasVideo {
		total += estimateSize(video, info.Duration)
	}
	// Отдельную аудиодорожку добавляем, только если её нет в выбранном видео.
	if hasAudio && (!hasVideo || video.AudioCodec == "") {
		total += estimateSize(audio, info.Duration)
	}
	return total
}

// audioSize оценивает размер лучшей аудиодорожки.
func audioSize(info ytdlp.Info) int64 {
	audio, ok := bestAudio(info.Formats)
	if !ok {
		return 0
	}
	return estimateSize(audio, info.Duration)
}

// estimateSize возвращает точный/оценочный размер формата, а при его отсутствии — оценку
// по битрейту и длительности. 0 — размер неизвестен.
func estimateSize(f ytdlp.Format, duration int64) int64 {
	if f.Filesize > 0 {
		return f.Filesize
	}
	if f.TotalBitrate > 0 && duration > 0 {
		return int64(f.TotalBitrate * 1000 / 8 * float64(duration))
	}
	return 0
}

// bestVideo выбирает видеопоток с наибольшим разрешением (не выше maxHeight), затем fps и
// битрейтом — приближённо повторяя выбор yt-dlp для `bv*[height<=maxHeight]`.
func bestVideo(formats []ytdlp.Format, maxHeight int) (ytdlp.Format, bool) {
	var best ytdlp.Format
	found := false
	for _, f := range formats {
		if f.Height <= 0 || (f.Kind != ytdlp.KindVideo && f.Kind != ytdlp.KindMuxed) {
			continue
		}
		if maxHeight > 0 && f.Height > maxHeight {
			continue
		}
		if !found || videoBetter(f, best) {
			best, found = f, true
		}
	}
	return best, found
}

// videoBetter сообщает, что поток a предпочтительнее b (разрешение, затем fps и битрейт).
func videoBetter(a, b ytdlp.Format) bool {
	if a.Height != b.Height {
		return a.Height > b.Height
	}
	if a.FPS != b.FPS {
		return a.FPS > b.FPS
	}
	if a.TotalBitrate != b.TotalBitrate {
		return a.TotalBitrate > b.TotalBitrate
	}
	return a.Filesize > b.Filesize
}

// bestAudio выбирает аудиопоток с наибольшим битрейтом.
func bestAudio(formats []ytdlp.Format) (ytdlp.Format, bool) {
	var best ytdlp.Format
	found := false
	for _, f := range formats {
		if f.AudioCodec == "" {
			continue
		}
		if !found || f.AudioBitrate > best.AudioBitrate ||
			(f.AudioBitrate == best.AudioBitrate && f.TotalBitrate > best.TotalBitrate) {
			best, found = f, true
		}
	}
	return best, found
}

// videoHint строит расшифровку для разрешения: доступные видеокодеки и максимальный fps.
func videoHint(formats []ytdlp.Format, height int) string {
	codecs := distinct(formats, height, func(f ytdlp.Format) string { return f.VideoCodec })

	maxFPS := 0
	for _, f := range formats {
		if f.Height == height && f.FPS > maxFPS {
			maxFPS = f.FPS
		}
	}

	switch {
	case len(codecs) > 0 && maxFPS > 30:
		return fmt.Sprintf("%s · до %d fps", strings.Join(codecs, ", "), maxFPS)
	case maxFPS > 30:
		return fmt.Sprintf("до %d fps", maxFPS)
	default:
		return strings.Join(codecs, ", ")
	}
}

// audioHint строит расшифровку для аудио: доступные аудиокодеки и максимальный битрейт.
func audioHint(formats []ytdlp.Format) string {
	seen := make(map[string]bool)
	var codecs []string
	maxBitrate := 0.0
	for _, f := range formats {
		if f.AudioCodec == "" {
			continue
		}
		if !seen[f.AudioCodec] {
			seen[f.AudioCodec] = true
			codecs = append(codecs, f.AudioCodec)
		}
		if f.AudioBitrate > maxBitrate {
			maxBitrate = f.AudioBitrate
		}
	}

	joined := strings.Join(codecs, ", ")
	switch {
	case joined != "" && maxBitrate > 0:
		return fmt.Sprintf("%s · до %d kbps", joined, roundBitrate(maxBitrate))
	case maxBitrate > 0:
		return fmt.Sprintf("до %d kbps", roundBitrate(maxBitrate))
	default:
		return joined
	}
}

// distinct собирает уникальные непустые значения video-кодеков форматов заданной высоты.
func distinct(formats []ytdlp.Format, height int, value func(ytdlp.Format) string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, f := range formats {
		if f.Height != height {
			continue
		}
		v := value(f)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// roundBitrate округляет битрейт до целого для отображения.
func roundBitrate(v float64) int {
	return int(v + 0.5)
}
