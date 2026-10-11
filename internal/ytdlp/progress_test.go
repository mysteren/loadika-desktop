package ytdlp

import (
	"strings"
	"testing"
)

func TestParseProgressLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want Progress
		ok   bool
	}{
		{
			name: "полный объект",
			line: `loadika-progress:{"status": "downloading", "downloaded_bytes": 1024, ` +
				`"total_bytes": 4096, "speed": 512.5, "eta": 6, "filename": "video.mp4"}`,
			want: Progress{
				Status:     "downloading",
				Downloaded: 1024,
				Total:      4096,
				Speed:      512.5,
				ETA:        6,
				Filename:   "video.mp4",
			},
			ok: true,
		},
		{
			name: "итог из оценки, когда total_bytes нет",
			line: `loadika-progress:{"status": "downloading", "downloaded_bytes": 100, ` +
				`"total_bytes": null, "total_bytes_estimate": 200.0}`,
			want: Progress{Status: "downloading", Downloaded: 100, Total: 200},
			ok:   true,
		},
		{
			name: "итог из оценки при дробном значении",
			line: `loadika-progress:{"downloaded_bytes": 1, "total_bytes_estimate": 1234.9}`,
			want: Progress{Downloaded: 1, Total: 1234},
			ok:   true,
		},
		{
			name: "все числовые поля отсутствуют",
			line: `loadika-progress:{"status": "finished"}`,
			want: Progress{Status: "finished"},
			ok:   true,
		},
		{
			name: "строки без префикса игнорируются",
			line: `[download] Destination: video.mp4`,
			ok:   false,
		},
		{
			name: "битый JSON после префикса игнорируется",
			line: `loadika-progress:{not json}`,
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseProgressLine(tt.line)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if !tt.ok {
				return
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestScanProgress(t *testing.T) {
	// Смешиваем строки прогресса с посторонним выводом: посторонние должны быть пропущены,
	// иначе они «протекут» в UI.
	input := strings.Join([]string{
		"[youtube] Extracting URL: https://example.com/v",
		`loadika-progress:{"status": "downloading", "downloaded_bytes": 10, "total_bytes": 100}`,
		`loadika-progress:{"status": "downloading", "downloaded_bytes": 50, "total_bytes": 100}`,
		"[download] 50% of 100B",
		`loadika-progress:{"status": "finished", "downloaded_bytes": 100, "total_bytes": 100}`,
	}, "\n")

	var got []Progress
	h := Handlers{Progress: func(p Progress) { got = append(got, p) }}
	if err := scanOutput(strings.NewReader(input), h); err != nil {
		t.Fatalf("scanOutput: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("получено %d прогрессов, want 3: %+v", len(got), got)
	}
	if got[0].Downloaded != 10 || got[1].Downloaded != 50 || got[2].Status != "finished" {
		t.Errorf("неожиданная последовательность: %+v", got)
	}
}

func TestScanOutputFilePath(t *testing.T) {
	input := strings.Join([]string{
		`loadika-progress:{"status": "finished", "downloaded_bytes": 100, "total_bytes": 100}`,
		"loadika-file=/tmp/dl/video.mp4",
		"\u043a\u0430\u043a\u0430\u044f-\u0442\u043e \u0441\u0442\u0440\u043e\u043a\u0430",
	}, "\n")

	var files []string
	h := Handlers{File: func(path string) { files = append(files, path) }}
	if err := scanOutput(strings.NewReader(input), h); err != nil {
		t.Fatalf("scanOutput: %v", err)
	}

	if len(files) != 1 || files[0] != "/tmp/dl/video.mp4" {
		t.Fatalf("файлы = %+v, want [/tmp/dl/video.mp4]", files)
	}
}

func TestScanOutputNilHandlers(t *testing.T) {
	// Без обработчиков чтение всё равно должно корректно дойти до конца.
	input := `loadika-progress:{"status": "downloading", "downloaded_bytes": 1}`
	if err := scanOutput(strings.NewReader(input), Handlers{}); err != nil {
		t.Fatalf("scanOutput: %v", err)
	}
}
