package ytdlp

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Виды формата (потока).
const (
	KindMuxed = "muxed" // видео + аудио в одном файле
	KindVideo = "video" // только видео
	KindAudio = "audio" // только аудио
)

// Info — метаданные видео, полученные командой `-J` (--dump-single-json).
type Info struct {
	Title    string   `json:"title"`
	Duration int64    `json:"duration"` // длительность в секундах, 0 — неизвестно
	Formats  []Format `json:"formats"`
}

// Format — один доступный формат загрузки, приведённый к виду, удобному для выбора.
// Кодеки расшифрованы в человекочитаемые имена, качество сведено в короткую метку.
type Format struct {
	ID           string  `json:"id"`
	Ext          string  `json:"ext"`
	Kind         string  `json:"kind"`         // muxed | video | audio
	Quality      string  `json:"quality"`      // "1080p60", "720p", "128 kbps"
	Width        int     `json:"width"`        // 0 — неизвестно
	Height       int     `json:"height"`       // 0 — неизвестно (для аудио всегда 0)
	FPS          int     `json:"fps"`          // 0 — неизвестно
	VideoCodec   string  `json:"videoCodec"`   // расшифровка ("" для аудио): "H.264", "VP9", "AV1"
	AudioCodec   string  `json:"audioCodec"`   // расшифровка ("" для видео): "AAC", "Opus", "MP3"
	Filesize     int64   `json:"filesize"`     // 0 — неизвестно (иначе — оценка)
	VideoBitrate float64 `json:"videoBitrate"` // vbr, кбит/с
	AudioBitrate float64 `json:"audioBitrate"` // abr, кбит/с
	TotalBitrate float64 `json:"totalBitrate"` // tbr, кбит/с
	SampleRate   int     `json:"sampleRate"`   // Гц, для аудио
	Channels     int     `json:"channels"`     // число каналов, для аудио
	Note         string  `json:"note"`         // метка yt-dlp (format_note)
}

// rawInfo — сырой объект `-J` (берём только нужные поля).
type rawInfo struct {
	Title    string      `json:"title"`
	Duration *float64    `json:"duration"`
	Formats  []rawFormat `json:"formats"`
}

// rawFormat — сырой формат. Числа читаем как *float64: yt-dlp может отдать целое, дробное
// значение либо null, когда величина неизвестна.
type rawFormat struct {
	FormatID       string   `json:"format_id"`
	Ext            string   `json:"ext"`
	Width          *float64 `json:"width"`
	Height         *float64 `json:"height"`
	FPS            *float64 `json:"fps"`
	VCodec         string   `json:"vcodec"`
	ACodec         string   `json:"acodec"`
	Filesize       *float64 `json:"filesize"`
	FilesizeApprox *float64 `json:"filesize_approx"`
	VBR            *float64 `json:"vbr"`
	ABR            *float64 `json:"abr"`
	TBR            *float64 `json:"tbr"`
	ASR            *float64 `json:"asr"`
	Channels       *float64 `json:"audio_channels"`
	FormatNote     string   `json:"format_note"`
}

// ParseInfo разбирает вывод `yt-dlp -J` в Info. Функция чистая.
func ParseInfo(data []byte) (Info, error) {
	var raw rawInfo
	if err := json.Unmarshal(data, &raw); err != nil {
		return Info{}, fmt.Errorf("разбор метаданных yt-dlp: %w", err)
	}

	formats := make([]Format, 0, len(raw.Formats))
	for _, f := range raw.Formats {
		kind := formatKind(f.VCodec, f.ACodec)
		height := int(deref(f.Height))
		fps := int(deref(f.FPS))
		abr := deref(f.ABR)
		formats = append(formats, Format{
			ID:           f.FormatID,
			Ext:          f.Ext,
			Kind:         kind,
			Quality:      qualityLabel(kind, height, fps, abr, deref(f.TBR)),
			Width:        int(deref(f.Width)),
			Height:       height,
			FPS:          fps,
			VideoCodec:   videoCodecLabel(f.VCodec),
			AudioCodec:   audioCodecLabel(f.ACodec),
			Filesize:     int64(filesize(f.Filesize, f.FilesizeApprox)),
			VideoBitrate: deref(f.VBR),
			AudioBitrate: abr,
			TotalBitrate: deref(f.TBR),
			SampleRate:   int(deref(f.ASR)),
			Channels:     int(deref(f.Channels)),
			Note:         f.FormatNote,
		})
	}

	return Info{Title: raw.Title, Duration: int64(deref(raw.Duration)), Formats: formats}, nil
}

