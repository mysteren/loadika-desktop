package deps

import "testing"

func TestYtDlpAsset(t *testing.T) {
	cases := []struct {
		goos    string
		goarch  string
		want    string
		wantErr bool
	}{
		{"windows", "amd64", "yt-dlp.exe", false},
		{"windows", "arm64", "yt-dlp_arm64.exe", false},
		{"linux", "amd64", "yt-dlp_linux", false},
		{"linux", "arm64", "yt-dlp_linux_aarch64", false},
		{"darwin", "amd64", "yt-dlp_macos", false},
		{"darwin", "arm64", "yt-dlp_macos", false},
		{"freebsd", "amd64", "", true},
		{"linux", "riscv64", "", true},
	}

	for _, c := range cases {
		got, err := ytdlpAsset(c.goos, c.goarch)
		if c.wantErr {
			if err == nil {
				t.Errorf("ytdlpAsset(%q, %q): ожидалась ошибка", c.goos, c.goarch)
			}
			continue
		}
		if err != nil {
			t.Errorf("ytdlpAsset(%q, %q): %v", c.goos, c.goarch, err)
			continue
		}
		if got != c.want {
			t.Errorf("ytdlpAsset(%q, %q) = %q, want %q", c.goos, c.goarch, got, c.want)
		}
	}
}

func TestYtDlpBinary(t *testing.T) {
	if got := ytdlpBinary("windows"); got != "yt-dlp.exe" {
		t.Errorf("ytdlpBinary(windows) = %q, want yt-dlp.exe", got)
	}
	if got := ytdlpBinary("linux"); got != "yt-dlp" {
		t.Errorf("ytdlpBinary(linux) = %q, want yt-dlp", got)
	}
}

func TestYtDlpRepo(t *testing.T) {
	cases := map[string]string{
		"stable":  "yt-dlp/yt-dlp",
		"nightly": "yt-dlp/yt-dlp-nightly-builds",
		"master":  "yt-dlp/yt-dlp-master-builds",
	}
	for channel, want := range cases {
		got, err := ytdlpRepo(channel)
		if err != nil {
			t.Errorf("ytdlpRepo(%q): %v", channel, err)
			continue
		}
		if got != want {
			t.Errorf("ytdlpRepo(%q) = %q, want %q", channel, got, want)
		}
	}
	if _, err := ytdlpRepo("bogus"); err == nil {
		t.Error("ytdlpRepo(bogus): ожидалась ошибка")
	}
}

func TestParseSums(t *testing.T) {
	text := "aaaa1111  yt-dlp.exe\nbbbb2222 *yt-dlp_linux\ncccc3333  yt-dlp_macos\n"

	got, err := parseSums(text, "yt-dlp_linux")
	if err != nil {
		t.Fatalf("parseSums: %v", err)
	}
	if got != "bbbb2222" {
		t.Fatalf("parseSums = %q, want bbbb2222", got)
	}

	got, err = parseSums(text, "yt-dlp_macos")
	if err != nil {
		t.Fatalf("parseSums: %v", err)
	}
	if got != "cccc3333" {
		t.Fatalf("parseSums = %q, want cccc3333", got)
	}

	if _, err := parseSums(text, "missing"); err == nil {
		t.Error("parseSums: ожидалась ошибка для отсутствующего файла")
	}
}
