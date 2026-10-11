package deps

import (
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// installSpec скачивает, проверяет и атомарно устанавливает зависимость по её описанию.
// channel влияет только на выбор репозитория (для deno/ffmpeg игнорируется).
func (m *Manager) installSpec(ctx context.Context, s spec, channel string, onProgress ProgressFunc) error {
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return fmt.Errorf("создание каталога зависимостей: %w", err)
	}

	repo, err := s.repo(runtime.GOOS, runtime.GOARCH, channel)
	if err != nil {
		return err
	}
	rel, err := fetchLatestRelease(ctx, repo)
	if err != nil {
		return err
	}
	comps, err := s.components(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	report(onProgress, Progress{Phase: "download"})
	var firstSHA string
	for i, comp := range comps {
		sha, err := m.installComponent(ctx, s, rel, comp, onProgress)
		if err != nil {
			return err
		}
		if i == 0 {
			firstSHA = sha
		}
	}

	version := rel.TagName
	if !s.updates {
		// У роллинг-релизов (ffmpeg) тег не является версией — берём её у самого бинарника.
		if primary := s.primaryBinaryName(); primary != "" {
			if v := binaryVersion(s, filepath.Join(m.dir, primary)); v != "" {
				version = v
			}
		}
	}

	mf := loadManifest(m.manifestPath())
	mf[s.name] = manifestEntry{
		Version:   version,
		Channel:   channel,
		Asset:     comps[0].asset,
		SHA256:    firstSHA,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	return saveManifest(m.manifestPath(), mf)
}

// installComponent скачивает и активирует один файл зависимости, возвращая его SHA-256.
func (m *Manager) installComponent(ctx context.Context, s spec, rel release, comp component, onProgress ProgressFunc) (string, error) {
	fileURL, ok := rel.assetURL(comp.asset)
	if !ok {
		return "", fmt.Errorf("в релизе %s нет файла %q", rel.TagName, comp.asset)
	}

	wantSHA, err := componentChecksum(ctx, s, rel, comp)
	if err != nil {
		return "", err
	}

	tmp := filepath.Join(m.dir, s.name+"-"+comp.asset+".tmp")
	defer func() { _ = os.Remove(tmp) }()
	if err := downloadFile(ctx, fileURL, tmp, wantSHA, func(received, total int64) {
		report(onProgress, Progress{Phase: "download", Received: received, Total: total})
	}); err != nil {
		return "", err
	}

	report(onProgress, Progress{Phase: "install"})
	if err := m.activateComponent(comp, tmp); err != nil {
		return "", err
	}
	return wantSHA, nil
}

// componentChecksum определяет ожидаемый SHA-256: из файла контрольных сумм либо
// из метаданных релиза (digest), если отдельного файла у источника нет.
func componentChecksum(ctx context.Context, s spec, rel release, comp component) (string, error) {
	if comp.checksums == "" {
		return rel.assetDigest(comp.asset)
	}

	sumsURL, ok := rel.assetURL(comp.checksums)
	if !ok {
		return "", fmt.Errorf("в релизе %s нет файла контрольных сумм %q", rel.TagName, comp.checksums)
	}
	sumsText, err := getBytes(ctx, sumsURL)
	if err != nil {
		return "", err
	}
	return s.hash(string(sumsText), comp.asset)
}

// activateComponent распаковывает файл (при необходимости) и атомарно кладёт бинарники
// на их итоговые места, выставляя бит исполнения.
func (m *Manager) activateComponent(comp component, src string) error {
	switch comp.archive {
	case archiveNone:
		return activateBinary(src, filepath.Join(m.dir, comp.outputs[0].dest))

	case archiveGzip:
		dst := filepath.Join(m.dir, comp.outputs[0].dest+".tmp")
		if err := gunzipFile(src, dst); err != nil {
			_ = os.Remove(dst)
			return err
		}
		return activateBinary(dst, filepath.Join(m.dir, comp.outputs[0].dest))

	case archiveZip:
		for _, o := range comp.outputs {
			dst := filepath.Join(m.dir, o.dest+".tmp")
			if err := extractZipFile(src, o.member, dst); err != nil {
				_ = os.Remove(dst)
				return err
			}
			if err := activateBinary(dst, filepath.Join(m.dir, o.dest)); err != nil {
				return err
			}
		}
		return nil

	default:
		return fmt.Errorf("неизвестный тип архива: %d", comp.archive)
	}
}

// activateBinary выставляет бит исполнения и атомарно переносит файл на место.
func activateBinary(src, dst string) error {
	if err := os.Chmod(src, 0o755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	return replaceFile(src, dst)
}

// gunzipFile распаковывает gzip-файл src в dst.
func gunzipFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("открытие %q: %w", src, err)
	}
	defer in.Close()

	gz, err := gzip.NewReader(in)
	if err != nil {
		return fmt.Errorf("чтение gzip: %w", err)
	}
	defer gz.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("создание %q: %w", dst, err)
	}
	if _, err := io.Copy(out, gz); err != nil {
		_ = out.Close()
		return fmt.Errorf("распаковка gzip: %w", err)
	}
	return out.Close()
}

// extractZipFile извлекает из zip-архива src файл с именем name (сравнение по базовому
// имени) в путь dst.
func extractZipFile(src, name, dst string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("открытие архива: %w", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		if f.FileInfo().IsDir() || filepath.Base(f.Name) != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("чтение архива: %w", err)
		}
		defer rc.Close()

		out, err := os.Create(dst)
		if err != nil {
			return fmt.Errorf("создание %q: %w", dst, err)
		}
		if _, err := io.Copy(out, rc); err != nil {
			_ = out.Close()
			return fmt.Errorf("распаковка %q: %w", name, err)
		}
		if err := out.Close(); err != nil {
			return fmt.Errorf("закрытие %q: %w", dst, err)
		}
		return nil
	}

	return fmt.Errorf("в архиве нет файла %q", name)
}
