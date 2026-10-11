package download

import (
	"testing"

	"loadika-desktop/internal/ytdlp"
)

// sampleInfo — небольшой набор форматов для проверки вывода вариантов качества.
func sampleInfo() ytdlp.Info {
	return ytdlp.Info{
		Title:    "T",
		Duration: 0, // размеры заданы явно через filesize
		Formats: []ytdlp.Format{
			{ID: "137", Kind: ytdlp.KindVideo, Height: 1080, FPS: 60, VideoCodec: "H.264", Filesize: 20_000_000, TotalBitrate: 4500},
			{ID: "248", Kind: ytdlp.KindVideo, Height: 1080, FPS: 60, VideoCodec: "VP9", Filesize: 15_000_000, TotalBitrate: 3000},
			{ID: "136", Kind: ytdlp.KindVideo, Height: 720, FPS: 30, VideoCodec: "H.264", Filesize: 10_000_000, TotalBitrate: 1500},
			{ID: "140", Kind: ytdlp.KindAudio, AudioCodec: "AAC", AudioBitrate: 128, Filesize: 1_600_000},
			{ID: "251", Kind: ytdlp.KindAudio, AudioCodec: "Opus", AudioBitrate: 160, Filesize: 2_000_000},
			{ID: "18", Kind: ytdlp.KindMuxed, Height: 360, FPS: 30, VideoCodec: "H.264", AudioCodec: "AAC", Filesize: 5_000_000},
			{ID: "sb0", Kind: "", Ext: "mhtml"},
		},
	}
}

func TestQualityOptions(t *testing.T) {
	options := QualityOptions(sampleInfo())

	// Ожидаем: best, 1080p, 720p, 360p, audio.
	ids := make([]string, 0, len(options))
	for _, o := range options {
		ids = append(ids, o.ID)
	}
	want := []string{"best", "1080p", "720p", "360p", "audio"}
	if len(ids) != len(want) {
		t.Fatalf("варианты = %v, want %v", ids, want)
	}
	for i, id := range want {
		if ids[i] != id {
			t.Fatalf("варианты = %v, want %v", ids, want)
		}
	}

	if options[0].Selector != "bv*+ba/b" {
		t.Errorf("best selector = %q", options[0].Selector)
	}
	if options[1].Selector != "bv*[height<=1080]+ba/b[height<=1080]" || options[1].Height != 1080 {
		t.Errorf("1080p = %+v", options[1])
	}
	if options[1].Hint != "H.264, VP9 · до 60 fps" {
		t.Errorf("1080p hint = %q", options[1].Hint)
	}

	audio := options[len(options)-1]
	if audio.Selector != "ba/b" || audio.Height != 0 {
		t.Errorf("audio = %+v", audio)
	}
	if audio.Hint != "AAC, Opus · до 160 kbps" {
		t.Errorf("audio hint = %q", audio.Hint)
	}

	// Оценки размера: лучший видеопоток + лучшая аудиодорожка (если видео без звука).
	sizes := map[string]int64{}
	for _, o := range options {
		sizes[o.ID] = o.Size
	}
	if sizes["best"] != 22_000_000 {
		t.Errorf("best size = %d, want 22000000", sizes["best"])
	}
	if sizes["1080p"] != 22_000_000 {
		t.Errorf("1080p size = %d, want 22000000", sizes["1080p"])
	}
	if sizes["720p"] != 12_000_000 {
		t.Errorf("720p size = %d, want 12000000", sizes["720p"])
	}
	// 360p — единственный поток 18 имеет и видео, и звук → размер только видео.
	if sizes["360p"] != 5_000_000 {
		t.Errorf("360p size = %d, want 5000000", sizes["360p"])
	}
	if sizes["audio"] != 2_000_000 {
		t.Errorf("audio size = %d, want 2000000", sizes["audio"])
	}
}

func TestQualityOptionsEstimateByBitrate(t *testing.T) {
	// Форматы без filesize: размер считаем по битрейту и длительности.
	info := ytdlp.Info{
		Duration: 10,
		Formats: []ytdlp.Format{
			{ID: "v", Kind: ytdlp.KindVideo, Height: 720, VideoCodec: "H.264", TotalBitrate: 8000},    // 10 МБ
			{ID: "a", Kind: ytdlp.KindAudio, AudioCodec: "AAC", AudioBitrate: 128, TotalBitrate: 128}, // 160 КБ
		},
	}

	options := QualityOptions(info)
	if len(options) == 0 {
		t.Fatal("нет вариантов")
	}
	if got := options[0].Size; got != 10_160_000 {
		t.Errorf("best size = %d, want 10160000", got)
	}
}

func TestQualityOptionsEmpty(t *testing.T) {
	if options := QualityOptions(ytdlp.Info{}); len(options) != 0 {
		t.Errorf("для пустых метаданных вариантов %d, want 0: %+v", len(options), options)
	}
}

func TestQualityOptionsAudioOnly(t *testing.T) {
	info := ytdlp.Info{Formats: []ytdlp.Format{
		{ID: "140", Kind: ytdlp.KindAudio, AudioCodec: "AAC", AudioBitrate: 128, Filesize: 1_000_000},
	}}
	options := QualityOptions(info)
	if len(options) != 2 {
		t.Fatalf("вариантов %d, want 2 (best, audio): %+v", len(options), options)
	}
	if options[0].ID != "best" || options[1].ID != "audio" {
		t.Errorf("варианты = %+v", options)
	}
	if options[1].Size != 1_000_000 {
		t.Errorf("audio size = %d, want 1000000", options[1].Size)
	}
}
