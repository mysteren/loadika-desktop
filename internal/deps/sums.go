package deps

import (
	"bufio"
	"fmt"
	"strings"
)

// parseSHA256 возвращает первый SHA-256 (64 hex-символа) из текста файла контрольных
// сумм. Подходит для per-asset файлов `*.sha256sum`, где хэш может быть как отдельно,
// так и в GNU-формате («хэш  имя-файла»).
func parseSHA256(text string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		for _, field := range strings.Fields(scanner.Text()) {
			candidate := strings.TrimPrefix(field, "*")
			if isSHA256Hex(candidate) {
				return strings.ToLower(candidate), nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("в файле нет контрольной суммы SHA-256")
}

// isSHA256Hex проверяет, что строка — 64 шестнадцатеричных символа.
func isSHA256Hex(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// parseSums находит SHA-256 для файла asset в тексте SHA2-256SUMS.
// Поддерживает GNU-формат: как с пробелом, так и со «*» перед именем файла.
func parseSums(text, asset string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if name == asset {
			return strings.ToLower(fields[0]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("в контрольных суммах нет файла %q", asset)
}
