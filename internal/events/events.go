// Package events содержит имена событий Go → Frontend и типы их данных.
// Имена синхронизированы с shared/config/events.ts.
package events

import "github.com/wailsapp/wails/v3/pkg/application"

// Имена событий.
const (
	DepProgressName  = "dep:progress"
	DepStateName     = "dep:state"
	TaskProgressName = "task:progress"
	TaskStateName    = "task:state"
	TaskLogName      = "task:log"
	ProxyHealthName  = "proxy:health"
)

// DepProgress — прогресс скачивания/установки зависимости.
type DepProgress struct {
	Name     string `json:"name"`
	Phase    string `json:"phase"` // "download" | "install"
	Received int64  `json:"received"`
	Total    int64  `json:"total"`
}

// DepState — смена состояния зависимости.
type DepState struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Version string `json:"version"`
}

// TaskProgress — ход загрузки задачи.
type TaskProgress struct {
	TaskID     string  `json:"taskId"`
	Downloaded int64   `json:"downloaded"`
	Total      int64   `json:"total"`
	Speed      float64 `json:"speed"`
	ETA        int64   `json:"eta"`
}

// TaskState — смена состояния/статуса задачи загрузки.
type TaskState struct {
	TaskID string `json:"taskId"`
	State  string `json:"state"`
	Error  string `json:"error"`
	Status string `json:"status"` // что происходит сейчас (например, «Попытка 2/3 (через …)»)
}

// TaskLog — строка журнала задачи загрузки (подробное информирование в клиенте).
type TaskLog struct {
	TaskID  string `json:"taskId"`
	Time    int64  `json:"time"`  // UnixMilli
	Level   string `json:"level"` // info | warn | error
	Message string `json:"message"`
}

// ProxyHealth — результат проверки прокси.
type ProxyHealth struct {
	ProxyID   string `json:"proxyId"`
	OK        bool   `json:"ok"`
	LatencyMs int64  `json:"latencyMs"`
}

// Регистрируем события, чтобы биндинги и проверка типов работали строго.
func init() {
	application.RegisterEvent[DepProgress](DepProgressName)
	application.RegisterEvent[DepState](DepStateName)
	application.RegisterEvent[TaskProgress](TaskProgressName)
	application.RegisterEvent[TaskState](TaskStateName)
	application.RegisterEvent[TaskLog](TaskLogName)
	application.RegisterEvent[ProxyHealth](ProxyHealthName)
}
