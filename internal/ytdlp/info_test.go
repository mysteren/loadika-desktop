package ytdlp

import "testing"

func TestParseInfo(t *testing.T) {
	data := []byte(`{
		"id": "abc",
		"title": "Пример видео",
		"formats": [
			{
				"format_id": "137",
				"ext": "mp4",
				"width": 1920,
				"height": 1080,
				"fps": 60,
				"vcodec": "avc1.640028",
				"acodec": "none",
				"filesize": 12345678,
				"vbr": 2500.5,
				"format_note": "1080p60"
			},
			{
				"format_id": "140",
				"ext": "m4a",
				"vcodec": "none",
				"acodec": "mp4a.40.2",
				"filesize": null,
				"filesize_approx": 1600000.0,
				"abr": 128.0,
				"asr": 44100,
				"audio_channels": 2
			},
			{
				"format_id": "18",
				"ext": "mp4",
				"height": 360,
				"fps": 30,
				"vcodec": "avc1.42001E",
				"acodec": "mp4a.40.2",
				"tbr": 500
			},
			{
				"format_id": "sb0",
				"ext": "mhtml",
				"vcodec": "none",
				"acodec": "none"
			}
		]
	}`)

	info, err := ParseInfo(data)
	if err != nil {
		t.Fatalf("ParseInfo: %v", err)
	}
	if info.Title != "Пример видео" {
		t.Errorf("Title = %q", info.Title)
	}
	if len(info.Formats) != 4 {
		t.Fatalf("форматов %d, want 4", len(info.Formats))
	}

	video := info.Formats[0]
	if video.Kind != KindVideo || video.Quality != "1080p60" {
		t.Errorf("видео: kind=%q quality=%q", video.Kind, video.Quality)
	}
	if video.Width != 1920 || video.Height != 1080 || video.FPS != 60 {
		t.Errorf("видео: размеры/fps = %+v", video)
	}
	if video.VideoCodec != "H.264" || video.AudioCodec != "" {
		t.Errorf("видео: кодеки = %q/%q", video.VideoCodec, video.AudioCodec)
	}
	if video.Filesize != 12345678 || video.VideoBitrate != 2500.5 {
		t.Errorf("видео: числа = %+v", video)
	}

	audio := info.Formats[1]
	if audio.Kind != KindAudio || audio.Quality != "128 kbps" {
		t.Errorf("аудио: kind=%q quality=%q", audio.Kind, audio.Quality)
	}
	if audio.AudioCodec != "AAC" || audio.VideoCodec != "" {
		t.Errorf("аудио: кодеки = %q/%q", audio.AudioCodec, audio.VideoCodec)
	}
	if audio.Filesize != 1600000 || audio.SampleRate != 44100 || audio.Channels != 2 {
		t.Errorf("аудио: числа = %+v", audio)
	}

	muxed := info.Formats[2]
	if muxed.Kind != KindMuxed || muxed.Quality != "360p" {
		t.Errorf("мультиплекс: kind=%q quality=%q", muxed.Kind, muxed.Quality)
	}

	storyboard := info.Formats[3]
	if storyboard.Kind != "" || storyboard.Quality != "" {
		t.Errorf("служебный формат без медиа: %+v", storyboard)
	}
}

func TestCodecLabels(t *testing.T) {
	videoCases := map[string]string{
		"avc1.640028": "H.264",
		"h264":        "H.264",
		"hev1.1.6":    "H.265",
		"vp09.00.10":  "VP9",
		"vp9":         "VP9",
		"av01.0.05M":  "AV1",
		"none":        "",
		"":            "",
		"weirdcodec":  "weirdcodec", // неизвестный — как есть
	}
	for raw, want := range videoCases {
		if got := videoCodecLabel(raw); got != want {
			t.Errorf("videoCodecLabel(%q) = %q, want %q", raw, got, want)
		}
	}

	audioCases := map[string]string{
		"mp4a.40.2": "AAC",
		"opus":      "Opus",
		"vorbis":    "Vorbis",
		"mp3":       "MP3",
		"ec-3":      "E-AC-3",
		"none":      "",
		"":          "",
	}
	for raw, want := range audioCases {
		if got := audioCodecLabel(raw); got != want {
			t.Errorf("audioCodecLabel(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestParseInfoErrors(t *testing.T) {
	if _, err := ParseInfo([]byte(`{не json}`)); err == nil {
		t.Error("ожидалась ошибка на битом JSON")
	}
}

func TestParseInfoNoFormats(t *testing.T) {
	info, err := ParseInfo([]byte(`{"title": "T"}`))
	if err != nil {
		t.Fatalf("ParseInfo: %v", err)
	}
	if info.Title != "T" || len(info.Formats) != 0 {
		t.Errorf("info = %+v", info)
	}
}
