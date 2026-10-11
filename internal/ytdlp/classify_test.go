package ytdlp

import "testing"

func TestClassifyError(t *testing.T) {
	network := []string{
		"ERROR: unable to download video data: HTTP Error 503: Service Unavailable",
		"[download] Got error: <urlopen error [Errno 111] Connection refused>",
		"ERROR: Unable to connect to proxy: timed out",
		"curl: (28) Connection timed out after 10000 milliseconds",
		"[Errno -3] Temporary failure in name resolution",
		"SSL: CERTIFICATE_VERIFY_FAILED",
		"ERROR: giving up after 10 fragment retries",
		"yt-dlp: превышено время ожидания: context deadline exceeded",
	}
	for _, text := range network {
		if got := ClassifyError(text); got != ErrorNetwork {
			t.Errorf("ClassifyError(%q) = %v, want ErrorNetwork", text, got)
		}
	}

	logical := []string{
		"ERROR: [youtube] abc: Video unavailable",
		"ERROR: This video is private",
		"ERROR: requested format not available",
		"ERROR: Unsupported URL: https://example.com/x",
		"",
	}
	for _, text := range logical {
		if got := ClassifyError(text); got != ErrorLogical {
			t.Errorf("ClassifyError(%q) = %v, want ErrorLogical", text, got)
		}
	}
}
