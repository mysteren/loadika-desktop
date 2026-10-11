package deps

import (
	"fmt"
	"strings"
)

// ytdlpSpec — зависимость yt-dlp: standalone-бинарники из GitHub Releases.
var ytdlpSpec = spec{
	name:       NameYtDlp,
	repo:       func(_, _, channel string) (string, error) { return ytdlpRepo(channel) },
	components: ytdlpComponents,
	hash:       parseSums,
	version:    firstLine,
	updates:    true,
}

// ytdlpComponents описывает единственный файл yt-dlp под указанную платформу.
func ytdlpComponents(goos, goarch string) ([]component, error) {
	asset, err := ytdlpAsset(goos, goarch)
	if err != nil {
		return nil, err
	}
	return []component{{
		asset:     asset,
		checksums: checksumsAsset,
		archive:   archiveNone,
		outputs:   []output{{dest: ytdlpBinary(goos)}},
	}}, nil
}

// ytdlpRepo возвращает GitHub-репозиторий yt-dlp для канала обновления.
func ytdlpRepo(channel string) (string, error) {
	repo, ok := channelRepos[channel]
	if !ok {
		return "", fmt.Errorf("неизвестный канал обновления: %q", channel)
	}
	return repo, nil
}

// ytdlpBinary возвращает имя итогового исполняемого файла yt-dlp.
func ytdlpBinary(goos string) string {
	if goos == "windows" {
		return "yt-dlp.exe"
	}
	return "yt-dlp"
}

// ytdlpAsset возвращает имя релизного файла yt-dlp под указанную платформу.
func ytdlpAsset(goos, goarch string) (string, error) {
	switch {
	case goos == "windows" && goarch == "amd64":
		return "yt-dlp.exe", nil
	case goos == "windows" && goarch == "arm64":
		return "yt-dlp_arm64.exe", nil
	case goos == "linux" && goarch == "amd64":
		return "yt-dlp_linux", nil
	case goos == "linux" && goarch == "arm64":
		return "yt-dlp_linux_aarch64", nil
	case goos == "darwin":
		return "yt-dlp_macos", nil
	default:
		return "", fmt.Errorf("платформа %s/%s не поддерживается", goos, goarch)
	}
}

// firstLine возвращает первую строку вывода (для однострочной версии yt-dlp).
func firstLine(raw string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(raw), "\n")
	return line
}
