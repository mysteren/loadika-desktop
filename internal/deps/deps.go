package deps

// Имена управляемых зависимостей.
const (
	NameYtDlp  = "yt-dlp"
	NameDeno   = "deno"
	NameFfmpeg = "ffmpeg"
)

// Статусы зависимости.
const (
	StatusMissing = "missing" // не найдена
	StatusSystem  = "system"  // используется системная
	StatusManaged = "managed" // установлена приложением
)

// Политики использования источника зависимости.
const (
	SourceAuto    = "auto"    // управляемая, если есть, иначе системная
	SourceSystem  = "system"  // только системная
	SourceManaged = "managed" // только управляемая
)

// IsValidSource сообщает, допустима ли политика источника зависимости.
func IsValidSource(v string) bool {
	switch v {
	case SourceAuto, SourceSystem, SourceManaged:
		return true
	default:
		return false
	}
}

// Dependency — сведения о зависимости для фронтенда.
type Dependency struct {
	Name            string `json:"name"`
	Status          string `json:"status"`
	Source          string `json:"source"` // политика использования: auto | system | managed
	Version         string `json:"version"`
	Path            string `json:"path"`
	UpdatedAt       string `json:"updatedAt"`       // RFC3339, пусто если неизвестно
	LatestVersion   string `json:"latestVersion"`   // заполняется при проверке обновлений
	UpdateAvailable bool   `json:"updateAvailable"` // true, если есть что обновить/установить
}

// Progress — прогресс скачивания зависимости.
type Progress struct {
	Phase    string `json:"phase"` // "download" | "install"
	Received int64  `json:"received"`
	Total    int64  `json:"total"`
}

// ProgressFunc получает обновления прогресса.
type ProgressFunc func(Progress)

// checksumsAsset — файл контрольных сумм в релизе yt-dlp.
const checksumsAsset = "SHA2-256SUMS"

// channelRepos сопоставляет канал обновления yt-dlp и GitHub-репозиторий.
var channelRepos = map[string]string{
	"stable":  "yt-dlp/yt-dlp",
	"nightly": "yt-dlp/yt-dlp-nightly-builds",
	"master":  "yt-dlp/yt-dlp-master-builds",
}
