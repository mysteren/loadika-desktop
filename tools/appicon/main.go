// Команда appicon собирает квадратную иконку приложения из SVG-логотипа:
// акцентный фон со скруглёнными углами и знак по центру (по умолчанию — белый).
// Результат кладётся в build/appicon.png; далее `wails3 generate icons` делает из него
// .ico/.icns. Отдельный модуль: не тянет зависимости в основной модуль приложения.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// brandColor — исходный цвет заливки в логотипе, который мы перекрашиваем.
const brandColor = "#9b5aa8"

var viewBoxRe = regexp.MustCompile(`viewBox="([0-9.\- ]+)"`)

func main() {
	in := flag.String("in", "frontend/public/logo.svg", "входной SVG-логотип")
	out := flag.String("out", "build/appicon.png", "выходной PNG")
	size := flag.Int("size", 1024, "сторона иконки, px")
	bgHex := flag.String("bg", "#9b5aa8", "цвет фона (hex)")
	markHex := flag.String("mark", "#ffffff", "цвет знака (hex)")
	scale := flag.Float64("scale", 0.9, "доля высоты знака от стороны иконки")
	flag.Parse()

	if err := generate(*in, *out, *size, *bgHex, *markHex, *scale); err != nil {
		fmt.Fprintln(os.Stderr, "appicon:", err)
		os.Exit(1)
	}
}

func generate(in, out string, size int, bgHex, markHex string, scale float64) error {
	raw, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	svg := strings.ReplaceAll(string(raw), brandColor, markHex)

	vbW, vbH := parseViewBox(svg)

	icon, err := oksvg.ReadIconStream(strings.NewReader(svg))
	if err != nil {
		return fmt.Errorf("разбор SVG: %w", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	fillRoundedRect(img, float64(size)*0.22, mustColor(bgHex))

	// Вписываем знак по высоте с сохранением пропорций, по центру.
	th := float64(size) * scale
	tw := th * (vbW / vbH)
	icon.SetTarget((float64(size)-tw)/2, (float64(size)-th)/2, tw, th)

	scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
	raster := rasterx.NewDasher(size, size, scanner)
	icon.Draw(raster, 1.0)

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	return f.Close()
}

// parseViewBox достаёт размеры из атрибута viewBox (или значения по умолчанию).
func parseViewBox(svg string) (w, h float64) {
	const defW, defH = 807, 1280
	m := viewBoxRe.FindStringSubmatch(svg)
	if m == nil {
		return defW, defH
	}
	parts := strings.Fields(m[1])
	if len(parts) != 4 {
		return defW, defH
	}
	w, _ = strconv.ParseFloat(parts[2], 64)
	h, _ = strconv.ParseFloat(parts[3], 64)
	if w <= 0 || h <= 0 {
		return defW, defH
	}
	return w, h
}

func mustColor(hex string) color.RGBA {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return color.RGBA{R: 0x9b, G: 0x5a, B: 0xa8, A: 0xff}
	}
	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xff}
}

// fillRoundedRect заливает квадрат со скруглёнными углами; за пределами радиуса — прозрачно.
func fillRoundedRect(img *image.RGBA, radius float64, c color.RGBA) {
	size := float64(img.Bounds().Dx())
	for y := 0; y < int(size); y++ {
		for x := 0; x < int(size); x++ {
			if insideRounded(float64(x)+0.5, float64(y)+0.5, size, radius) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

// insideRounded сообщает, попадает ли точка в квадрат со скруглёнными углами.
func insideRounded(x, y, size, r float64) bool {
	cx, cy := x, y
	switch {
	case x < r && y < r:
		cx, cy = r, r
	case x > size-r && y < r:
		cx, cy = size-r, r
	case x < r && y > size-r:
		cx, cy = r, size-r
	case x > size-r && y > size-r:
		cx, cy = size-r, size-r
	default:
		return true
	}
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= r*r
}
