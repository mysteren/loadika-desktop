package ytdlp

import "strings"

// ErrorClass — класс ошибки yt-dlp для решения о смене прокси.
type ErrorClass int

const (
	// ErrorLogical — логическая ошибка (видео удалено/приватное, неверный формат): прокси не
	// виноват, переключаться не нужно.
	ErrorLogical ErrorClass = iota
	// ErrorNetwork — сетевой сбой/таймаут/ошибка прокси: имеет смысл попробовать другой прокси.
	ErrorNetwork
)

// networkMarkers — подстроки (в нижнем регистре), указывающие на сетевой сбой.
var networkMarkers = []string{
	"unable to connect",
	"connection refused",
	"connection reset",
	"connection timed out",
	"timed out",
	"timeout",
	"temporary failure in name resolution",
	"name or service not known",
	"no route to host",
	"network is unreachable",
	"failed to resolve",
	"getaddrinfo",
	"remote end closed connection",
	"read timed out",
	"eof occurred",
	"deadline exceeded",
	"превышено время ожидания",
	"ssl",
	"tls",
	"proxy",
	"gave up after",
	"giving up after",
	"unable to download",
	"http error 500",
	"http error 502",
	"http error 503",
	"http error 504",
	"bad gateway",
	"service unavailable",
}

// ClassifyError определяет класс ошибки по тексту вывода yt-dlp (stderr + сообщение). Функция
// чистая: если текст содержит сетевой маркер — ErrorNetwork, иначе ErrorLogical.
func ClassifyError(text string) ErrorClass {
	lower := strings.ToLower(text)
	for _, marker := range networkMarkers {
		if strings.Contains(lower, marker) {
			return ErrorNetwork
		}
	}
	return ErrorLogical
}
