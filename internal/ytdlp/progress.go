package ytdlp

import (
	"encoding/json"
	"strings"
)

// Progress — разобранный прогресс загрузки из строки вывода yt-dlp (%(progress)j).
// Числовые поля могут отсутствовать в JSON (значение неизвестно) — тогда они нулевые.
type Progress struct {
	Status     string  `json:"status"`     // "downloading" | "finished" | "error"
	Downloaded int64   `json:"downloaded"` // скачано байт
	Total      int64   `json:"total"`      // всего байт; 0 — неизвестно (в т.ч. оценка)
	Speed      float64 `json:"speed"`      // байт/с; 0 — неизвестно
	ETA        int64   `json:"eta"`        // секунд до завершения; 0 — неизвестно
	Filename   string  `json:"filename"`   // текущий файл
}

// progressJSON — сырой объект прогресса из JSON yt-dlp. Числа читаем как float64 через
// указатели: yt-dlp может отдать как целое (байты), так и дробное (скорость, оценка) значение,
// а также null, когда величина неизвестна.
type progressJSON struct {
	Status        string   `json:"status"`
	Downloaded    *float64 `json:"downloaded_bytes"`
	Total         *float64 `json:"total_bytes"`
	TotalEstimate *float64 `json:"total_bytes_estimate"`
	Speed         *float64 `json:"speed"`
	ETA           *float64 `json:"eta"`
	Filename      string   `json:"filename"`
}

// parseProgressLine разбирает строку вывода yt-dlp. Возвращает false, если строка не является
// строкой прогресса (нет префикса) или содержит некорректный JSON — такие строки игнорируются.
// Функция чистая; при разборе используется префикс из args.go.
func parseProgressLine(line string) (Progress, bool) {
	raw, ok := strings.CutPrefix(line, progressPrefix)
	if !ok {
		return Progress{}, false
	}

	var pj progressJSON
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &pj); err != nil {
		return Progress{}, false
	}
	return pj.progress(), true
}

// progress переводит сырой объект в Progress. Итоговый размер берём из total_bytes, а при его
// отсутствии — из оценочного total_bytes_estimate.
func (pj progressJSON) progress() Progress {
	total := pj.Total
	if total == nil {
		total = pj.TotalEstimate
	}

	return Progress{
		Status:     pj.Status,
		Downloaded: int64(deref(pj.Downloaded)),
		Total:      int64(deref(total)),
		Speed:      deref(pj.Speed),
		ETA:        int64(deref(pj.ETA)),
		Filename:   pj.Filename,
	}
}

// deref возвращает значение указателя или 0, если он nil.
func deref(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}
