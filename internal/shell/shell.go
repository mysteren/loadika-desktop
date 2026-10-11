// Package shell открывает файлы и каталоги средствами ОС: приложением по умолчанию и
// файловым менеджером. Не зависит от Wails и домена загрузок.
package shell

import "errors"

// ErrEmptyPath возвращается при пустом пути.
var ErrEmptyPath = errors.New("не задан путь")

// Open открывает путь (файл, каталог или URL) приложением по умолчанию.
func Open(path string) error {
	if path == "" {
		return ErrEmptyPath
	}
	return open(path)
}

// Reveal открывает файловый менеджер с выделением файла (где это поддерживает ОС).
// На Linux стандартного «выделения» нет — открываем содержащий каталог.
func Reveal(path string) error {
	if path == "" {
		return ErrEmptyPath
	}
	return reveal(path)
}
