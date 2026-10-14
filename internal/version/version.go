// Package version хранит версию приложения.
package version

// Version — версия приложения.
//
// Может быть переопределена при сборке:
//
//	go build -ldflags "-X loadika-desktop/internal/version.Version=1.2.3"
var Version = "0.1.4"
