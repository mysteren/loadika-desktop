package deps

import (
	"fmt"
	"strings"
)

// denoSpec — зависимость deno: JS-рантайм для yt-dlp-ejs (zip-архивы из GitHub Releases).
var denoSpec = spec{
	name:       NameDeno,
	repo:       func(_, _, _ string) (string, error) { return "denoland/deno", nil },
	components: denoComponents,
	hash:       func(text, _ string) (string, error) { return parseSHA256(text) },
	version:    denoVersion,
	versionKey: denoVersionKey,
	updates:    true,
}

// denoComponents описывает архив deno под указанную платформу.
func denoComponents(goos, goarch string) ([]component, error) {
	asset, err := denoAsset(goos, goarch)
	if err != nil {
		return nil, err
	}
	bin := denoBinary(goos)
	return []component{{
		asset:     asset,
		checksums: asset + ".sha256sum",
		archive:   archiveZip,
		outputs:   []output{{member: bin, dest: bin}},
	}}, nil
}

// denoBinary возвращает имя исполняемого файла deno.
func denoBinary(goos string) string {
	if goos == "windows" {
		return "deno.exe"
	}
	return "deno"
}

// denoAsset возвращает имя релизного архива deno под указанную платформу.
func denoAsset(goos, goarch string) (string, error) {
	switch {
	case goos == "linux" && goarch == "amd64":
		return "deno-x86_64-unknown-linux-gnu.zip", nil
	case goos == "linux" && goarch == "arm64":
		return "deno-aarch64-unknown-linux-gnu.zip", nil
	case goos == "darwin" && goarch == "amd64":
		return "deno-x86_64-apple-darwin.zip", nil
	case goos == "darwin" && goarch == "arm64":
		return "deno-aarch64-apple-darwin.zip", nil
	case goos == "windows" && goarch == "amd64":
		return "deno-x86_64-pc-windows-msvc.zip", nil
	case goos == "windows" && goarch == "arm64":
		return "deno-aarch64-pc-windows-msvc.zip", nil
	default:
		return "", fmt.Errorf("платформа %s/%s не поддерживается", goos, goarch)
	}
}

// denoVersionKey приводит версию к сравнимому виду: тег релиза «v2.9.7» и вывод
// `deno --version` («2.9.7») должны считаться одинаковыми.
func denoVersionKey(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

// denoVersion извлекает номер версии из вывода `deno --version` («deno 2.9.7 …»).
func denoVersion(raw string) string {
	line := firstLine(raw)
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		return fields[1]
	}
	return line
}
