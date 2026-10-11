package deps

import (
	"fmt"
	"strings"
)

// ffmpegSpec — зависимость ffmpeg (вместе с ffprobe). Нужна лениво — только для
// склейки/постобработки. По умолчанию берём лёгкие статические сборки ffmpeg-static,
// а для Windows ARM64 (где их нет) — сборку yt-dlp/FFmpeg-Builds.
var ffmpegSpec = spec{
	name:        NameFfmpeg,
	repo:        ffmpegRepo,
	components:  ffmpegComponents,
	hash:        parseSums,
	version:     ffmpegVersion,
	versionArgs: []string{"-version"}, // ffmpeg понимает одиночный дефис
	updates:     false,                // FFmpeg-Builds использует роллинг-тег `latest`
}

// ffmpegRepo выбирает репозиторий по платформе.
func ffmpegRepo(goos, goarch, _ string) (string, error) {
	if goos == "windows" && goarch == "arm64" {
		return "yt-dlp/FFmpeg-Builds", nil
	}
	return "eugeneware/ffmpeg-static", nil
}

// ffmpegComponents описывает файлы зависимости под указанную платформу.
func ffmpegComponents(goos, goarch string) ([]component, error) {
	// Windows ARM64: в ffmpeg-static сборок нет — берём единый zip из FFmpeg-Builds,
	// содержащий ffmpeg.exe и ffprobe.exe.
	if goos == "windows" && goarch == "arm64" {
		return []component{{
			asset:     "ffmpeg-master-latest-winarm64-gpl.zip",
			checksums: "checksums.sha256",
			archive:   archiveZip,
			outputs: []output{
				{member: "ffmpeg.exe", dest: "ffmpeg.exe"},
				{member: "ffprobe.exe", dest: "ffprobe.exe"},
			},
		}}, nil
	}

	platform, err := ffmpegStaticPlatform(goos, goarch)
	if err != nil {
		return nil, err
	}
	return []component{
		{
			asset:   "ffmpeg-" + platform + ".gz",
			archive: archiveGzip,
			outputs: []output{{dest: ffmpegBinary(goos)}},
		},
		{
			asset:   "ffprobe-" + platform + ".gz",
			archive: archiveGzip,
			outputs: []output{{dest: ffprobeBinary(goos)}},
		},
	}, nil
}

// ffmpegStaticPlatform возвращает суффикс ассетов ffmpeg-static по платформе.
func ffmpegStaticPlatform(goos, goarch string) (string, error) {
	switch {
	case goos == "darwin" && goarch == "amd64":
		return "darwin-x64", nil
	case goos == "darwin" && goarch == "arm64":
		return "darwin-arm64", nil
	case goos == "linux" && goarch == "amd64":
		return "linux-x64", nil
	case goos == "linux" && goarch == "arm64":
		return "linux-arm64", nil
	case goos == "linux" && goarch == "arm":
		return "linux-arm", nil
	case goos == "linux" && goarch == "386":
		return "linux-ia32", nil
	case goos == "windows" && goarch == "amd64":
		return "win32-x64", nil
	default:
		return "", fmt.Errorf("платформа %s/%s не поддерживается", goos, goarch)
	}
}

// ffmpegBinary возвращает имя исполняемого файла ffmpeg.
func ffmpegBinary(goos string) string {
	if goos == "windows" {
		return "ffmpeg.exe"
	}
	return "ffmpeg"
}

// ffprobeBinary возвращает имя исполняемого файла ffprobe.
func ffprobeBinary(goos string) string {
	if goos == "windows" {
		return "ffprobe.exe"
	}
	return "ffprobe"
}

// ffmpegVersion извлекает версию из вывода `ffmpeg -version` («ffmpeg version 6.1.1 …»).
func ffmpegVersion(raw string) string {
	fields := strings.Fields(firstLine(raw))
	if len(fields) >= 3 && fields[1] == "version" {
		return fields[2]
	}
	return firstLine(raw)
}
