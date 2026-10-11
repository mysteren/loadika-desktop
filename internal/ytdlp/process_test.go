package ytdlp

import (
	"context"
	"errors"
	"testing"
)

func TestStopError(t *testing.T) {
	readClosed := errors.New("read |0: file already closed")
	exitOne := errors.New("exit status 1")

	tests := []struct {
		name    string
		ctxErr  error
		waitErr error
		scanErr error
		want    bool // ожидаем errStoppedByUs
	}{
		{
			// Процесс завершился сам, контекст не отменён — обычный результат.
			name: "без отмены",
		},
		{
			// Ложная отмена на Windows: ctx помечен отменённым, но yt-dlp докачал сам и вывод
			// прочитан целиком — это не наша остановка.
			name:   "ложная отмена",
			ctxErr: context.Canceled,
		},
		{
			// Наша пауза: os.Process.Kill завершает процесс кодом 1 (не STATUS_CONTROL_C_EXIT).
			name:    "убит с кодом 1",
			ctxErr:  context.Canceled,
			waitErr: exitOne,
			want:    true,
		},
		{
			// Пауза во время post-processing: watchCtx закрыл пайп, сканер вернул ошибку чтения.
			name:    "обрыв чтения пайпа",
			ctxErr:  context.Canceled,
			scanErr: readClosed,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stopError(tt.ctxErr, tt.waitErr, tt.scanErr)
			if tt.want {
				if !errors.Is(got, context.Canceled) {
					t.Fatalf("stopError = %v, want errStoppedByUs (context.Canceled)", got)
				}
				return
			}
			if got != nil {
				t.Fatalf("stopError = %v, want nil", got)
			}
		})
	}
}