// formatKind определяет вид потока по строкам кодеков. Пустой результат — служебный формат
// без медиаданных (например, раскадровка/storyboard), такой формат для выбора не нужен.
func formatKind(vcodec, acodec string) string {
	hasVideo := hasMedia(vcodec)
	hasAudio := hasMedia(acodec)
	switch {
	case hasVideo && hasAudio:
		return KindMuxed
	case hasVideo:
		return KindVideo
	case hasAudio:
		return KindAudio
	default:
		return ""
	}
}

// hasMedia сообщает, что кодек задан (не пустой и не "none").
func hasMedia(codec string) bool {
	return codec != "" && codec != "none"
}

// qualityLabel строит короткую метку качества: для видео — разрешение (и частоту кадров),
// для аудио — битрейт.
func qualityLabel(kind string, height, fps int, abr, tbr float64) string {
	if kind == KindAudio {
		if abr > 0 {
			return fmt.Sprintf("%d kbps", round(abr))
		}
		return ""
	}
	if height > 0 {
		if fps > 30 {
			return fmt.Sprintf("%dp%d", height, fps)
		}
		return fmt.Sprintf("%dp", height)
	}
	if tbr > 0 {
		return fmt.Sprintf("%d kbps", round(tbr))
	}
	return ""
}

// videoCodecLabel расшифровывает строку видеокодека yt-dlp в читаемое имя. Неизвестный кодек
// возвращаем как есть, чтобы не терять информацию.
func videoCodecLabel(raw string) string {
	switch {
	case !hasMedia(raw):
		return ""
	case hasPrefix(raw, "avc1", "avc3"), raw == "h264":
		return "H.264"
	case hasPrefix(raw, "hev1", "hvc1"), raw == "hevc":
		return "H.265"
	case hasPrefix(raw, "av01"), raw == "av1":
		return "AV1"
	case hasPrefix(raw, "vp09"), raw == "vp9":
		return "VP9"
	case hasPrefix(raw, "vp08"), raw == "vp8":
		return "VP8"
	case raw == "mp4v":
		return "MPEG-4"
	case raw == "theora":
		return "Theora"
	default:
		return raw
	}
}

// audioCodecLabel расшифровывает строку аудиокодека yt-dlp в читаемое имя.
func audioCodecLabel(raw string) string {
	switch {
	case !hasMedia(raw):
		return ""
	case hasPrefix(raw, "mp4a"), raw == "aac":
		return "AAC"
	case raw == "opus":
		return "Opus"
	case raw == "vorbis":
		return "Vorbis"
	case hasPrefix(raw, "mp3"):
		return "MP3"
	case hasPrefix(raw, "ec-3"):
		return "E-AC-3"
	case hasPrefix(raw, "ac-3"):
		return "AC-3"
	case hasPrefix(raw, "flac"):
		return "FLAC"
	case hasPrefix(raw, "pcm"):
		return "PCM"
	default:
		return raw
	}
}

// hasPrefix сообщает, начинается ли s с любого из префиксов.
func hasPrefix(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// round округляет до ближайшего целого (для отображения битрейта).
func round(v float64) int {
	return int(v + 0.5)
}

// filesize выбирает известный размер, предпочитая точный (filesize) оценочному.
func filesize(exact, approx *float64) float64 {
	if exact != nil {
		return *exact
	}
	return deref(approx)
}
